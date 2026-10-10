// Package stats records every time a session waits on the user, and sums it up.
package stats

import (
	"cmp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/session"
)

// Opened starts a wait for a session that now needs the user: the reason comes from its record.
func (w *Waits) Opened(id, title string, issue int, model string, rec session.Record, at time.Time) {
	kind, label := CauseOf(rec)
	w.Begin(id, Wait{SessionTitle: title, Issue: issue, Model: model, Kind: kind, Label: label, StartedAt: at.UnixMilli()})
}

// Closed ends the session's open wait, if any, and reports whether there was one.
func (w *Waits) Closed(id string, at time.Time) bool { return w.End(id, at) }

// Wait is one time a session waited on the user.
type Wait struct {
	SessionTitle string `json:"sessionTitle"`
	Issue        int    `json:"issue"`
	Model        string `json:"model"` // the model the session ran, "" when it was not set
	Kind         string `json:"kind"`  // permission, question or idle
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

// CauseOf says why the record's session waits on the user, in the words the Stats view groups by.
func CauseOf(rec session.Record) (kind, label string) {
	switch {
	case rec.State == session.Idle:
		return "idle", "Reply landed"
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

// Waits records when sessions wait on the user: open ones in memory, finished ones in the event log.
type Waits struct {
	log *Log

	mu   sync.Mutex
	open map[string]Wait // by session id
}

// New records each finished wait in log.
func New(log *Log) *Waits { return &Waits{log: log, open: map[string]Wait{}} }

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
	w.log.Record(session.ProjectKey(id), activity.Event{At: at.UnixMilli(), Kind: activity.Wait, Session: id, Issue: wait.Issue, Model: wait.Model, Ms: wait.WaitedMs, Label: wait.Kind, Title: wait.SessionTitle, Cause: wait.Label})
	return true
}

// Stats sums up a project's waits over the last days days, today included.
func (w *Waits) Stats(key string, days int, now time.Time) (Stats, error) {
	days = max(days, 1)
	st := Stats{Days: days, ByCause: []CauseCount{}, ByDay: []DayCount{}, Recent: []Wait{}}
	y, m, d := now.Date()
	from := time.Date(y, m, d-(days-1), 0, 0, 0, 0, now.Location())
	var waits []Wait
	w.log.Since(key, localDay(from), func(e activity.Event) {
		if e.Kind != activity.Wait {
			return
		}
		waits = append(waits, Wait{SessionTitle: e.Title, Issue: e.Issue, Model: e.Model, Kind: e.Label, Label: cmp.Or(e.Cause, e.Label), StartedAt: e.At - e.Ms, WaitedMs: e.Ms})
	})
	w.mu.Lock()
	for id, wait := range w.open {
		if session.ProjectKey(id) == key {
			wait.WaitedMs = 0
			waits = append(waits, wait)
		}
	}
	w.mu.Unlock()

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
