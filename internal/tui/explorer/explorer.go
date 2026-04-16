package explorer

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"

	"github.com/noamsto/wt/internal/git"
	"github.com/noamsto/wt/internal/tmux"
	"github.com/noamsto/wt/internal/zoxide"
)

var (
	colorRed   = lipgloss.Color("#f38ba8")
	colorGreen = lipgloss.Color("#a6e3a1")
	colorBlue  = lipgloss.Color("#89b4fa")
	colorDim   = lipgloss.Color("#6c7086")
	colorText  = lipgloss.Color("#cdd6f4")
	colorPeach = lipgloss.Color("#fab387")
)

var (
	staleStyle     = lipgloss.NewStyle().Foreground(colorRed)
	selectedStyle  = lipgloss.NewStyle().Foreground(colorGreen)
	cursorStyle    = lipgloss.NewStyle().Foreground(colorBlue).Bold(true)
	dimStyle       = lipgloss.NewStyle().Foreground(colorDim)
	headerStyle    = lipgloss.NewStyle().Foreground(colorText).Bold(true)
	warnStyle      = lipgloss.NewStyle().Foreground(colorPeach)
	borderStyle    = lipgloss.NewStyle().Foreground(colorDim)
	statusBarStyle = lipgloss.NewStyle().Foreground(colorDim)
)

type listItem struct {
	wtIndex  int
	fileName string
}

func (item listItem) isFile() bool {
	return item.fileName != ""
}

type model struct {
	repoRoot     string
	worktrees    []git.Worktree
	tmuxClient   *tmux.Client
	zoxideClient *zoxide.Client
	items        []listItem
	cursor       int
	selected     map[int]bool
	expanded     map[int]bool
	diffCache    map[string]string
	query        string
	searching    bool
	preview      viewport.Model
	width        int
	height       int
	ready        bool
	confirmMsg   string
	confirmForce bool
	statusMsg    string
	staleCount   int
}

type detailsLoadedMsg struct {
	index int
}

type diffLoadedMsg struct {
	cacheKey string
	diff     string
}

// Run launches the interactive TUI explorer.
func Run(repoRoot string, worktrees []git.Worktree, tmuxClient *tmux.Client, zoxideClient *zoxide.Client) error {
	m := model{
		repoRoot:     repoRoot,
		worktrees:    worktrees,
		tmuxClient:   tmuxClient,
		zoxideClient: zoxideClient,
		selected:     make(map[int]bool),
		expanded:     make(map[int]bool),
		diffCache:    make(map[string]string),
		preview:      viewport.New(),
	}
	m.rebuildItems()
	m.recomputeStaleCount()

	p := tea.NewProgram(m)
	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	if len(m.items) > 0 {
		idx := m.items[0].wtIndex
		if !m.worktrees[idx].DetailsLoaded {
			return m.loadDetailsCmd(idx)
		}
	}
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updatePreviewSize()
		return m, nil
	case detailsLoadedMsg:
		m.worktrees[msg.index].DetailsLoaded = true
		m.rebuildItems()
		return m, nil
	case diffLoadedMsg:
		m.diffCache[msg.cacheKey] = msg.diff
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.confirmMsg != "" {
		switch msg.String() {
		case "y", "Y":
			m.executeDelete()
		default:
			m.statusMsg = "Cancelled."
		}
		m.confirmMsg = ""
		return m, nil
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}
	return m.handleNormalKey(msg)
}

func (m model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "enter":
		m.searching = false
		return m, nil
	case "backspace":
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, nil
	default:
		key := msg.Key()
		if key.Text != "" && key.Mod == 0 {
			m.query += key.Text
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, nil
	}
}

func (m model) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "q", "esc":
		if m.query != "" {
			m.query = ""
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, tea.Quit
	case "/":
		m.searching = true
		return m, nil
	case "up", "k":
		return m, m.moveCursor(-1)
	case "down", "j":
		return m, m.moveCursor(1)
	case "space", " ":
		if len(m.items) > 0 {
			item := m.items[m.cursor]
			if !item.isFile() {
				if m.selected[item.wtIndex] {
					delete(m.selected, item.wtIndex)
				} else {
					m.selected[item.wtIndex] = true
				}
			}
		}
		return m, nil
	case "a":
		for _, item := range m.items {
			if !item.isFile() && m.worktrees[item.wtIndex].IsStale() {
				m.selected[item.wtIndex] = true
			}
		}
		return m, nil
	case "e":
		if len(m.items) > 0 {
			wtIdx := m.items[m.cursor].wtIndex
			if m.expanded[wtIdx] {
				delete(m.expanded, wtIdx)
			} else {
				m.expanded[wtIdx] = true
			}
			m.rebuildItems()
		}
		return m, nil
	case "d":
		m.startDelete(false)
		return m, nil
	case "D":
		m.startDelete(true)
		return m, nil
	}
	return m, nil
}

func (m *model) moveCursor(delta int) tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	m.cursor += delta
	m.cursor = max(0, min(m.cursor, len(m.items)-1))
	return m.ensureLoaded()
}

func (m *model) ensureLoaded() tea.Cmd {
	if len(m.items) == 0 {
		return nil
	}
	item := m.items[m.cursor]
	wt := &m.worktrees[item.wtIndex]

	var cmds []tea.Cmd
	if !wt.DetailsLoaded {
		cmds = append(cmds, m.loadDetailsCmd(item.wtIndex))
	}
	if item.isFile() {
		key := diffCacheKey(wt.Path, item.fileName)
		if _, ok := m.diffCache[key]; !ok {
			cmds = append(cmds, m.loadDiffCmd(wt.Path, item.fileName))
		}
	}
	if len(cmds) == 0 {
		return nil
	}
	return tea.Batch(cmds...)
}

func (m *model) loadDetailsCmd(idx int) tea.Cmd {
	wt := &m.worktrees[idx]
	return func() tea.Msg {
		git.LoadDetails(wt)
		return detailsLoadedMsg{index: idx}
	}
}

func (m *model) loadDiffCmd(wtPath, fileName string) tea.Cmd {
	key := diffCacheKey(wtPath, fileName)
	return func() tea.Msg {
		diff := git.LoadFileDiff(wtPath, fileName)
		return diffLoadedMsg{cacheKey: key, diff: diff}
	}
}

func diffCacheKey(wtPath, fileName string) string {
	return wtPath + "\x00" + fileName
}

func (m *model) rebuildItems() {
	m.items = m.items[:0]
	q := strings.ToLower(m.query)
	for i, wt := range m.worktrees {
		if q != "" && !strings.Contains(strings.ToLower(wt.Branch), q) {
			continue
		}
		m.items = append(m.items, listItem{wtIndex: i})
		if m.expanded[i] && wt.DetailsLoaded && len(wt.DirtyFileNames) > 0 {
			for _, name := range wt.DirtyFileNames {
				m.items = append(m.items, listItem{wtIndex: i, fileName: name})
			}
		}
	}
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
}

func (m *model) startDelete(force bool) {
	targets := m.deleteTargets()
	if len(targets) == 0 {
		return
	}
	var names []string
	for _, idx := range targets {
		names = append(names, m.worktrees[idx].Branch)
	}
	verb := "Remove"
	if force {
		verb = "Force remove"
	}
	m.confirmMsg = fmt.Sprintf("%s %d worktree(s) [%s]? y/n", verb, len(targets), strings.Join(names, ", "))
	m.confirmForce = force
}

func (m *model) deleteTargets() []int {
	seen := make(map[int]bool)
	var targets []int
	for _, item := range m.items {
		if item.isFile() {
			continue
		}
		if m.selected[item.wtIndex] && !seen[item.wtIndex] {
			targets = append(targets, item.wtIndex)
			seen[item.wtIndex] = true
		}
	}
	if len(targets) == 0 && len(m.items) > 0 {
		targets = []int{m.items[m.cursor].wtIndex}
	}
	return targets
}

func (m *model) executeDelete() {
	targets := m.deleteTargets()
	if len(targets) == 0 {
		return
	}

	removedSet := make(map[int]bool, len(targets))
	var removed, failed int
	var lastErr string
	for _, idx := range targets {
		wt := m.worktrees[idx]
		err := git.RemoveWorktree(m.repoRoot, wt.Path, m.confirmForce)
		if err != nil {
			failed++
			lastErr = fmt.Sprintf("Error removing %s: %v", wt.Branch, err)
		} else {
			m.tmuxClient.KillWindow(m.repoRoot, wt.Path)
			m.zoxideClient.Remove(wt.Path)
			removedSet[idx] = true
			removed++
		}
	}

	if removed > 0 {
		var newWorktrees []git.Worktree
		indexMap := make(map[int]int)
		for i, wt := range m.worktrees {
			if !removedSet[i] {
				indexMap[i] = len(newWorktrees)
				newWorktrees = append(newWorktrees, wt)
			}
		}

		newSelected := make(map[int]bool)
		for oldIdx := range m.selected {
			if newIdx, ok := indexMap[oldIdx]; ok {
				newSelected[newIdx] = true
			}
		}
		newExpanded := make(map[int]bool)
		for oldIdx := range m.expanded {
			if newIdx, ok := indexMap[oldIdx]; ok {
				newExpanded[newIdx] = true
			}
		}

		m.worktrees = newWorktrees
		m.selected = newSelected
		m.expanded = newExpanded
		m.rebuildItems()
		m.recomputeStaleCount()
	}

	switch {
	case failed == 0:
		m.statusMsg = fmt.Sprintf("Removed %d worktree(s).", removed)
	case removed == 0:
		m.statusMsg = lastErr
	default:
		m.statusMsg = fmt.Sprintf("Removed %d, failed %d. %s", removed, failed, lastErr)
	}
}

func (m *model) recomputeStaleCount() {
	m.staleCount = 0
	for i := range m.worktrees {
		if m.worktrees[i].IsStale() {
			m.staleCount++
		}
	}
}

func (m *model) updatePreviewSize() {
	previewW, previewH := m.previewDimensions()
	m.preview = viewport.New(
		viewport.WithWidth(previewW),
		viewport.WithHeight(previewH),
	)
}

func (m *model) listWidth() int {
	w := m.width * 3 / 5
	return max(30, min(w, m.width-20))
}

func (m *model) previewDimensions() (int, int) {
	lw := m.listWidth()
	pw := max(10, m.width-lw-3)
	ph := max(3, m.height-6)
	return pw, ph
}

func (m model) View() tea.View {
	var content string
	if !m.ready {
		content = "Loading..."
	} else {
		content = m.renderFull()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m *model) renderFull() string {
	var b strings.Builder

	var searchLine string
	if m.searching {
		searchLine = cursorStyle.Render("/") + m.query + cursorStyle.Render("│")
	} else if m.query != "" {
		searchLine = dimStyle.Render("/") + m.query
	} else {
		searchLine = dimStyle.Render("/ to search")
	}
	b.WriteString(borderStyle.Render("── Search "+strings.Repeat("─", max(0, m.width-10))) + "\n")
	b.WriteString(searchLine + "\n")

	lw := m.listWidth()
	_, ph := m.previewDimensions()

	b.WriteString(padRight(headerStyle.Render(" Worktrees"), lw) + borderStyle.Render(" │ ") + headerStyle.Render(" Details") + "\n")

	listLines := m.renderListLines(lw, ph)

	previewContent := m.renderPreview()
	m.preview.SetContent(previewContent)
	previewRendered := m.preview.View()
	previewLines := strings.Split(previewRendered, "\n")

	for i := range ph {
		var left, right string
		if i < len(listLines) {
			left = listLines[i]
		}
		left = padRight(left, lw)
		if i < len(previewLines) {
			right = previewLines[i]
		}
		b.WriteString(left + borderStyle.Render(" │ ") + right + "\n")
	}

	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")
	if m.searching {
		b.WriteString(dimStyle.Render("type to filter  enter/esc accept  q clear+quit") + "\n")
	} else {
		b.WriteString(dimStyle.Render("j/k navigate  space select  a sel stale  e expand  d/D delete  / search  q quit") + "\n")
	}

	if m.confirmMsg != "" {
		b.WriteString(warnStyle.Render(m.confirmMsg))
	} else if m.statusMsg != "" {
		b.WriteString(statusBarStyle.Render(m.statusMsg))
	} else {
		b.WriteString(statusBarStyle.Render(fmt.Sprintf("%d worktrees, %d stale, %d selected",
			len(m.worktrees), m.staleCount, len(m.selected))))
	}

	return b.String()
}

func (m *model) renderListLines(width, height int) []string {
	lines := make([]string, 0, height)

	start := 0
	if m.cursor >= height {
		start = m.cursor - height + 1
	}

	for i, item := range m.items {
		if i < start {
			continue
		}
		if len(lines) >= height {
			break
		}

		if item.isFile() {
			lines = append(lines, m.renderFileLine(i, item, width))
		} else {
			lines = append(lines, m.renderWorktreeLine(i, item, width))
		}
	}

	for len(lines) < height {
		lines = append(lines, "")
	}
	return lines
}

func (m *model) renderWorktreeLine(i int, item listItem, width int) string {
	wt := m.worktrees[item.wtIndex]
	var line strings.Builder

	if i == m.cursor {
		line.WriteString(cursorStyle.Render("> "))
	} else {
		line.WriteString("  ")
	}

	if m.selected[item.wtIndex] {
		line.WriteString(selectedStyle.Render("✓"))
	} else {
		line.WriteString(" ")
	}

	line.WriteString(" ")

	switch {
	case wt.IsStale():
		line.WriteString(staleStyle.Render("●"))
	case wt.DetailsLoaded && len(wt.DirtyFileNames) > 0:
		line.WriteString(warnStyle.Render("+"))
	default:
		line.WriteString(" ")
	}

	line.WriteString(" ")
	if i == m.cursor {
		line.WriteString(cursorStyle.Render(wt.Branch))
	} else {
		line.WriteString(wt.Branch)
	}

	if wt.IsStale() {
		line.WriteString(" ")
		line.WriteString(staleStyle.Render("[" + wt.StaleReason + "]"))
	}

	if m.expanded[item.wtIndex] && wt.DetailsLoaded && len(wt.DirtyFileNames) > 0 {
		line.WriteString(" ")
		line.WriteString(dimStyle.Render(fmt.Sprintf("(%d)", len(wt.DirtyFileNames))))
	}

	return truncateToWidth(line.String(), width)
}

func (m *model) renderFileLine(i int, item listItem, width int) string {
	var line strings.Builder

	if i == m.cursor {
		line.WriteString(cursorStyle.Render("> "))
	} else {
		line.WriteString("  ")
	}

	// Blank columns to align with worktree: selected(1) + space(1) + stale(1) + space(1)
	line.WriteString("    ")

	isLast := i+1 >= len(m.items) ||
		!m.items[i+1].isFile() ||
		m.items[i+1].wtIndex != item.wtIndex

	if isLast {
		line.WriteString(dimStyle.Render("╰─ "))
	} else {
		line.WriteString(dimStyle.Render("├─ "))
	}

	if i == m.cursor {
		line.WriteString(cursorStyle.Render(item.fileName))
	} else {
		line.WriteString(warnStyle.Render(item.fileName))
	}

	return truncateToWidth(line.String(), width)
}

func (m *model) renderPreview() string {
	if len(m.items) == 0 {
		return dimStyle.Render("No worktrees to display.")
	}

	item := m.items[m.cursor]
	if item.isFile() {
		return m.renderFileDiffPreview(item)
	}
	return m.renderWorktreePreview(item)
}

func (m *model) renderWorktreePreview(item listItem) string {
	wt := m.worktrees[item.wtIndex]
	pw, _ := m.previewDimensions()
	sep := dimStyle.Render(strings.Repeat("─", max(0, pw-1)))

	var b strings.Builder

	b.WriteString(headerStyle.Render("  "+wt.Branch) + "\n")
	b.WriteString(dimStyle.Render("  "+wt.Path) + "\n")

	if wt.IsStale() {
		b.WriteString(staleStyle.Render("  ● "+wt.StaleReason) + "\n")
	}

	if !wt.DetailsLoaded {
		b.WriteString("\n" + dimStyle.Render("  Loading..."))
		return b.String()
	}

	b.WriteString(sep + "\n")

	dirtyLabel := dimStyle.Render("  clean")
	if len(wt.DirtyFileNames) > 0 {
		hint := " [e expand]"
		if m.expanded[item.wtIndex] {
			hint = " [e collapse]"
		}
		dirtyLabel = warnStyle.Render(fmt.Sprintf("  %d dirty file(s)", len(wt.DirtyFileNames))) +
			dimStyle.Render(hint)
	}
	b.WriteString(dirtyLabel + "\n")

	unpushedLabel := dimStyle.Render("  pushed")
	if len(wt.UnpushedLog) > 0 {
		unpushedLabel = warnStyle.Render(fmt.Sprintf("  %d unpushed commit(s)", len(wt.UnpushedLog)))
	}
	b.WriteString(unpushedLabel + "\n")

	if len(wt.UnpushedLog) > 0 {
		b.WriteString(sep + "\n")
		b.WriteString(headerStyle.Render("  Unpushed") + "\n")
		for _, line := range wt.UnpushedLog {
			b.WriteString(dimStyle.Render("  ") + line + "\n")
		}
	}

	if wt.LastCommit != "" {
		b.WriteString(sep + "\n")
		b.WriteString(headerStyle.Render("  Last commit") + "\n")
		b.WriteString(dimStyle.Render("  ") + wt.LastCommit + "\n")
	}

	return b.String()
}

func (m *model) renderFileDiffPreview(item listItem) string {
	wt := m.worktrees[item.wtIndex]
	pw, _ := m.previewDimensions()
	sep := dimStyle.Render(strings.Repeat("─", max(0, pw-1)))

	var b strings.Builder

	b.WriteString(headerStyle.Render("  "+item.fileName) + "\n")
	b.WriteString(dimStyle.Render("  "+wt.Branch+" · "+wt.Path) + "\n")
	b.WriteString(sep + "\n")

	key := diffCacheKey(wt.Path, item.fileName)
	diff, ok := m.diffCache[key]
	if !ok {
		b.WriteString(dimStyle.Render("  Loading..."))
		return b.String()
	}

	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "---"):
			b.WriteString(dimStyle.Render("  "+line) + "\n")
		case strings.HasPrefix(line, "+"):
			b.WriteString(selectedStyle.Render("  "+line) + "\n")
		case strings.HasPrefix(line, "-"):
			b.WriteString(staleStyle.Render("  "+line) + "\n")
		case strings.HasPrefix(line, "@@"):
			b.WriteString(cursorStyle.Render("  "+line) + "\n")
		default:
			b.WriteString(dimStyle.Render("  "+line) + "\n")
		}
	}

	return b.String()
}

func padRight(s string, width int) string {
	w := lipgloss.Width(s)
	if w >= width {
		return s
	}
	return s + strings.Repeat(" ", width-w)
}

func truncateToWidth(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	var result []rune
	visW := 0
	inEscape := false
	for _, r := range runes {
		if r == '\x1b' {
			inEscape = true
			result = append(result, r)
			continue
		}
		if inEscape {
			result = append(result, r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		rw := lipgloss.Width(string(r))
		if visW+rw > width {
			break
		}
		visW += rw
		result = append(result, r)
	}
	return string(result)
}
