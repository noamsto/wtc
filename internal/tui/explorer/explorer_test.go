package explorer

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/noamsto/wt/internal/git"
)

func newTestModel(t *testing.T) model {
	t.Helper()
	m := model{
		repoRoot: "/r",
		worktrees: []git.Worktree{
			{Branch: "feat-a", Path: "/r/.worktrees/feat-a", DetailsLoaded: true},
			{Branch: "feat-b", Path: "/r/.worktrees/feat-b", StaleReason: "merged", DetailsLoaded: true},
			{Branch: "fix-c", Path: "/r/.worktrees/fix-c", DetailsLoaded: true},
		},
		selected:  map[int]bool{},
		expanded:  map[int]bool{},
		diffCache: map[string]string{},
		preview:   viewport.New(),
		width:     120,
		height:    30,
		ready:     true,
		keys:      defaultKeys(),
		help:      help.New(),
	}
	m.rebuildItems()
	m.recomputeStaleCount()
	return m
}

// keyPress builds a v2 key message; single runes carry Text, named keys use Code.
func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "up":
		return tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		return tea.KeyPressMsg{Code: tea.KeyDown}
	case "backspace":
		return tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "ctrl+c":
		return tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	case "ctrl+j":
		return tea.KeyPressMsg{Code: 'j', Mod: tea.ModCtrl}
	case "ctrl+k":
		return tea.KeyPressMsg{Code: 'k', Mod: tea.ModCtrl}
	case "alt+j":
		return tea.KeyPressMsg{Code: 'j', Mod: tea.ModAlt}
	case "alt+k":
		return tea.KeyPressMsg{Code: 'k', Mod: tea.ModAlt}
	case "f1":
		return tea.KeyPressMsg{Code: tea.KeyF1}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}

func drive(m model, keys ...string) model {
	for _, k := range keys {
		u, _ := m.Update(keyPress(k))
		m = u.(model)
	}
	return m
}

func TestNavMovesCursor(t *testing.T) {
	m := newTestModel(t)
	if m.cursor != 0 {
		t.Fatalf("cursor starts at %d, want 0", m.cursor)
	}
	m = drive(m, "j")
	if m.cursor != 1 {
		t.Fatalf("j should move cursor to 1, got %d", m.cursor)
	}
	m = drive(m, "k")
	if m.cursor != 0 {
		t.Fatalf("k should move cursor to 0, got %d", m.cursor)
	}
}

func TestSlashEntersSearchAndTypeFilters(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "/")
	if !m.searching {
		t.Fatal("/ should enter search mode")
	}
	m = drive(m, "f", "e", "a")
	if m.query != "fea" {
		t.Fatalf("typing should build query, got %q", m.query)
	}
	m = drive(m, "backspace")
	if m.query != "fe" {
		t.Fatalf("backspace should trim query, got %q", m.query)
	}
}

func TestEscTwoStage(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "/", "f")
	// stage 1: esc in search mode blurs but keeps the query
	m = drive(m, "esc")
	if m.searching {
		t.Fatal("esc should leave search mode")
	}
	if m.query != "f" {
		t.Fatalf("esc-blur must keep the query, got %q", m.query)
	}
	// stage 2: esc in normal mode with a query clears it, does not quit
	u, cmd := m.Update(keyPress("esc"))
	m = u.(model)
	if m.query != "" {
		t.Fatalf("normal esc should clear the query, got %q", m.query)
	}
	if cmd != nil {
		t.Fatal("clearing the query must not quit")
	}
	// stage 3: esc with empty query quits
	_, cmd = m.Update(keyPress("esc"))
	if cmd == nil {
		t.Fatal("esc on an empty query should quit")
	}
}

func TestSpaceSelectsAndAStale(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "space")
	if len(m.selected) != 1 {
		t.Fatalf("space should select the cursor worktree, got %d selected", len(m.selected))
	}
	m = drive(m, "space")
	if len(m.selected) != 0 {
		t.Fatalf("space should toggle off, got %d selected", len(m.selected))
	}
	// 'a' with nothing selected selects all stale (feat-b is stale)
	m = drive(m, "a")
	if len(m.selected) != 1 {
		t.Fatalf("a should select the 1 stale worktree, got %d", len(m.selected))
	}
}

func TestDeleteOpensConfirm(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "d")
	if m.confirmMsg == "" {
		t.Fatal("d should open the delete confirm prompt")
	}
	// any non-y key cancels
	u, _ := m.Update(keyPress("n"))
	m = u.(model)
	if m.confirmMsg != "" {
		t.Fatal("a non-y key should dismiss the confirm prompt")
	}
}

func TestCtrlCQuits(t *testing.T) {
	m := newTestModel(t)
	_, cmd := m.Update(keyPress("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c should quit")
	}
}

func TestExpandToggles(t *testing.T) {
	m := newTestModel(t)
	wtIdx := m.items[m.cursor].wtIndex
	m = drive(m, "e")
	if !m.expanded[wtIdx] {
		t.Fatal("e should expand the cursor worktree")
	}
	m = drive(m, "e")
	if m.expanded[wtIdx] {
		t.Fatal("e again should collapse the cursor worktree")
	}
}

func TestForceDeleteOpensConfirm(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "D")
	if m.confirmMsg == "" {
		t.Fatal("D should open the delete confirm prompt")
	}
	if !m.confirmForce {
		t.Fatal("D should set confirmForce")
	}
}

func TestAClearsExistingSelection(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "space")
	if len(m.selected) != 1 {
		t.Fatalf("space should select the cursor worktree, got %d selected", len(m.selected))
	}
	m = drive(m, "a")
	if len(m.selected) != 0 {
		t.Fatalf("a should clear the existing selection, got %d selected", len(m.selected))
	}
}

func TestSearchCtrlCQuits(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "/")
	_, cmd := m.Update(keyPress("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c in search mode should quit")
	}
}

func TestCtrlJKMovesCursor(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "ctrl+j")
	if m.cursor != 1 {
		t.Fatalf("ctrl+j should move cursor down, got %d", m.cursor)
	}
	m = drive(m, "ctrl+k")
	if m.cursor != 0 {
		t.Fatalf("ctrl+k should move cursor up, got %d", m.cursor)
	}
}

func TestHelpOverlayTogglesAndListsKeys(t *testing.T) {
	m := newTestModel(t)
	m = drive(m, "?")
	if !m.showHelp {
		t.Fatal("? should open the help overlay")
	}
	out := m.renderFull()
	if !strings.Contains(out, "delete") || !strings.Contains(out, "scroll") {
		t.Fatalf("help overlay should document the keys: %q", out)
	}
	m = drive(m, "?")
	if m.showHelp {
		t.Fatal("? should close the help overlay")
	}
}

func TestFilterBarAlwaysVisibleAndFooterMentionsHelp(t *testing.T) {
	m := newTestModel(t) // not searching, empty query
	out := m.renderFull()
	if !strings.Contains(out, "/ to search") {
		t.Fatalf("blurred board should always show the search bar: %q", out)
	}
	if !strings.Contains(out, "?") {
		t.Fatalf("footer should advertise the ? help key: %q", out)
	}
}

func TestPreviewScroll(t *testing.T) {
	m := newTestModel(t)
	m.preview = viewport.New(viewport.WithWidth(40), viewport.WithHeight(5))
	m.preview.SetContent(strings.Repeat("line\n", 20))

	if m.preview.YOffset() != 0 {
		t.Fatalf("expected initial preview offset 0, got %d", m.preview.YOffset())
	}

	m = drive(m, "alt+j")
	if m.preview.YOffset() != 1 {
		t.Fatalf("alt+j should scroll the preview down to offset 1, got %d", m.preview.YOffset())
	}
	if m.cursor != 0 {
		t.Fatalf("alt+j should not move the list cursor, got %d", m.cursor)
	}

	m = drive(m, "alt+k")
	if m.preview.YOffset() != 0 {
		t.Fatalf("alt+k should scroll the preview back to offset 0, got %d", m.preview.YOffset())
	}
}
