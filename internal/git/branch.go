package git

import (
	"os/exec"
	"strings"
)

// ParseBranchList parses output from `git branch --merged` into a set of branch names.
func ParseBranchList(output string) map[string]bool {
	branches := make(map[string]bool)
	for line := range strings.SplitSeq(output, "\n") {
		if len(line) < 3 {
			continue
		}
		name := strings.TrimSpace(line[2:])
		if name != "" {
			branches[name] = true
		}
	}
	return branches
}

// BranchExists checks if a branch exists locally.
func BranchExists(repoRoot, branch string) bool {
	return exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil
}

// RemoteBranchExists checks if a branch exists on origin.
func RemoteBranchExists(repoRoot, branch string) bool {
	return exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", "refs/remotes/origin/"+branch).Run() == nil
}
