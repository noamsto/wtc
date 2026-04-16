package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Worktree represents a git worktree with its metadata.
type Worktree struct {
	Branch         string
	Path           string
	StaleReason    string
	DirtyFileNames []string
	UnpushedLog    []string
	LastCommit     string
	DetailsLoaded  bool
}

// IsStale returns true if the worktree has been marked stale.
func (w *Worktree) IsStale() bool {
	return w.StaleReason != ""
}

// ListWorktrees returns all worktrees excluding the repo root and default branch.
func ListWorktrees(repoRoot, defaultBranch string) ([]Worktree, error) {
	out, err := exec.Command("git", "-C", repoRoot, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return nil, fmt.Errorf("git worktree list: %w", err)
	}
	return ParseWorktreesPorcelain(string(out), repoRoot, defaultBranch), nil
}

// ParseWorktreesPorcelain parses `git worktree list --porcelain` output.
func ParseWorktreesPorcelain(output, repoRoot, defaultBranch string) []Worktree {
	var worktrees []Worktree
	var currentPath string

	for line := range strings.SplitSeq(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			currentPath = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			branch := strings.TrimPrefix(line, "branch refs/heads/")
			cleanPath := filepath.Clean(currentPath)
			cleanRoot := filepath.Clean(repoRoot)
			if cleanPath != cleanRoot && branch != defaultBranch {
				worktrees = append(worktrees, Worktree{
					Branch: branch,
					Path:   currentPath,
				})
			}
			currentPath = ""
		}
	}
	return worktrees
}

// RemoveWorktree removes a worktree by path. If force is true, uses --force.
func RemoveWorktree(repoRoot, path string, force bool) error {
	args := []string{"-C", repoRoot, "worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	out, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// PruneWorktrees removes stale worktree references.
func PruneWorktrees(repoRoot string) error {
	return exec.Command("git", "-C", repoRoot, "worktree", "prune").Run()
}

// FetchPrune runs git fetch --prune with a 30s timeout.
func FetchPrune(repoRoot string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", repoRoot, "fetch", "--prune")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

