package session

import (
	"os"
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
