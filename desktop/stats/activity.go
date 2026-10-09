package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Day is what one project's agents did on one local day. The first four fields are also those of
// activity.json, the record before the event log.
type Day struct {
	Prompts       int   `json:"prompts"`
	Sessions      int   `json:"sessions"`
	IssueSessions int   `json:"issueSessions"` // of Sessions, those started from an issue
	WorkMs        int64 `json:"workMs"`        // time sessions spent working

	Ended        int   `json:"-"` // sessions that ended
	SessionMs    int64 `json:"-"` // how long the ended sessions ran
	PRsOpened    int   `json:"-"`
	PRsMerged    int   `json:"-"`
	PRsClosed    int   `json:"-"` // closed without a merge
	LeadMs       int64 `json:"-"` // of the merged PRs, the time from first seen open to merged
	IssuesFiled  int   `json:"-"`
	ClosedIssues []int `json:"-"` // the issues whose session's PR merged
}

// readActivity is a project's activity.json, by local day "2006-01-02". The app no longer writes it.
func readActivity(dir, key string) (map[string]Day, error) {
	days := map[string]Day{}
	path := filepath.Join(dir, key, "activity.json")
	b, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return days, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading activity: %w", err)
	}
	if err := json.Unmarshal(b, &days); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return days, nil
}
