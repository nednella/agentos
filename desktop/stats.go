package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/session"
)

// Wait is one time a session waited on the user.
type Wait struct {
	SessionTitle string `json:"sessionTitle"`
	Issue        int    `json:"issue"`
	Kind         string `json:"kind"`
	Label        string `json:"label"`
	StartedAt    int64  `json:"startedAt"`
	WaitedMs     int64  `json:"waitedMs"`
}

type CauseCount struct {
	Kind        string `json:"kind"`
	Label       string `json:"label"`
	Count       int    `json:"count"`
	TotalWaitMs int64  `json:"totalWaitMs"`
}

type DayCount struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

type Stats struct {
	Days         int          `json:"days"`
	Total        int          `json:"total"`
	TotalWaitMs  int64        `json:"totalWaitMs"`
	MedianWaitMs int64        `json:"medianWaitMs"`
	ByCause      []CauseCount `json:"byCause"`
	ByDay        []DayCount   `json:"byDay"`
	Recent       []Wait       `json:"recent"`
}

const recentWaits = 50

// causeOf says why the record's session waits on the user, in the words the Stats view groups by.
func causeOf(rec session.Record) (kind, label string) {
	switch {
	case rec.State == session.Finished:
		return "finished", "Turn finished"
	case rec.Notify == "permission_prompt" || (rec.Notify == "" && strings.Contains(strings.ToLower(rec.Detail), "permission")):
		return "permission", permissionLabel(rec.Tool, rec.Command)
	}
	return "question", "Question"
}

func permissionLabel(tool, command string) string {
	switch {
	case tool == "":
		return "Permission"
	case tool == "Bash" && command != "":
		return "Bash: " + strings.Join(firstWords(command, 3), " ")
	}
	return tool
}

func firstWords(s string, n int) []string {
	w := strings.Fields(s)
	return w[:min(len(w), n)]
}

// Waits records when sessions wait on the user: open ones in memory, finished ones in a file per project.
type Waits struct {
	dir string

	mu   sync.Mutex
	open map[string]Wait // by session id
}

func newWaits(dir string) *Waits { return &Waits{dir: dir, open: map[string]Wait{}} }

func (w *Waits) path(key string) string { return filepath.Join(w.dir, key, "stats.jsonl") }

// Begin opens a wait for the session.
func (w *Waits) Begin(id string, wait Wait) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.open[id] = wait
}

// End closes the session's open wait, if any, and reports whether there was one.
func (w *Waits) End(id string, at time.Time) bool {
	w.mu.Lock()
	wait, ok := w.open[id]
	delete(w.open, id)
	w.mu.Unlock()
	if !ok {
		return false
	}
	wait.WaitedMs = max(at.UnixMilli()-wait.StartedAt, 1)
	if err := w.append(projectKeyOf(id), wait); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: recording a wait: %v\n", err)
	}
	return true
}

func projectKeyOf(id string) string {
	name, err := session.ParseName(id)
	if err != nil {
		return "project"
	}
	return name.Project
}

// append adds a line to the project's file. The file is replaced whole, so a
// synced folder never holds half a line.
func (w *Waits) append(key string, wait Wait) error {
	line, err := json.Marshal(wait)
	if err != nil {
		return fmt.Errorf("encoding wait: %w", err)
	}
	old, err := os.ReadFile(w.path(key))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("reading stats file: %w", err)
	}
	return atomicfile.Write(w.path(key), append(append(old, line...), '\n'), 0o600)
}

func (w *Waits) read(key string) ([]Wait, error) {
	f, err := os.Open(w.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading stats: %w", err)
	}
	defer f.Close()
	var out []Wait
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var wait Wait
		if json.Unmarshal(sc.Bytes(), &wait) == nil {
			out = append(out, wait)
		}
	}
	return out, sc.Err()
}

// Stats sums up a project's waits over the last days days, today included.
func (w *Waits) Stats(key string, days int, now time.Time) (Stats, error) {
	days = max(days, 1)
	st := Stats{Days: days, ByCause: []CauseCount{}, ByDay: []DayCount{}, Recent: []Wait{}}
	waits, err := w.read(key)
	if err != nil {
		return st, err
	}
	w.mu.Lock()
	for id, wait := range w.open {
		if projectKeyOf(id) == key {
			wait.WaitedMs = 0
			waits = append(waits, wait)
		}
	}
	w.mu.Unlock()

	y, m, d := now.Date()
	from := time.Date(y, m, d-(days-1), 0, 0, 0, 0, now.Location())
	perDay := map[string]int{}
	causes := map[[2]string]*CauseCount{}
	var closed []int64
	var inRange []Wait
	for _, wait := range waits {
		if wait.StartedAt < from.UnixMilli() {
			continue
		}
		inRange = append(inRange, wait)
		st.Total++
		st.TotalWaitMs += wait.WaitedMs
		perDay[time.UnixMilli(wait.StartedAt).In(now.Location()).Format(time.DateOnly)]++
		c := causes[[2]string{wait.Kind, wait.Label}]
		if c == nil {
			c = &CauseCount{Kind: wait.Kind, Label: wait.Label}
			causes[[2]string{wait.Kind, wait.Label}] = c
		}
		c.Count++
		c.TotalWaitMs += wait.WaitedMs
		if wait.WaitedMs > 0 {
			closed = append(closed, wait.WaitedMs)
		}
	}
	slices.Sort(closed)
	if n := len(closed); n > 0 {
		st.MedianWaitMs = closed[n/2]
		if n%2 == 0 {
			st.MedianWaitMs = (closed[n/2-1] + closed[n/2]) / 2
		}
	}
	for _, c := range causes {
		st.ByCause = append(st.ByCause, *c)
	}
	slices.SortFunc(st.ByCause, func(a, b CauseCount) int {
		if a.Count != b.Count {
			return b.Count - a.Count
		}
		if a.TotalWaitMs != b.TotalWaitMs {
			return int(b.TotalWaitMs - a.TotalWaitMs)
		}
		return strings.Compare(a.Label, b.Label)
	})
	for i := range days {
		day := from.AddDate(0, 0, i).Format(time.DateOnly)
		st.ByDay = append(st.ByDay, DayCount{Day: day, Count: perDay[day]})
	}
	slices.SortStableFunc(inRange, func(a, b Wait) int { return int(b.StartedAt - a.StartedAt) })
	st.Recent = append(st.Recent, inRange[:min(len(inRange), recentWaits)]...)
	return st, nil
}
