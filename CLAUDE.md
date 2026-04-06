# CLAUDE.md

## Project Overview

`wt` is a standalone git worktree manager. Single Go binary, no required deps beyond `git`. Optional auto-detected integrations: `tmux` (window management), `zoxide` (frecency tracking), `gh` (squash-merge detection).

## Build and Test

```bash
go build ./cmd/wt          # Build binary
go test ./...              # Run all tests
nix build .                # Build via Nix
nix flake check            # Run flake checks
```

## Architecture

- `cmd/wt/main.go` — CLI entry point, flag parsing, command dispatch
- `internal/git/` — all git operations via exec.Command (no go-git)
- `internal/tmux/` — tmux window/session management (no-op when unavailable)
- `internal/zoxide/` — zoxide add/remove (no-op when unavailable)
- `internal/gh/` — GitHub PR squash-merge detection (no-op when unavailable)
- `internal/runtime/` — detect which optional tools are available
- `internal/tui/explorer/` — Bubble Tea TUI for interactive worktree cleanup
- `internal/tui/prompt/` — confirm/filter prompts using huh v2
- `internal/cmd/` — command implementations

## Conventions

- Shell out to git/tmux/zoxide/gh via exec.Command
- Optional integrations auto-detect at startup and silently no-op when unavailable
- Tests use temp git repos — never touch the real filesystem
