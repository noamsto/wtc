# TODO

## Migration: Replace wt with Worktrunk

Worktrunk (`github:max-sixty/worktrunk`) replaces the core worktree management
(create/switch/list/remove/merge). The lazytmux bash `wt` and Go `wt` are retired.

### What Worktrunk provides
- Rich `wt list` with status, divergence, CI, LLM summaries
- Hooks system (post-switch, post-start, post-remove) for tmux window lifecycle
- Claude Code plugin with activity tracking (robot/speech markers)
- Shell integration with auto-cd (no more `cd "$(wt -yqn ...)"`)
- LLM commit messages, build cache sharing, PR checkout
- Fish completions built-in

### What needs a standalone tool: stale worktree cleanup

Extract the cleanup TUI into a standalone binary (e.g. `wt-clean` or `git-worktree-clean`):
- 3-strategy stale detection (merged + remote deleted + GH squash-merged)
- Interactive Bubble Tea TUI with search, select, bulk delete, preview
- tmux window kill on remove

#### TUI improvements to make during extraction
- **Checkmark spacing too dense** — add space between `✓` and `●` columns
- **Expand mode for dirty files** — `e`/`enter` toggles showing actual dirty filenames
  - Store `DirtyFileNames []string` in Worktree (not just count)
  - Load lazily in `LoadDetails`
  - Render in preview pane when expanded
