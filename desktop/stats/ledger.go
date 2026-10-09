package stats

import (
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/project"
)

// heatDays is how many days the heat map covers: 52 weeks.
const heatDays = 364

// Ledger is what the agents did across every project, from the event log.
type Ledger struct {
	Days     int         `json:"days"`     // 0 for lifetime
	Since    string      `json:"since"`    // the first day with any record, "" when there is none
	Totals   LedgerRow   `json:"totals"`   // every project summed
	Projects []LedgerRow `json:"projects"` // most prompts first, then by name
	Heat     []DayCount  `json:"heat"`     // prompts per day, the 364 days ending today, oldest first
}

// LedgerRow is one project's figures, or all of them in the totals.
type LedgerRow struct {
	Project       string `json:"project"`
	Prompts       int    `json:"prompts"`
	Sessions      int    `json:"sessions"`
	IssueSessions int    `json:"issueSessions"`
	WorkMs        int64  `json:"workMs"`
	PRsOpened     int    `json:"prsOpened"`
	PRsMerged     int    `json:"prsMerged"`
	PRsClosed     int    `json:"prsClosed"` // closed without a merge
	IssuesFiled   int    `json:"issuesFiled"`
	IssuesClosed  int    `json:"issuesClosed"`
	AvgSessionMs  int64  `json:"avgSessionMs"` // 0 when no session ended
	AvgLeadMs     int64  `json:"avgLeadMs"`    // from a PR first seen open to merged; 0 when none merged
}

// total is days added up, and the issues closed over them.
type total struct {
	Day
	issuesClosed int
}

func (t total) row(project string) LedgerRow {
	r := LedgerRow{
		Project: project, Prompts: t.Prompts, Sessions: t.Sessions, IssueSessions: t.IssueSessions, WorkMs: t.WorkMs,
		PRsOpened: t.PRsOpened, PRsMerged: t.PRsMerged, PRsClosed: t.PRsClosed, IssuesFiled: t.IssuesFiled, IssuesClosed: t.issuesClosed,
	}
	if t.Ended > 0 {
		r.AvgSessionMs = t.SessionMs / int64(t.Ended)
	}
	if t.PRsMerged > 0 {
		r.AvgLeadMs = t.LeadMs / int64(t.PRsMerged)
	}
	return r
}

// Registry lists the configured projects.
type Registry interface{ List() []project.Project }

// Books builds the ledger from the event log.
type Books struct {
	log      *Log
	projects Registry
}

func NewBooks(l *Log, p Registry) *Books {
	return &Books{log: l, projects: p}
}

// Ledger sums the last days days, today included; 0 is all of the record. An issue closed on two days
// counts once.
func (b *Books) Ledger(days int, now time.Time) (Ledger, error) {
	days = max(days, 0)
	y, m, d := now.Date()
	from := ""
	if days > 0 {
		from = time.Date(y, m, d-(days-1), 0, 0, 0, 0, now.Location()).Format(time.DateOnly)
	}
	projects := b.projects.List()
	led := Ledger{Days: days, Projects: make([]LedgerRow, len(projects)), Heat: make([]DayCount, heatDays)}
	heat := map[string]int{}
	var totals total
	for i, p := range projects {
		var sum total
		closed := map[int]bool{}
		for day, v := range b.log.Days(p.Key()) {
			if led.Since == "" || day < led.Since {
				led.Since = day
			}
			heat[day] += v.Prompts
			if day >= from {
				sum.add(v)
				for _, n := range v.ClosedIssues {
					closed[n] = true
				}
			}
		}
		sum.issuesClosed = len(closed)
		totals.add(sum.Day)
		totals.issuesClosed += sum.issuesClosed
		led.Projects[i] = sum.row(p.Name)
	}
	for i := range led.Heat {
		day := time.Date(y, m, d-(heatDays-1-i), 0, 0, 0, 0, now.Location()).Format(time.DateOnly)
		led.Heat[i] = DayCount{Day: day, Count: heat[day]}
	}

	slices.SortFunc(led.Projects, func(a, b LedgerRow) int {
		if a.Prompts != b.Prompts {
			return b.Prompts - a.Prompts
		}
		return strings.Compare(a.Project, b.Project)
	})
	led.Totals = totals.row("")
	return led, nil
}
