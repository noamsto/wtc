package git

import (
	"os/exec"
	"strings"
	"sync"
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

// DetectStale marks worktrees as stale using two local strategies:
// merged branches and deleted remote branches.
func DetectStale(repoRoot, defaultBranch string, worktrees []Worktree) {
	// Strategy 1: git branch --merged
	out, err := exec.Command("git", "-C", repoRoot, "branch", "--merged", defaultBranch).Output() //nolint:gosec // fixed git binary; defaultBranch is a discrete argv element, not shell-interpreted
	if err == nil {
		merged := ParseBranchList(string(out))
		for i := range worktrees {
			if merged[worktrees[i].Branch] {
				worktrees[i].StaleReason = "merged into " + defaultBranch
			}
		}
	}

	// Strategy 2: remote branch deleted (only when origin remote exists)
	hasOrigin := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Run() == nil //nolint:gosec // fixed git binary; repoRoot is a discrete argv element, not shell-interpreted
	if hasOrigin {
		for i := range worktrees {
			if worktrees[i].IsStale() {
				continue
			}
			ref := "refs/remotes/origin/" + worktrees[i].Branch
			if exec.Command("git", "-C", repoRoot, "show-ref", "--verify", "--quiet", ref).Run() != nil { //nolint:gosec // fixed git binary; ref is a discrete argv element, not shell-interpreted
				worktrees[i].StaleReason = "remote branch deleted"
			}
		}
	}
}

// DetectStaleGh checks unchecked worktrees against GitHub PRs for squash-merges.
func DetectStaleGh(repoRoot string, worktrees []Worktree) {
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		return
	}

	var unchecked []int
	for i := range worktrees {
		if !worktrees[i].IsStale() {
			unchecked = append(unchecked, i)
		}
	}
	if len(unchecked) == 0 {
		return
	}

	type result struct {
		index int
		prNum string
	}

	var (
		mu      sync.Mutex
		results []result
		wg      sync.WaitGroup
	)

	for _, idx := range unchecked {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			cmd := exec.Command(ghPath, "pr", "list", //nolint:gosec // fixed gh (from LookPath) binary; the branch name is a discrete argv element, not shell-interpreted
				"--head", worktrees[i].Branch,
				"--state", "merged",
				"--json", "number",
				"--jq", ".[0].number",
			)
			cmd.Dir = repoRoot
			out, err := cmd.Output()
			if err != nil {
				return
			}
			prNum := strings.TrimSpace(string(out))
			if prNum != "" {
				mu.Lock()
				results = append(results, result{index: i, prNum: prNum})
				mu.Unlock()
			}
		}(idx)
	}
	wg.Wait()

	for _, r := range results {
		worktrees[r.index].StaleReason = "PR #" + r.prNum + " merged"
	}
}
