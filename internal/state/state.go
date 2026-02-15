package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PersistentState is serialized to ~/.config/cmux/state.json.
type PersistentState struct {
	LastSeen map[string]time.Time `json:"last_seen"`
	Version  int                  `json:"version"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cmux")
}

func statePath() string {
	return filepath.Join(configDir(), "state.json")
}

// Load reads state from disk. Returns empty state if file doesn't exist.
func Load() (*PersistentState, error) {
	data, err := os.ReadFile(statePath())
	if err != nil {
		if os.IsNotExist(err) {
			return newState(), nil
		}
		return nil, err
	}
	var s PersistentState
	if err := json.Unmarshal(data, &s); err != nil {
		return newState(), nil
	}
	if s.LastSeen == nil {
		s.LastSeen = make(map[string]time.Time)
	}
	return &s, nil
}

// Save writes state to disk, creating the directory if needed.
func Save(s *PersistentState) error {
	if err := os.MkdirAll(configDir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(statePath(), data, 0o644)
}

// MarkSeen records that the user jumped to this window at the current time.
func (s *PersistentState) MarkSeen(windowID string) {
	s.LastSeen[windowID] = time.Now()
}

func newState() *PersistentState {
	return &PersistentState{
		LastSeen: make(map[string]time.Time),
		Version:  1,
	}
}
