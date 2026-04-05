package vcs

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

type Git struct {
	Root string
}

func New(root string) Git {
	return Git{Root: root}
}

func (g Git) IsRepository(ctx context.Context) bool {
	_, err := g.run(ctx, "rev-parse", "--is-inside-work-tree")
	return err == nil
}

func (g Git) Status(ctx context.Context) (string, error) {
	return g.run(ctx, "status", "--short", "--branch")
}

func (g Git) Diff(ctx context.Context) (string, error) {
	return g.run(ctx, "diff", "--stat")
}

func (g Git) Branch(ctx context.Context) (string, error) {
	return g.run(ctx, "branch", "--show-current")
}

func (g Git) CreateBranch(ctx context.Context, name string) (string, error) {
	return g.run(ctx, "checkout", "-b", name)
}

func (g Git) Commit(ctx context.Context, message string) (string, error) {
	if _, err := g.run(ctx, "add", "."); err != nil {
		return "", err
	}
	return g.run(ctx, "commit", "-m", message)
}

func (g Git) run(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = filepath.Clean(g.Root)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		return text, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return text, nil
}
