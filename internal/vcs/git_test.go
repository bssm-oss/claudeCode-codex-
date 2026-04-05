package vcs

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitStatusAndBranch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "config", "user.email", "test@example.com")
	runGit(t, root, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	git := New(root)
	status, err := git.Status(context.Background())
	if err != nil {
		t.Fatalf("git status: %v", err)
	}
	if status == "" {
		t.Fatal("expected git status output")
	}

	branch, err := git.Branch(context.Background())
	if err != nil {
		t.Fatalf("git branch: %v", err)
	}
	if branch == "" {
		t.Fatal("expected current branch output")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
