package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const maxTranscriptLineBytes = 1024 * 1024

type Event struct {
	Time    time.Time       `json:"time"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type Transcript struct {
	path string
	file *os.File
}

type Summary struct {
	Path       string
	EventCount int
	StartedAt  time.Time
	EndedAt    time.Time
}

type Match struct {
	Path    string
	Time    time.Time
	Type    string
	Snippet string
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

func List(dir string, limit int) ([]Summary, error) {
	summaries, err := listAll(dir)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if len(summaries) > limit {
		summaries = summaries[:limit]
	}
	return summaries, nil
}

func listAll(dir string) ([]Summary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read transcript dir: %w", err)
	}
	summaries := make([]Summary, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		summary, err := summarize(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		if summary.EventCount == 0 {
			continue
		}
		summaries = append(summaries, summary)
	}
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].StartedAt.After(summaries[j].StartedAt)
	})
	return summaries, nil
}

func Search(dir, query string, limit int) ([]Match, error) {
	if limit <= 0 {
		limit = 20
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}
	if _, err := os.ReadDir(dir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read transcript dir: %w", err)
	}
	queryLower := strings.ToLower(query)
	matches := make([]Match, 0, limit)
	summaries, err := listAll(dir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		paths = append(paths, summary.Path)
	}
	if len(paths) == 0 {
		return nil, nil
	}
	for _, path := range paths {
		fileMatches, err := searchFile(path, queryLower, limit-len(matches))
		if err != nil {
			return nil, err
		}
		matches = append(matches, fileMatches...)
		if len(matches) >= limit {
			break
		}
	}
	return matches, nil
}

func summarize(path string) (Summary, error) {
	file, err := os.Open(path)
	if err != nil {
		return Summary{}, fmt.Errorf("open transcript %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTranscriptLineBytes)
	summary := Summary{Path: path}
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return Summary{}, fmt.Errorf("parse transcript %s: %w", path, err)
		}
		if summary.EventCount == 0 {
			summary.StartedAt = event.Time
		}
		summary.EventCount++
		summary.EndedAt = event.Time
	}
	if err := scanner.Err(); err != nil {
		return Summary{}, fmt.Errorf("scan transcript %s: %w", path, err)
	}
	return summary, nil
}

func searchFile(path, query string, remaining int) ([]Match, error) {
	if remaining <= 0 {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open transcript %s: %w", path, err)
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), maxTranscriptLineBytes)
	matches := make([]Match, 0, remaining)
	for scanner.Scan() {
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("parse transcript %s: %w", path, err)
		}
		text := strings.ToLower(event.Type + " " + string(event.Payload))
		if !strings.Contains(text, query) {
			continue
		}
		matches = append(matches, Match{
			Path:    path,
			Time:    event.Time,
			Type:    event.Type,
			Snippet: string(event.Payload),
		})
		if len(matches) >= remaining {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan transcript %s: %w", path, err)
	}
	return matches, nil
}
