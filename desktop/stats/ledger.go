package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/util"
)

// heatDays is how many days the heat map covers: 52 weeks.
const heatDays = 364

// Ledger is what the agents did across every project.
type Ledger struct {
	Days     int         `json:"days"`     // 0 for lifetime
	Since    string      `json:"since"`    // the first day with any record, "" when there is none
	Totals   LedgerRow   `json:"totals"`   // every project summed
	Projects []LedgerRow `json:"projects"` // most prompts first, then by name
	Heat     []DayCount  `json:"heat"`     // prompts per day, the 364 days ending today, oldest first
	GitHub   string      `json:"github"`   // why the GitHub numbers are missing, "" when none is
}

// LedgerRow is one project's figures, or all of them in the totals. A GitHub number is nil when
// the project has no repo or GitHub did not answer.
type LedgerRow struct {
	Project       string `json:"project"`
	Prompts       int    `json:"prompts"`
	Sessions      int    `json:"sessions"`
	IssueSessions int    `json:"issueSessions"`
	WorkMs        int64  `json:"workMs"`
	PRsMerged     *int   `json:"prsMerged"`
	PRsClosed     *int   `json:"prsClosed"` // closed without merging
	IssuesClosed  *int   `json:"issuesClosed"`
	IssuesOpen    *int   `json:"issuesOpen"` // open now, whatever the range
}

// Registry lists the configured projects.
type Registry interface{ List() []project.Project }

// Repos says which GitHub repository a folder belongs to.
type Repos interface {
	Repo(ctx context.Context, dir string) string
}

// Books builds the ledger from the activity record and GitHub.
type Books struct {
	activity *Activity
	projects Registry
	repos    Repos
	gh       run.Runner
}

func NewBooks(a *Activity, p Registry, r Repos, gh run.Runner) *Books {
	return &Books{activity: a, projects: p, repos: r, gh: gh}
}

// Ledger sums the last days days, today included; 0 is all of the record. GitHub is asked once per project.
func (b *Books) Ledger(ctx context.Context, days int, now time.Time) (Ledger, error) {
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

	errs := make([]error, len(projects))
	var wg sync.WaitGroup
	for i, p := range projects {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs[i] = b.github(ctx, p, from, &led.Projects[i])
		}()
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			led.GitHub = util.FirstLine(err.Error())
			break
		}
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
		add(&led.Totals.PRsMerged, row.PRsMerged)
		add(&led.Totals.PRsClosed, row.PRsClosed)
		add(&led.Totals.IssuesClosed, row.IssuesClosed)
		add(&led.Totals.IssuesOpen, row.IssuesOpen)
	}
	return led, nil
}

// add sums n into *total, which stays nil until some project has a number.
func add(total **int, n *int) {
	if n == nil {
		return
	}
	if *total == nil {
		*total = new(int)
	}
	**total += *n
}

// github fills the row's GitHub numbers from the project's repository, if it has one. The range
// starts on from, "" for all time.
func (b *Books) github(ctx context.Context, p project.Project, from string, row *LedgerRow) error {
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	repo := b.repos.Repo(ctx, p.Dir)
	if repo == "" {
		return nil
	}
	out, err := b.gh(ctx, p.Dir, "gh", "api", "graphql", "-f", "query="+ledgerQuery(repo, from))
	if err != nil {
		return fmt.Errorf("asking GitHub about %s: %w", repo, err)
	}
	var v struct {
		Data map[string]struct {
			IssueCount int `json:"issueCount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return fmt.Errorf("reading GitHub's answer about %s: %w", repo, err)
	}
	count := func(name string) *int {
		c, ok := v.Data[name]
		if !ok {
			return nil
		}
		return &c.IssueCount
	}
	row.PRsMerged, row.PRsClosed = count("merged"), count("closedPRs")
	row.IssuesClosed, row.IssuesOpen = count("issuesClosed"), count("issuesOpen")
	return nil
}

func ledgerQuery(repo, from string) string {
	since := func(field string) string {
		if from == "" {
			return ""
		}
		return " " + field + ":>=" + from
	}
	search := func(name, terms string) string {
		return fmt.Sprintf(`%s: search(query:"repo:%s %s", type:ISSUE, first:0){issueCount}`, name, repo, terms)
	}
	return "{ " + strings.Join([]string{
		search("merged", "is:pr is:merged author:@me"+since("merged")),
		search("closedPRs", "is:pr is:closed is:unmerged author:@me"+since("closed")),
		search("issuesClosed", "is:issue is:closed"+since("closed")),
		search("issuesOpen", "is:issue is:open"),
	}, " ") + " }"
}
