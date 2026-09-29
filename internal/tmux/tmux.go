package tmux

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Client manages tmux operations. All methods are no-ops if tmux is unavailable.
type Client struct {
	active bool
}

// New creates a Client. If active is false, all operations are no-ops.
func New(active bool) *Client {
	return &Client{active: active}
}

// FindWindowByWorktree returns the window index for a worktree path, or empty string.
func (c *Client) FindWindowByWorktree(session, worktreePath string) string {
	if !c.active {
		return ""
	}

	out, err := exec.Command("tmux", "list-windows", "-t", session, //nolint:gosec // fixed tmux binary; session is a discrete argv element
		"-F", "#{window_index}\t#{@worktree}\t#{pane_current_path}").Output()
	if err != nil {
		return ""
	}

	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		if parts[1] == worktreePath || parts[2] == worktreePath {
			return parts[0]
		}
	}
	return ""
}

// KillWindow kills the tmux window associated with a worktree path.
func (c *Client) KillWindow(repoRoot, worktreePath string) {
	if !c.active {
		return
	}
	sessionName := filepath.Base(repoRoot)
	if !hasSession(sessionName) {
		return
	}
	windowIdx := c.FindWindowByWorktree(sessionName, worktreePath)
	if windowIdx != "" {
		_ = exec.Command("tmux", "kill-window", "-t", sessionName+":"+windowIdx).Run() //nolint:gosec // fixed tmux binary; target is a discrete argv element
	}
}

func hasSession(name string) bool {
	return exec.Command("tmux", "has-session", "-t", name).Run() == nil //nolint:gosec // fixed tmux binary; name is a discrete argv element
}
