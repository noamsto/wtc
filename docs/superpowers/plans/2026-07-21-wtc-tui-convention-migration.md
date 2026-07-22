# wtc TUI Convention Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bring wtc's explorer into conformance with the TUI Interaction Convention as a *board* archetype — `key.Binding`/`help.Model` machinery, a `?`/`F1` keymap overlay, and `ctrl+j/k` selection nav — while keeping its `/`-gated filter and single-key actions.

**Architecture:** wtc's TUI is a single Bubble Tea v2 `model` in `internal/tui/explorer/explorer.go` with raw `switch msg.String()` handlers (`handleKey`→`handleSearchKey`/`handleNormalKey`). The file is currently **untested**, so Task 1 adds a characterization safety net *before* the behavior-preserving refactor in Task 2. The always-visible filter bar (`renderFull`) and two-stage `esc` already conform to the spine — this plan preserves them and adds tests, rather than rebuilding them.

**Tech Stack:** Go 1.25, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2` (`viewport`, and newly `key` + `help` — already in the module, just unimported), `charm.land/lipgloss/v2`.

## Global Constraints

- Bubble Tea **v2** API (`tea.KeyPressMsg`, `msg.String()` / `msg.Key()`). Imports use the `charm.land/...` vanity paths already in this file.
- wtc is a **board**: do NOT gate the filter differently (`/` stays the entry) and do NOT change the semantics of the bare-letter actions `d`/`D` (delete/force-delete), `space` (multi-select), `a` (select-stale/clear), `e` (expand), `enter`/`/` etc. Only the *dispatch mechanism* and additive keys change.
- Preserve the existing **always-visible search bar** (`renderFull`, the "── Search ──" block) and **two-stage `esc`** (search-mode esc/enter blurs keeping the query; normal-mode esc/q clears the query if present, else quits).
- Module path is `github.com/noamsto/wt` (dir is `wtc`).
- Out of scope: the `huh`-based `internal/tui/prompt` fallback; the shared `tuikit` extraction (deferred to second use).
- Convention spec of record: `noamsto/prdash` PR #46 (`KEYMAP.md`, `docs/superpowers/specs/2026-07-20-tui-interaction-convention-design.md`).

## File Structure

- `internal/tui/explorer/explorer.go` (modify) — add `keyMap` + `help.Model`; replace raw switches with `key.Matches`; add `showHelp` state + overlay; extend nav bindings.
- `internal/tui/explorer/explorer_test.go` (new) — characterization + new-behavior tests, in `package explorer`.

---

### Task 1: Characterization tests (safety net before refactor)

**Files:**
- Create: `internal/tui/explorer/explorer_test.go`

**Interfaces:**
- Consumes: unexported `model` (in-package), `model.Update`, `model.rebuildItems()`, `model.recomputeStaleCount()`, `git.Worktree{Branch, Path, StaleReason, DirtyFileNames, UnpushedLog, LastCommit, DetailsLoaded}` (`IsStale() == StaleReason != ""`).
- Produces: `newTestModel(t) model` and `keyPress(s string) tea.KeyPressMsg` helpers for later tasks.

- [ ] **Step 1: Write the test helpers + characterization tests**

```go
package explorer

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"

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
```

- [ ] **Step 2: Run — expect all PASS against current behavior (characterization)**

Run: `go test ./internal/tui/explorer/ -v`
Expected: PASS. If any fails, the assertion misreads current behavior — fix the *test* to match what the code does today (do not change explorer.go in this task).

- [ ] **Step 3: Commit**

```bash
git add internal/tui/explorer/explorer_test.go
git commit -m "test(tui): characterization tests for explorer key behavior"
```

---

### Task 2: Migrate raw switches to key.Binding + help.Model

**Files:**
- Modify: `internal/tui/explorer/explorer.go` — add imports, `keyMap` type, `defaultKeys()`, a `keys keyMap` + `help help.Model` field on `model`, and replace the `switch msg.String()` bodies in `handleSearchKey` (142) and `handleNormalKey` (167) with `key.Matches`.
- Test: guarded by Task 1's tests (no behavior change).

**Interfaces:**
- Produces: `keyMap` struct with exported-to-package bindings `Up, Down, PreviewUp, PreviewDown, Search, Select, Stale, Expand, Delete, ForceDelete, Help, Quit, ForceQuit, Accept, Back` (`key.Binding`); `defaultKeys() keyMap`; `keyMap.ShortHelp() []key.Binding`; `keyMap.FullHelp() [][]key.Binding`.

- [ ] **Step 1: Run Task 1 tests to confirm the safety net is green before refactoring**

Run: `go test ./internal/tui/explorer/`
Expected: PASS.

- [ ] **Step 2: Add imports and the keyMap** (top of explorer.go, after the existing charm imports):

```go
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
```

Add the type and constructor near the `model` definition:

```go
type keyMap struct {
	Up, Down                 key.Binding
	PreviewUp, PreviewDown   key.Binding
	Search                   key.Binding
	Select, Stale, Expand    key.Binding
	Delete, ForceDelete      key.Binding
	Help, Quit, ForceQuit    key.Binding
	Accept, Back             key.Binding
}

func defaultKeys() keyMap {
	return keyMap{
		Up:          key.NewBinding(key.WithKeys("up", "k", "ctrl+k"), key.WithHelp("↑/k", "up")),
		Down:        key.NewBinding(key.WithKeys("down", "j", "ctrl+j"), key.WithHelp("↓/j", "down")),
		PreviewUp:   key.NewBinding(key.WithKeys("alt+k"), key.WithHelp("alt+k", "scroll up")),
		PreviewDown: key.NewBinding(key.WithKeys("alt+j"), key.WithHelp("alt+j", "scroll down")),
		Search:      key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search")),
		Select:      key.NewBinding(key.WithKeys("space", " "), key.WithHelp("space", "select")),
		Stale:       key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "toggle stale")),
		Expand:      key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "expand")),
		Delete:      key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "delete")),
		ForceDelete: key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "force delete")),
		Help:        key.NewBinding(key.WithKeys("?", "f1"), key.WithHelp("?/F1", "keys")),
		Quit:        key.NewBinding(key.WithKeys("q", "esc"), key.WithHelp("q", "quit")),
		ForceQuit:   key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("ctrl+c", "quit")),
		Accept:      key.NewBinding(key.WithKeys("enter", "esc"), key.WithHelp("enter/esc", "accept")),
		Back:        key.NewBinding(key.WithKeys("backspace"), key.WithHelp("⌫", "delete char")),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Search, k.Select, k.Delete, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PreviewUp, k.PreviewDown},
		{k.Search, k.Select, k.Stale, k.Expand},
		{k.Delete, k.ForceDelete},
		{k.Help, k.Quit, k.ForceQuit},
	}
}
```

Add to the `model` struct (Task 1's `newTestModel` builds the struct literal directly, so it must keep compiling — the two new fields are zero-valued there; set them in `Run` and default them lazily, see Step 4):

```go
	keys keyMap
	help help.Model
```

- [ ] **Step 3: Initialize in `Run`** (in the `model{...}` literal at line 76):

```go
		keys: defaultKeys(),
		help: help.New(),
```

- [ ] **Step 4: Replace the search-mode switch** (`handleSearchKey`, 142-165) with `key.Matches`:

```go
func (m model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.ForceQuit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Accept): // enter/esc: blur, keep the query
		m.searching = false
		return m, nil
	case key.Matches(msg, m.keys.Back):
		if len(m.query) > 0 {
			m.query = m.query[:len(m.query)-1]
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, nil
	default:
		k := msg.Key()
		if k.Text != "" && k.Mod == 0 {
			m.query += k.Text
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, nil
	}
}
```

> Note: `Accept` and `Quit` both bind `esc`. That's fine — `handleSearchKey` only runs while `m.searching`, and it checks `Accept` (blur). `Quit`/`Back` semantics live in `handleNormalKey`. Never let `esc` fall to the default rune branch (it has no `.Text`, so it wouldn't append anyway).

- [ ] **Step 5: Replace the normal-mode switch** (`handleNormalKey`, 167-232) with `key.Matches`, preserving exact behavior:

```go
func (m model) handleNormalKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.ForceQuit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Quit): // q/esc: clear query if present, else quit
		if m.query != "" {
			m.query = ""
			m.rebuildItems()
			return m, m.ensureLoaded()
		}
		return m, tea.Quit
	case key.Matches(msg, m.keys.Search):
		m.searching = true
		return m, nil
	case key.Matches(msg, m.keys.Up):
		return m, m.moveCursor(-1)
	case key.Matches(msg, m.keys.Down):
		return m, m.moveCursor(1)
	case key.Matches(msg, m.keys.PreviewUp):
		m.preview.ScrollUp(1)
		return m, nil
	case key.Matches(msg, m.keys.PreviewDown):
		m.preview.ScrollDown(1)
		return m, nil
	case key.Matches(msg, m.keys.Select):
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
	case key.Matches(msg, m.keys.Stale):
		if len(m.selected) > 0 {
			m.selected = make(map[int]bool)
		} else {
			for _, item := range m.items {
				if !item.isFile() && m.worktrees[item.wtIndex].IsStale() {
					m.selected[item.wtIndex] = true
				}
			}
		}
		return m, nil
	case key.Matches(msg, m.keys.Expand):
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
	case key.Matches(msg, m.keys.Delete):
		m.startDelete(false)
		return m, nil
	case key.Matches(msg, m.keys.ForceDelete):
		m.startDelete(true)
		return m, nil
	}
	return m, nil
}
```

> `Quit` binds `q`+`esc` and `Down` binds `j`+`ctrl+j` etc.; `key.Matches` compares against the binding's key set, so ordering only matters where one key appears in two bindings — here `esc` is in both `Quit` and `Accept`, but `Accept` is only consulted in search mode, so normal mode's `esc`→`Quit` is unambiguous.

- [ ] **Step 6: Run the safety-net tests — behavior must be unchanged**

Run: `go test ./internal/tui/explorer/ -v`
Expected: PASS (all Task 1 tests still green). Then `go build ./... && go vet ./...`.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/explorer/explorer.go
git commit -m "refactor(tui): migrate explorer keys to key.Binding + help.Model"
```

---

### Task 3: `ctrl+j/k` selection nav (verify via test)

The bindings from Task 2 already include `ctrl+j`/`ctrl+k` in `Down`/`Up`. This task adds the guarding test.

**Files:**
- Test: `internal/tui/explorer/explorer_test.go`

- [ ] **Step 1: Write the failing test** (fails only if Task 2's bindings omit `ctrl+j/k`):

```go
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
```

- [ ] **Step 2: Run to verify it passes** (Task 2 already wired the keys):

Run: `go test ./internal/tui/explorer/ -run TestCtrlJKMovesCursor -v`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/tui/explorer/explorer_test.go
git commit -m "test(tui): assert ctrl+j/k move the selection"
```

---

### Task 4: `?`/`F1` keymap overlay

**Files:**
- Modify: `internal/tui/explorer/explorer.go` — `showHelp bool` field; handle `Help` in `handleNormalKey`; a `showHelp` intercept in `handleKey`; render the overlay in `View`/`renderFull`.
- Test: `internal/tui/explorer/explorer_test.go`

**Interfaces:**
- Consumes: `m.help.View(m.keys)` (the `help.Model` renders `FullHelp` when `help.ShowAll` is true).

- [ ] **Step 1: Write the failing test**

```go
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
```

(Add `"strings"` to the test imports.)

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/tui/explorer/ -run TestHelpOverlayTogglesAndListsKeys -v`
Expected: FAIL (`showHelp` undefined / `?` unhandled).

- [ ] **Step 3: Add the field + handling.** Add `showHelp bool` to `model`. In `handleKey` (124), intercept before the search/normal split:

```go
	if m.showHelp {
		m.showHelp = false // any key closes the overlay
		return m, nil
	}
```

In `handleNormalKey`, add a case (before `Search`):

```go
	case key.Matches(msg, m.keys.Help):
		m.showHelp = true
		return m, nil
```

- [ ] **Step 4: Render the overlay.** In `renderFull` (444), at the top, short-circuit when `showHelp`:

```go
	if m.showHelp {
		m.help.ShowAll = true
		return headerStyle.Render(" Keys") + "\n\n" + m.help.View(m.keys)
	}
```

- [ ] **Step 5: Run to verify it passes**

Run: `go test ./internal/tui/explorer/ -run TestHelpOverlayTogglesAndListsKeys -v`
Expected: PASS. Then full package + build: `go test ./internal/tui/explorer/ && go build ./...`.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/explorer/explorer.go internal/tui/explorer/explorer_test.go
git commit -m "feat(tui): ?/F1 keymap overlay (help.Model)"
```

---

### Task 5: Spine conformance guards + footer copy

**Files:**
- Modify: `internal/tui/explorer/explorer.go` — the normal-mode footer hint string in `renderFull` (486).
- Test: `internal/tui/explorer/explorer_test.go`

- [ ] **Step 1: Write the failing test**

```go
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
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/tui/explorer/ -run TestFilterBarAlwaysVisibleAndFooterMentionsHelp -v`
Expected: FAIL on the `?` assertion (current footer omits it).

- [ ] **Step 3: Update the normal-mode footer copy** (renderFull, 486) to advertise the new keys:

```go
		b.WriteString(dimStyle.Render("j/k move  ctrl+j/k too  alt+j/k scroll  space sel  a stale  e expand  d/D del  / search  ? keys  q quit") + "\n")
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/tui/explorer/ -run TestFilterBarAlwaysVisibleAndFooterMentionsHelp -v`
Expected: PASS. Then the whole package + build + vet: `go test ./internal/tui/explorer/ && go build ./... && go vet ./...`.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/explorer/explorer.go internal/tui/explorer/explorer_test.go
git commit -m "docs(tui): footer advertises ? help and ctrl+j/k; guard always-visible bar"
```

---

## Self-review notes

- **Spec coverage (wtc row):** always-visible bar — *already present*, now guarded (T5) ✓; two-stage esc — *already present*, now guarded (T1) ✓; keep `/` + `d/D/space/a/e` — preserved through the key.Binding migration (T2) ✓; `ctrl+j/k` nav (T2 bindings + T3 test) ✓; `?`/`F1` overlay (T4) ✓; `key.Binding`+`help.Model` migration (T2) ✓.
- **Untested-file risk:** Task 1 establishes the safety net before Task 2's refactor; every later task runs the package suite.
- **Out of scope (deferred):** the `huh` `prompt.Filter` fallback; the shared `tuikit` package.
- **Compile-coupling note:** Task 1's `newTestModel` builds the `model{...}` literal directly, so Task 2 adding `keys`/`help` fields leaves them zero-valued in tests. `help.Model`'s zero value renders; but the tests that call `renderFull`/help (T4, T5) go through `newTestModel`, so **T4 Step 3 must also set `m.keys`/`m.help` in `newTestModel`** (add `keys: defaultKeys(), help: help.New()` there) — fold that one-line test-helper update into Task 4.
