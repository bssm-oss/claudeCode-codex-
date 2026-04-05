package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	Time    time.Time       `json:"time"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Transcript struct {
	path string
	file *os.File
}

func New(dir string) (*Transcript, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create transcript dir: %w", err)
	}

	name := time.Now().Format("20060102-150405") + ".jsonl"
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open transcript: %w", err)
	}

	return &Transcript{path: path, file: file}, nil
}

func (t *Transcript) Path() string {
	if t == nil {
		return ""
	}
	return t.path
}

func (t *Transcript) Append(kind string, payload any) error {
	if t == nil {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal transcript payload: %w", err)
	}

	event := Event{
		Time:    time.Now().UTC(),
		Type:    kind,
		Payload: body,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal transcript event: %w", err)
	}

	if _, err := t.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("write transcript event: %w", err)
	}

	return nil
}

func (t *Transcript) Close() error {
	if t == nil || t.file == nil {
		return nil
	}
	return t.file.Close()
}
