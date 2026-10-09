package stats

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/project"
)

// heatDays is how many days the heat map covers: 52 weeks.
const heatDays = 364

// Ledger is what the agents did across every project, from the activity record.
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
}

// Registry lists the configured projects.
type Registry interface{ List() []project.Project }

// Books builds the ledger from the activity record.
type Books struct {
	activity *Activity
	projects Registry
}

func NewBooks(a *Activity, p Registry) *Books {
	return &Books{activity: a, projects: p}
}

// Ledger sums the last days days, today included; 0 is all of the record.
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
	for i, p := range projects {
		record, err := b.activity.Days(p.Key())
		if err != nil {
			return Ledger{}, fmt.Errorf("reading the activity of %s: %w", p.Name, err)
		}
		row := &led.Projects[i]
		row.Project = p.Name
		for day, v := range record {
			if led.Since == "" || day < led.Since {
				led.Since = day
			}
			heat[day] += v.Prompts
			if day >= from {
				row.Prompts += v.Prompts
				row.Sessions += v.Sessions
				row.IssueSessions += v.IssueSessions
				row.WorkMs += v.WorkMs
			}
		}
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
	for _, row := range led.Projects {
		led.Totals.Prompts += row.Prompts
		led.Totals.Sessions += row.Sessions
		led.Totals.IssueSessions += row.IssueSessions
		led.Totals.WorkMs += row.WorkMs
	}
	return led, nil
}
