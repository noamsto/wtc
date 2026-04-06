package git

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// RepoRoot finds the true repository root (works from worktrees too).
func RepoRoot() (string, error) {
	commonDir, err := exec.Command("git", "rev-parse", "--git-common-dir").Output()
	if err != nil {
		return "", fmt.Errorf("not in a git repository")
	}
	cd := strings.TrimSpace(string(commonDir))

	if cd == ".git" {
		out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
		if err != nil {
			return "", fmt.Errorf("git show-toplevel: %w", err)
		}
		return strings.TrimSpace(string(out)), nil
	}

	abs, err := filepath.Abs(cd)
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(abs, "/.git"), nil
}

// DefaultBranch detects whether the repo uses "main" or "master".
func DefaultBranch(repoRoot string) (string, error) {
	for _, branch := range []string{"main", "master"} {
		if exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil {
			return branch, nil
		}
	}
	return "", fmt.Errorf("could not find main or master branch")
}
