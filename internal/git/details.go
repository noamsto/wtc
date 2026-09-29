package git

import (
	"os/exec"
	"strings"
)

// LoadDetails populates a worktree's detail fields (dirty files, unpushed commits, last commit).
func LoadDetails(wt *Worktree) {
	out, err := exec.Command("git", "-C", wt.Path, "status", "--porcelain").Output() //nolint:gosec // fixed git binary; wt.Path is a discrete arg after -C, not shell-interpreted
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) == 1 && lines[0] == "" {
			wt.DirtyFileNames = nil
		} else {
			names := make([]string, 0, len(lines))
			for _, l := range lines {
				if len(l) > 3 {
					names = append(names, l[3:])
				}
			}
			wt.DirtyFileNames = names
		}
	}

	out, err = exec.Command("git", "-C", wt.Path, "log", "--oneline", "@{upstream}..HEAD").Output() //nolint:gosec // fixed git binary; wt.Path is a discrete arg after -C, not shell-interpreted
	if err == nil {
		text := strings.TrimSpace(string(out))
		if text != "" {
			wt.UnpushedLog = strings.Split(text, "\n")
		}
	}

	out, err = exec.Command("git", "-C", wt.Path, "log", "-1", //nolint:gosec // fixed git binary; wt.Path is a discrete argv element, not shell-interpreted
		"--format=%h %s (%cr)", "--date=relative").Output()
	if err == nil {
		wt.LastCommit = strings.TrimSpace(string(out))
	}

	wt.DetailsLoaded = true
}

// LoadFileDiff returns the diff output for a single file in a worktree.
func LoadFileDiff(wtPath, fileName string) string {
	// Covers staged and unstaged changes vs HEAD
	out, _ := exec.Command("git", "-C", wtPath, "diff", "HEAD", "--", fileName).Output() //nolint:gosec // fixed git binary; wtPath/fileName is a discrete argv element, not shell-interpreted
	if text := strings.TrimSpace(string(out)); text != "" {
		return text
	}
	// Untracked files: diff against empty
	out, _ = exec.Command("git", "-C", wtPath, "diff", "--no-index", "--", "/dev/null", fileName).Output() //nolint:gosec // fixed git binary; wtPath/fileName is a discrete argv element, not shell-interpreted
	if text := strings.TrimSpace(string(out)); text != "" {
		return text
	}
	return "(no diff available)"
}
