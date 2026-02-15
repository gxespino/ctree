package git

import (
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type cacheEntry struct {
	branch    string
	added     int
	removed   int
	dirty     bool
	fetchedAt time.Time
}

var (
	cache    = make(map[string]cacheEntry)
	cacheMu  sync.Mutex
	cacheTTL = 3 * time.Second
)

var shortstatRegex = regexp.MustCompile(
	`(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`,
)

// GetStats returns git branch and diff statistics for a directory.
// Results are cached with a TTL.
func GetStats(workingDir string) (branch string, added, removed int, dirty bool, err error) {
	cacheMu.Lock()
	if entry, ok := cache[workingDir]; ok && time.Since(entry.fetchedAt) < cacheTTL {
		cacheMu.Unlock()
		return entry.branch, entry.added, entry.removed, entry.dirty, nil
	}
	cacheMu.Unlock()

	branch, err = getBranch(workingDir)
	if err != nil {
		return "", 0, 0, false, err
	}

	added, removed = getDiffStats(workingDir)
	dirty = added > 0 || removed > 0

	cacheMu.Lock()
	cache[workingDir] = cacheEntry{
		branch:    branch,
		added:     added,
		removed:   removed,
		dirty:     dirty,
		fetchedAt: time.Now(),
	}
	cacheMu.Unlock()

	return branch, added, removed, dirty, nil
}

func getBranch(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func getDiffStats(dir string) (added, removed int) {
	// Unstaged changes
	out, err := exec.Command("git", "-C", dir, "diff", "--shortstat").Output()
	if err == nil {
		a, r := parseShortstat(string(out))
		added += a
		removed += r
	}
	// Staged changes
	out, err = exec.Command("git", "-C", dir, "diff", "--cached", "--shortstat").Output()
	if err == nil {
		a, r := parseShortstat(string(out))
		added += a
		removed += r
	}
	return added, removed
}

func parseShortstat(output string) (added, removed int) {
	matches := shortstatRegex.FindStringSubmatch(output)
	if matches == nil {
		return 0, 0
	}
	if len(matches) > 2 && matches[2] != "" {
		added, _ = strconv.Atoi(matches[2])
	}
	if len(matches) > 3 && matches[3] != "" {
		removed, _ = strconv.Atoi(matches[3])
	}
	return added, removed
}
