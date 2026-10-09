package stats

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/session"
)

// Day is what one project's agents did on one local day.
type Day struct {
	Prompts       int   `json:"prompts"`
	Sessions      int   `json:"sessions"`
	IssueSessions int   `json:"issueSessions"` // of Sessions, those started from an issue
	WorkMs        int64 `json:"workMs"`        // time sessions spent working
}

// Activity records what the agents do, per project and local day, in a file per project.
type Activity struct {
	dir string

	mu sync.Mutex
}

// NewActivity keeps the record under dir.
func NewActivity(dir string) *Activity { return &Activity{dir: dir} }

func (a *Activity) path(key string) string { return filepath.Join(a.dir, key, "activity.json") }

// Prompt counts a prompt the user sent to the session.
func (a *Activity) Prompt(id string, at time.Time) { a.add(id, at, func(d *Day) { d.Prompts++ }) }

// Session counts a session started, and whether it came from an issue.
func (a *Activity) Session(id string, issue bool, at time.Time) {
	a.add(id, at, func(d *Day) {
		d.Sessions++
		if issue {
			d.IssueSessions++
		}
	})
}

// Worked adds time the session spent working, to the day the work ended.
func (a *Activity) Worked(id string, ms int64, at time.Time) {
	if ms > 0 {
		a.add(id, at, func(d *Day) { d.WorkMs += ms })
	}
}

func (a *Activity) add(id string, at time.Time, change func(*Day)) {
	key := session.ProjectKey(id)
	a.mu.Lock()
	defer a.mu.Unlock()
	days, err := a.read(key)
	if err == nil {
		day := at.In(time.Local).Format(time.DateOnly)
		d := days[day]
		change(&d)
		days[day] = d
		err = a.write(key, days)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: recording activity: %v\n", err)
	}
}

// read returns the project's days, by local day "2006-01-02".
func (a *Activity) read(key string) (map[string]Day, error) {
	days := map[string]Day{}
	b, err := os.ReadFile(a.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return days, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading activity: %w", err)
	}
	if err := json.Unmarshal(b, &days); err != nil {
		return nil, fmt.Errorf("reading %s: %w", a.path(key), err)
	}
	return days, nil
}

func (a *Activity) write(key string, days map[string]Day) error {
	b, err := json.Marshal(days)
	if err != nil {
		return fmt.Errorf("encoding activity: %w", err)
	}
	return atomicfile.Write(a.path(key), b, 0o600)
}

// Days is the project's record, read under the lock.
func (a *Activity) Days(key string) (map[string]Day, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.read(key)
}
