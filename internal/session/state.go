package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const sessionsIndexFile = "sessions.json"

type State struct {
	ID             string    `json:"id"`
	Name           string    `json:"name,omitempty"`
	TranscriptPath string    `json:"transcript_path"`
	Workspace      string    `json:"workspace,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	LastActiveAt   time.Time `json:"last_active_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	LastResponseID string    `json:"last_response_id,omitempty"`
}

type SessionView struct {
	ID             string
	Name           string
	TranscriptPath string
	EventCount     int
	StartedAt      time.Time
	UpdatedAt      time.Time
	LastResponseID string
	Resumable      bool
}

func Start(dir, workspace, transcriptPath string) (State, error) {
	state := State{
		ID:             strings.TrimSuffix(filepath.Base(transcriptPath), filepath.Ext(transcriptPath)),
		TranscriptPath: transcriptPath,
		Workspace:      workspace,
		CreatedAt:      time.Now().UTC(),
		LastActiveAt:   time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}
	states, err := loadStates(dir)
	if err != nil {
		return State{}, err
	}
	states = upsertState(states, state)
	if err := saveStates(dir, states); err != nil {
		return State{}, err
	}
	return state, nil
}

func UpdateResponse(dir, id, responseID string) error {
	states, err := loadStates(dir)
	if err != nil {
		return err
	}
	for i := range states {
		if states[i].ID != id {
			continue
		}
		states[i].LastResponseID = responseID
		states[i].LastActiveAt = time.Now().UTC()
		states[i].UpdatedAt = time.Now().UTC()
		return saveStates(dir, states)
	}
	return fmt.Errorf("session %s not found", id)
}

func Rename(dir, id, name string) error {
	states, err := loadStates(dir)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(name)
	for _, state := range states {
		if state.ID == id {
			continue
		}
		if trimmed != "" && state.Name == trimmed {
			return fmt.Errorf("session name %s already exists", trimmed)
		}
	}
	for i := range states {
		if states[i].ID != id {
			continue
		}
		states[i].Name = trimmed
		states[i].UpdatedAt = time.Now().UTC()
		return saveStates(dir, states)
	}
	return fmt.Errorf("session %s not found", id)
}

func Latest(dir string) (State, error) {
	states, err := loadStates(dir)
	if err != nil {
		return State{}, err
	}
	for _, state := range states {
		if strings.TrimSpace(state.LastResponseID) != "" {
			return state, nil
		}
	}
	return State{}, errors.New("no resumable sessions found")
}

func Get(dir, id string) (State, error) {
	states, err := loadStates(dir)
	if err != nil {
		return State{}, err
	}
	for _, state := range states {
		if state.ID == id {
			if strings.TrimSpace(state.LastResponseID) == "" {
				return State{}, fmt.Errorf("session %s has no resumable response chain", id)
			}
			return state, nil
		}
	}
	var matches []State
	for _, state := range states {
		if state.Name == id {
			matches = append(matches, state)
		}
	}
	if len(matches) == 1 {
		if strings.TrimSpace(matches[0].LastResponseID) == "" {
			return State{}, fmt.Errorf("session %s has no resumable response chain", id)
		}
		return matches[0], nil
	}
	if len(matches) > 1 {
		return State{}, fmt.Errorf("session name %s is ambiguous", id)
	}
	return State{}, fmt.Errorf("session %s not found", id)
}

func Views(dir string, limit int) ([]SessionView, error) {
	states, err := loadStates(dir)
	if err != nil {
		return nil, err
	}
	byPath := make(map[string]State, len(states))
	for _, state := range states {
		byPath[state.TranscriptPath] = state
	}
	summaries, err := listAll(dir)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	views := make([]SessionView, 0, len(summaries))
	for _, summary := range summaries {
		state := byPath[summary.Path]
		id := state.ID
		if id == "" {
			id = strings.TrimSuffix(filepath.Base(summary.Path), filepath.Ext(summary.Path))
		}
		updatedAt := state.UpdatedAt
		if updatedAt.IsZero() {
			updatedAt = summary.EndedAt
		}
		resumable := strings.TrimSpace(state.LastResponseID) != ""
		views = append(views, SessionView{
			ID:             id,
			Name:           state.Name,
			TranscriptPath: summary.Path,
			EventCount:     summary.EventCount,
			StartedAt:      summary.StartedAt,
			UpdatedAt:      updatedAt,
			LastResponseID: state.LastResponseID,
			Resumable:      resumable,
		})
	}
	if len(views) > limit {
		views = views[:limit]
	}
	return views, nil
}

func loadStates(dir string) ([]State, error) {
	path := filepath.Join(dir, sessionsIndexFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read sessions index: %w", err)
	}
	var states []State
	if err := json.Unmarshal(data, &states); err != nil {
		return nil, fmt.Errorf("parse sessions index: %w", err)
	}
	sort.Slice(states, func(i, j int) bool {
		return activityTime(states[i]).After(activityTime(states[j]))
	})
	return states, nil
}

func saveStates(dir string, states []State) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create transcript dir: %w", err)
	}
	sort.Slice(states, func(i, j int) bool {
		return activityTime(states[i]).After(activityTime(states[j]))
	})
	data, err := json.MarshalIndent(states, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions index: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, sessionsIndexFile), append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write sessions index: %w", err)
	}
	return nil
}

func upsertState(states []State, state State) []State {
	for i := range states {
		if states[i].ID != state.ID {
			continue
		}
		states[i] = state
		return states
	}
	return append(states, state)
}

func activityTime(state State) time.Time {
	if !state.LastActiveAt.IsZero() {
		return state.LastActiveAt
	}
	if !state.UpdatedAt.IsZero() {
		return state.UpdatedAt
	}
	return state.CreatedAt
}
