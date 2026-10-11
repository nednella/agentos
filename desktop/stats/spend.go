package stats

import (
	"cmp"
	"maps"
	"slices"
	"strconv"

	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/usage"
)

// spend is what API calls used and cost, and which sessions made them.
type spend struct {
	usage.Tokens
	sessions map[string]bool
}

func (s *spend) count(id string, t usage.Tokens) {
	s.Add(t)
	if s.sessions == nil {
		s.sessions = map[string]bool{}
	}
	s.sessions[id] = true
}

func (s *spend) merge(o spend) {
	s.Add(o.Tokens)
	if s.sessions == nil {
		s.sessions = map[string]bool{}
	}
	maps.Copy(s.sessions, o.sessions)
}

func (s spend) row(model, effort string) UsageRow {
	return UsageRow{
		Model: model, Effort: effort, Sessions: len(s.sessions), Tokens: s.Total(), Cost: s.Dollars(),
		Input: s.Input, Output: s.Output, CacheRead: s.CacheRead, CacheWrite: s.CacheWrite, Thinking: s.Thinking,
	}
}

// work is what one issue's sessions used, or one session with no issue.
type work struct {
	issue   int
	session string // "" for an issue
	title   string // of the latest session
	lastAt  int64
	spend
}

func (w *work) merge(o work) {
	w.issue, w.session = o.issue, o.session
	if o.lastAt >= w.lastAt {
		w.lastAt, w.title = o.lastAt, o.title
	}
	w.spend.merge(o.spend)
}

type modelEffort struct{ model, effort string }

// spending is the usage of a day or of a range of days, in all and split up. Chats count in all and
// by model, not in tasks or by work.
type spending struct {
	all     spend
	tasks   spend
	byModel map[modelEffort]*spend
	byWork  map[string]*work // by "#<issue>" or session id
}

func (s *spending) count(e activity.Event) {
	if s.byModel == nil {
		s.byModel, s.byWork = map[modelEffort]*spend{}, map[string]*work{}
	}
	s.all.count(e.Session, e.Tokens)
	key := modelEffort{e.Model, e.Effort}
	if s.byModel[key] == nil {
		s.byModel[key] = &spend{}
	}
	s.byModel[key].count(e.Session, e.Tokens)
	if e.Kind == activity.ChatUsage {
		return
	}
	s.tasks.count(e.Session, e.Tokens)
	one := work{issue: e.Issue, title: e.Title, lastAt: e.At}
	one.count(e.Session, e.Tokens)
	if e.Issue == 0 {
		one.session = e.Session
	}
	s.mergeWork(workKey(e.Issue, e.Session), one)
}

func workKey(issue int, session string) string {
	if issue > 0 {
		return "#" + strconv.Itoa(issue)
	}
	return session
}

func (s *spending) mergeWork(key string, o work) {
	if s.byWork[key] == nil {
		s.byWork[key] = &work{}
	}
	s.byWork[key].merge(o)
}

// merge adds o, never sharing its maps: a day's rollup stays in the cache.
func (s *spending) merge(o spending) {
	if o.byModel == nil {
		return
	}
	if s.byModel == nil {
		s.byModel, s.byWork = map[modelEffort]*spend{}, map[string]*work{}
	}
	s.all.merge(o.all)
	s.tasks.merge(o.tasks)
	for key, sp := range o.byModel {
		if s.byModel[key] == nil {
			s.byModel[key] = &spend{}
		}
		s.byModel[key].merge(*sp)
	}
	for key, w := range o.byWork {
		s.mergeWork(key, *w)
	}
}

// UsageRow is the tokens a model used at one effort, or all of them in the totals.
type UsageRow struct {
	Model      string  `json:"model"`  // "" in the totals
	Effort     string  `json:"effort"` // "" when the session was started without one
	Sessions   int     `json:"sessions"`
	Tokens     int64   `json:"tokens"` // input, output and cache reads and writes together
	Input      int64   `json:"input"`
	Output     int64   `json:"output"` // thinking included
	CacheRead  int64   `json:"cacheRead"`
	CacheWrite int64   `json:"cacheWrite"`
	Thinking   int64   `json:"thinking"`
	Cost       float64 `json:"cost"` // US dollars at list prices
}

// WorkRow is what one issue's sessions used, resumed ones included, or one session with no issue.
type WorkRow struct {
	Project  string  `json:"project"`
	Issue    int     `json:"issue"`   // 0 for a session with no issue
	Session  string  `json:"session"` // "" for an issue
	Title    string  `json:"title"`
	Sessions int     `json:"sessions"`
	Tokens   int64   `json:"tokens"`
	Cost     float64 `json:"cost"`
	LastAt   int64   `json:"lastAt"` // unix ms of the latest usage
}

// Usage is what the agents used over the ledger's days.
type Usage struct {
	Totals     UsageRow   `json:"totals"`  // tasks and chats
	Tasks      UsageRow   `json:"tasks"`   // tasks alone
	ByModel    []UsageRow `json:"byModel"` // tasks and chats, costliest first
	PricesAsOf string     `json:"pricesAsOf"`
}

func (s spending) usage() Usage {
	u := Usage{Totals: s.all.row("", ""), Tasks: s.tasks.row("", ""), ByModel: []UsageRow{}, PricesAsOf: usage.PricesAsOf}
	for key, sp := range s.byModel {
		u.ByModel = append(u.ByModel, sp.row(key.model, key.effort))
	}
	slices.SortFunc(u.ByModel, func(a, b UsageRow) int {
		return cmp.Or(cmp.Compare(b.Cost, a.Cost), cmp.Compare(b.Tokens, a.Tokens), cmp.Compare(a.Model, b.Model), cmp.Compare(a.Effort, b.Effort))
	})
	return u
}

func (s spending) work(project string) []WorkRow {
	out := make([]WorkRow, 0, len(s.byWork))
	for _, w := range s.byWork {
		out = append(out, WorkRow{
			Project: project, Issue: w.issue, Session: w.session, Title: w.title,
			Sessions: len(w.sessions), Tokens: w.Total(), Cost: w.Dollars(), LastAt: w.lastAt,
		})
	}
	return out
}
