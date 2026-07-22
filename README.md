# wtc

Git worktree cleanup tool. Finds and removes stale worktrees.

![wtc demo](demo/wtc-demo.gif)

Optional integrations are auto-detected and silently no-op when unavailable:

- **tmux** — kills windows for removed worktrees; `wtc -i` auto-opens as a floating popup when `$TMUX` is set
- **gh** — squash-merge detection for stale branch cleanup

## Install

### Nix flake

```nix
# flake input
inputs.wtc.url = "github:noamsto/wtc";

# use the package
wtc.packages.${system}.default
```

#### Binary cache

```nix
nix.settings = {
  substituters = ["https://wtc.cachix.org"];
  trusted-public-keys = ["wtc.cachix.org-1:rXr2jpSoAtqezDz8xK/gPMjH2Rgyda0zOErfk3N5WnI="];
};
```

### Go

```bash
go install github.com/noamsto/wt/cmd/wtc@latest
```

## Usage

```
wtc               Find and remove stale worktrees (interactive prompt)
wtc -i            Interactive TUI explorer
wtc -y            Skip confirmation prompts
wtc -q            Quiet mode
wtc -h            Show help
```

### Interactive explorer

`wtc -i` opens a TUI for inspecting and cleaning up worktrees:

```
j/k  navigate       space  select        a  select all stale
e    expand dirty    d      delete        D  force delete
/    search          q      quit
```

The preview pane shows branch details, dirty files, unpushed commits, and last commit info. Press `e` to expand the list of dirty files for a worktree.

### Tmux popup

When `wtc -i` runs inside a tmux session, it re-execs itself as a
`display-popup` so the TUI floats over your workspace instead of taking over
the current pane.

```
WTC_NO_POPUP=1      Keep the TUI inline (opt out)
WTC_POPUP_WIDTH     Popup width  (default 90%; any `display-popup -w` value)
WTC_POPUP_HEIGHT    Popup height (default 90%; any `display-popup -h` value)
```

### Stale detection

`wtc` identifies worktrees whose branches are:

- Merged into the default branch
- Squash-merged via GitHub PR (requires `gh`)
- Deleted on the remote

## Worktree layout

`wtc` discovers worktrees from `git worktree list`, so any layout works — nested (`<repo>/.worktrees/<branch>`), sibling (`<repo>-worktrees/<branch>`), or a shared external root (`<root>/.worktrees/<org>/<repo>/<branch>`), including [worktrunk](https://worktrunk.dev/)'s.

## License

MIT
