package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// PersistentState is serialized to ~/.config/ctree/state.json.
type PersistentState struct {
	LastSeen map[string]time.Time `json:"last_seen"`
	Version  int                  `json:"version"`
}

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "ctree")
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

// previewFlagPath is a zero-byte file whose existence means "preview on".
func previewFlagPath() string {
	return filepath.Join(configDir(), "preview")
}

// SetPreview persists the preview toggle so all ctree instances stay in sync.
func SetPreview(on bool) {
	if on {
		_ = os.MkdirAll(configDir(), 0o755)
		_ = os.WriteFile(previewFlagPath(), nil, 0o644)
	} else {
		_ = os.Remove(previewFlagPath())
	}
}

// GetPreview reads the shared preview toggle state.
func GetPreview() bool {
	_, err := os.Stat(previewFlagPath())
	return err == nil
}

// bellMutedFlagPath is a zero-byte file whose existence means "bells muted".
// No file = bells ON (preserves default behavior).
func bellMutedFlagPath() string {
	return filepath.Join(configDir(), "bell-muted")
}

// SetBell persists the bell toggle so all ctree instances stay in sync.
func SetBell(on bool) {
	if on {
		_ = os.Remove(bellMutedFlagPath())
	} else {
		_ = os.MkdirAll(configDir(), 0o755)
		_ = os.WriteFile(bellMutedFlagPath(), nil, 0o644)
	}
}

// GetBell reads the shared bell toggle state.
func GetBell() bool {
	_, err := os.Stat(bellMutedFlagPath())
	return err != nil // no file = bells on
}

// slackFlagPath is a zero-byte file whose existence means "slack notifications on".
// No file = slack OFF (default).
func slackFlagPath() string {
	return filepath.Join(configDir(), "slack-enabled")
}

// SetSlack persists the slack toggle so all ctree instances stay in sync.
func SetSlack(on bool) {
	if on {
		_ = os.MkdirAll(configDir(), 0o755)
		_ = os.WriteFile(slackFlagPath(), nil, 0o644)
	} else {
		_ = os.Remove(slackFlagPath())
	}
}

// GetSlack reads the shared slack toggle state.
func GetSlack() bool {
	_, err := os.Stat(slackFlagPath())
	return err == nil // no file = slack off
}
