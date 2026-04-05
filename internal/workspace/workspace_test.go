package workspace

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceReadSearchAndReplace(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	file := filepath.Join(root, "example.txt")
	if err := os.WriteFile(file, []byte("hello\nworld\nhello again\n"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	ws, err := New(root)
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}

	lines, err := ws.Read("example.txt", 2, 2)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(lines) != 2 || !strings.Contains(lines[0], "world") {
		t.Fatalf("unexpected read output: %#v", lines)
	}

	results, err := ws.Search("hello", 10)
	if err != nil {
		t.Fatalf("search file: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(results))
	}

	preview, err := ws.Replace("example.txt", "world", "codex", false)
	if err != nil {
		t.Fatalf("replace file: %v", err)
	}
	if !strings.Contains(preview, "codex") {
		t.Fatalf("expected preview to contain replacement: %s", preview)
	}

	updated, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if !strings.Contains(string(updated), "codex") {
		t.Fatalf("expected file update, got %q", string(updated))
	}
}

func TestWorkspaceRejectsEscapingPaths(t *testing.T) {
	t.Parallel()

	ws, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new workspace: %v", err)
	}

	if _, err := ws.Resolve("../outside.txt"); err == nil {
		t.Fatal("expected escaping path to fail")
	}
}
