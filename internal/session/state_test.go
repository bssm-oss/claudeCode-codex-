package session

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestStartUpdateRenameAndViewSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "20260407-010203.jsonl")
	transcript, err := Open(transcriptPath)
	if err != nil {
		t.Fatalf("open transcript: %v", err)
	}
	defer func() { _ = transcript.Close() }()
	if err := transcript.Append("assistant", map[string]string{"content": "hello"}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}

	state, err := Start(dir, "/tmp/project", transcriptPath)
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if err := UpdateResponse(dir, state.ID, "resp_123"); err != nil {
		t.Fatalf("update response: %v", err)
	}
	if err := Rename(dir, state.ID, "main chat"); err != nil {
		t.Fatalf("rename session: %v", err)
	}

	latest, err := Latest(dir)
	if err != nil {
		t.Fatalf("latest session: %v", err)
	}
	if latest.LastResponseID != "resp_123" || latest.Name != "main chat" {
		t.Fatalf("unexpected latest state: %#v", latest)
	}

	views, err := Views(dir, 10)
	if err != nil {
		t.Fatalf("views: %v", err)
	}
	if len(views) != 1 || views[0].ID != state.ID || views[0].Name != "main chat" {
		t.Fatalf("unexpected views: %#v", views)
	}

	got, err := Get(dir, "main chat")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if got.ID != state.ID {
		t.Fatalf("unexpected get result: %#v", got)
	}
}

func TestViewsIncludeLegacyTranscriptWithoutState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "legacy.jsonl")
	transcript, err := Open(transcriptPath)
	if err != nil {
		t.Fatalf("open transcript: %v", err)
	}
	defer func() { _ = transcript.Close() }()
	if err := transcript.Append("assistant", map[string]string{"content": "legacy"}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}

	views, err := Views(dir, 10)
	if err != nil {
		t.Fatalf("views: %v", err)
	}
	if len(views) != 1 || !strings.Contains(views[0].TranscriptPath, "legacy.jsonl") {
		t.Fatalf("unexpected views: %#v", views)
	}
	if views[0].Resumable {
		t.Fatalf("expected legacy transcript view to be non-resumable: %#v", views)
	}
}

func TestRenameRejectsDuplicateNames(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstPath := filepath.Join(dir, "one.jsonl")
	secondPath := filepath.Join(dir, "two.jsonl")
	first, _ := Open(firstPath)
	defer func() { _ = first.Close() }()
	_ = first.Append("assistant", map[string]string{"content": "one"})
	second, _ := Open(secondPath)
	defer func() { _ = second.Close() }()
	_ = second.Append("assistant", map[string]string{"content": "two"})
	firstState, err := Start(dir, "/tmp/project", firstPath)
	if err != nil {
		t.Fatalf("start first session: %v", err)
	}
	secondState, err := Start(dir, "/tmp/project", secondPath)
	if err != nil {
		t.Fatalf("start second session: %v", err)
	}
	if err := Rename(dir, firstState.ID, "dup"); err != nil {
		t.Fatalf("rename first session: %v", err)
	}
	if err := Rename(dir, secondState.ID, "dup"); err == nil {
		t.Fatal("expected duplicate session name to fail")
	}
}

func TestLatestPrefersLastActiveNotRenameTime(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	firstPath := filepath.Join(dir, "one.jsonl")
	secondPath := filepath.Join(dir, "two.jsonl")
	first, _ := Open(firstPath)
	defer func() { _ = first.Close() }()
	_ = first.Append("assistant", map[string]string{"content": "one"})
	second, _ := Open(secondPath)
	defer func() { _ = second.Close() }()
	_ = second.Append("assistant", map[string]string{"content": "two"})
	firstState, err := Start(dir, "/tmp/project", firstPath)
	if err != nil {
		t.Fatalf("start first session: %v", err)
	}
	secondState, err := Start(dir, "/tmp/project", secondPath)
	if err != nil {
		t.Fatalf("start second session: %v", err)
	}
	if err := UpdateResponse(dir, firstState.ID, "resp-1"); err != nil {
		t.Fatalf("update first response: %v", err)
	}
	if err := UpdateResponse(dir, secondState.ID, "resp-2"); err != nil {
		t.Fatalf("update second response: %v", err)
	}
	if err := Rename(dir, firstState.ID, "renamed-old"); err != nil {
		t.Fatalf("rename first session: %v", err)
	}
	latest, err := Latest(dir)
	if err != nil {
		t.Fatalf("latest session: %v", err)
	}
	if latest.ID != secondState.ID {
		t.Fatalf("expected second session to remain latest, got %#v", latest)
	}
}

func TestLatestSkipsNonResumableSessions(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	transcriptPath := filepath.Join(dir, "one.jsonl")
	transcript, err := Open(transcriptPath)
	if err != nil {
		t.Fatalf("open transcript: %v", err)
	}
	defer func() { _ = transcript.Close() }()
	_ = transcript.Append("assistant", map[string]string{"content": "one"})
	if _, err := Start(dir, "/tmp/project", transcriptPath); err != nil {
		t.Fatalf("start session: %v", err)
	}
	if _, err := Latest(dir); err == nil {
		t.Fatal("expected latest to reject non-resumable session")
	}
}
