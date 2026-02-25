package hookdata

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HookStatus represents the status written by a Claude Code hook.
type HookStatus struct {
	PaneID    string    `json:"pane_id"`
	SessionID string    `json:"session_id,omitempty"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Dir returns the hooks directory path (~/.config/cmux/hooks/).
func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "cmux", "hooks")
}

// Write atomically writes a status file for the given pane.
// Uses temp file + rename for atomic writes on the same filesystem.
func Write(status HookStatus) error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(status)
	if err != nil {
		return err
	}

	// Atomic write: temp file in same dir, then rename
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, filepath.Join(dir, fileKey(status.PaneID)))
}

// Read reads the status file for a pane. Returns nil if not found.
func Read(paneID string) (*HookStatus, error) {
	data, err := os.ReadFile(filepath.Join(Dir(), fileKey(paneID)))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var hs HookStatus
	if err := json.Unmarshal(data, &hs); err != nil {
		return nil, err
	}
	return &hs, nil
}

// ReadAll reads all status files and returns a map of paneID → HookStatus.
func ReadAll() map[string]*HookStatus {
	result := make(map[string]*HookStatus)

	entries, err := os.ReadDir(Dir())
	if err != nil {
		return result
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(Dir(), e.Name()))
		if err != nil {
			continue
		}

		var hs HookStatus
		if err := json.Unmarshal(data, &hs); err != nil {
			continue
		}
		result[hs.PaneID] = &hs
	}

	return result
}

// Cleanup removes status files older than maxAge.
func Cleanup(maxAge time.Duration) {
	entries, err := os.ReadDir(Dir())
	if err != nil {
		return
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}

		path := filepath.Join(Dir(), e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var hs HookStatus
		if err := json.Unmarshal(data, &hs); err != nil {
			os.Remove(path)
			continue
		}

		if hs.IsStale(maxAge) {
			os.Remove(path)
		}
	}
}

// IsStale returns true if the status is older than the given duration.
func (h *HookStatus) IsStale(maxAge time.Duration) bool {
	return time.Since(h.Timestamp) > maxAge
}

// fileKey converts a pane ID to a filename (e.g., "%25" → "%25.json").
func fileKey(paneID string) string {
	return paneID + ".json"
}
