package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	ctl "github.com/nednella/agentos/internal/control"
)

// Asker says which project an agent command is about.
type Asker interface{ AskerProject(req ctl.Request) string }

// Commands answers agentos stats.
type Commands struct {
	waits *Waits
	asker Asker
}

func NewCommands(w *Waits, a Asker) *Commands { return &Commands{waits: w, asker: a} }

// Stats prints the interruption tally of the asking session's project.
func (c *Commands) Stats(_ context.Context, req ctl.Request) (string, error) {
	days := 7
	if v := req.Opts["days"]; v != "" {
		var err error
		if days, err = strconv.Atoi(v); err != nil || days < 1 {
			return "", fmt.Errorf("--days needs a positive number, got %q", v)
		}
	}
	key := c.asker.AskerProject(req)
	st, err := c.waits.Stats(key, days, time.Now())
	if err != nil {
		return "", err
	}
	if req.Opts["json"] != "" {
		out, err := json.Marshal(st)
		return string(out), err
	}
	return formatStats(key, st), nil
}

func formatStats(key string, st Stats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Waits on the owner in %s over the last %d days: %d, %s in all, median %s.\n",
		key, st.Days, st.Total, shortDuration(st.TotalWaitMs), shortDuration(st.MedianWaitMs))
	if st.Total == 0 {
		return strings.TrimRight(b.String(), "\n")
	}
	b.WriteString("Most frequent causes first (count, time waited):\n")
	for _, c := range st.ByCause[:min(len(st.ByCause), 20)] {
		fmt.Fprintf(&b, "  %3dx  %-8s %-40s %s\n", c.Count, c.Kind, c.Label, shortDuration(c.TotalWaitMs))
	}
	return strings.TrimRight(b.String(), "\n")
}

func shortDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
