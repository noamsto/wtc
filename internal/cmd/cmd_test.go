package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/noamsto/wt/internal/runtime"
	"github.com/noamsto/wt/internal/tmux"
)

// testRepo creates a temp git repo with an initial commit on main.
func testRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	if err := os.MkdirAll(repo, 0o750); err != nil {
		t.Fatal(err)
	}
	gitRun(t, repo, "init", "-b", "main")
	gitRun(t, repo, "commit", "--allow-empty", "-m", "init")
	return repo
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec // fixed git binary; test-supplied args are discrete argv elements, not shell-interpreted
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=Test",
		"GIT_AUTHOR_EMAIL=test@test.com",
		"GIT_COMMITTER_NAME=Test",
		"GIT_COMMITTER_EMAIL=test@test.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func testRuntime() runtime.Runtime {
	return runtime.Runtime{Yes: true, Quiet: true, NoSwitch: true}
}

func testTmux() *tmux.Client {
	return tmux.New(false)
}

func TestClean_NoWorktrees(t *testing.T) {
	repo := testRepo(t)
	rt := testRuntime()

	err := Clean(repo, false, rt, testTmux())
	if err != nil {
		t.Fatalf("Clean() error: %v", err)
	}
}

func TestClean_NoStale(t *testing.T) {
	repo := testRepo(t)
	rt := testRuntime()

	// Create a worktree that is not stale
	wtPath := filepath.Join(repo, ".worktrees", "feat-active")
	gitRun(t, repo, "worktree", "add", "-b", "feat-active", wtPath)
	// Make a commit on the branch so it's not merged
	gitRun(t, wtPath, "commit", "--allow-empty", "-m", "active work")

	err := Clean(repo, false, rt, testTmux())
	if err != nil {
		t.Fatalf("Clean() error: %v", err)
	}
}
