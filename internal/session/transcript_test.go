package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptAppendWritesJSONLines(t *testing.T) {
	t.Parallel()

	transcript, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("new transcript: %v", err)
	}
	defer func() { _ = transcript.Close() }()

	if err := transcript.Append("assistant", map[string]string{"content": "hello"}); err != nil {
		t.Fatalf("append transcript: %v", err)
	}

	data, err := os.ReadFile(transcript.Path())
	if err != nil {
		t.Fatalf("read transcript: %v", err)
	}
	if !strings.Contains(string(data), `"type":"assistant"`) {
		t.Fatalf("unexpected transcript contents: %s", data)
	}
}

func TestListAndSearchTranscripts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "sample.jsonl")
	data := strings.Join([]string{
		`{"time":"2026-04-06T00:00:00Z","type":"user","payload":{"prompt":"hello codex"}}`,
		`{"time":"2026-04-06T00:00:01Z","type":"assistant","payload":{"content":"world"}}`,
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write transcript fixture: %v", err)
	}

	summaries, err := List(dir, 10)
	if err != nil {
		t.Fatalf("list transcripts: %v", err)
	}
	if len(summaries) != 1 || summaries[0].EventCount != 2 {
		t.Fatalf("unexpected summaries: %#v", summaries)
	}

	matches, err := Search(dir, "codex", 10)
	if err != nil {
		t.Fatalf("search transcripts: %v", err)
	}
	if len(matches) != 1 || !strings.Contains(matches[0].Snippet, "hello codex") {
		t.Fatalf("unexpected matches: %#v", matches)
	}
}

func TestListSkipsEmptyTranscriptFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.jsonl"), nil, 0o644); err != nil {
		t.Fatalf("write empty transcript: %v", err)
	}
	summaries, err := List(dir, 10)
	if err != nil {
		t.Fatalf("list transcripts: %v", err)
	}
	if len(summaries) != 0 {
		t.Fatalf("expected empty file to be skipped, got %#v", summaries)
	}
}

func TestSearchHandlesLargeTranscriptLine(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	large := strings.Repeat("x", 70*1024)
	data := `{"time":"2026-04-06T00:00:00Z","type":"tool","payload":{"output":"` + large + ` codex"}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "large.jsonl"), []byte(data), 0o644); err != nil {
		t.Fatalf("write large transcript: %v", err)
	}
	matches, err := Search(dir, "codex", 10)
	if err != nil {
		t.Fatalf("search large transcript: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %#v", matches)
	}
}

func TestSearchScansBeyondDefaultListCap(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	for i := 0; i < 25; i++ {
		name := filepath.Join(dir, fmt.Sprintf("%02d.jsonl", i))
		payload := "no-match"
		if i == 0 {
			payload = "very old codex match"
		}
		data := fmt.Sprintf("{\"time\":\"2026-04-06T00:%02d:00Z\",\"type\":\"assistant\",\"payload\":{\"content\":\"%s\"}}\n", i, payload)
		if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
			t.Fatalf("write transcript %d: %v", i, err)
		}
	}
	matches, err := Search(dir, "codex", 10)
	if err != nil {
		t.Fatalf("search transcripts: %v", err)
	}
	if len(matches) != 1 || !strings.Contains(matches[0].Snippet, "very old codex match") {
		t.Fatalf("expected search to include older matching transcript, got %#v", matches)
	}
}
