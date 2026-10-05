package project

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// DefaultPRPoll is how often pull requests are polled when pr_poll is not set.
const DefaultPRPoll = 30 * time.Second

var defaultLanes = map[string]string{
	"ready": "ready", "needs-plan": "plan", "needs-human": "you", "idea": "idea",
}

var defaultCommands = map[string]string{
	"ready": "/work {n}", "plan": "/investigate {n}", "inbox": "", "idea": "", "note": "{text}",
}

// laneOrder ranks lanes when an issue's labels map to several.
var laneOrder = []string{"ready", "plan", "you", "idea"}

// Lane places an issue by its labels: the first lane in laneOrder that a label
// maps to, "inbox" when there are no labels at all, else "idea".
func (p Project) Lane(labels []string) string {
	lanes := p.Lanes
	if len(lanes) == 0 {
		lanes = defaultLanes
	}
	var hit []string
	for _, l := range labels {
		hit = append(hit, lanes[l])
	}
	for _, lane := range laneOrder {
		if slices.Contains(hit, lane) {
			return lane
		}
	}
	if len(labels) == 0 {
		return "inbox"
	}
	return "idea"
}

func (p Project) command(key string) string {
	if c, ok := p.Commands[key]; ok {
		return c
	}
	return defaultCommands[key]
}

// IssueCommand is what a session for an issue in the lane sends to the agent, or "".
func (p Project) IssueCommand(lane string, number int) string {
	if lane != "ready" && lane != "plan" && lane != "inbox" && lane != "idea" {
		return ""
	}
	return strings.ReplaceAll(p.command(lane), "{n}", strconv.Itoa(number))
}

// NoteCommand is what a session for a note types into the agent.
func (p Project) NoteCommand(text string) string {
	return strings.ReplaceAll(p.command("note"), "{text}", text)
}

// BranchFor is the branch the work on an issue happens on.
func (p Project) BranchFor(number int) string {
	pattern := p.Branch
	if pattern == "" {
		pattern = "issue-{n}"
	}
	return strings.ReplaceAll(pattern, "{n}", strconv.Itoa(number))
}

// BrowserOn says whether sessions get the browser and evidence commands.
func (p Project) BrowserOn() bool { return p.Browser == nil || *p.Browser }

// DigestOn says whether the weekly digest runs by itself.
func (p Project) DigestOn() bool { return p.Digest != "off" }

// PRPollEvery is how often the project's pull requests are polled.
func (p Project) PRPollEvery() (time.Duration, error) {
	if p.PRPoll == "" {
		return DefaultPRPoll, nil
	}
	d, err := time.ParseDuration(p.PRPoll)
	if err != nil || d < time.Second {
		return 0, fmt.Errorf("pr_poll must be a duration of at least 1s, like 30s, not %q", p.PRPoll)
	}
	return d, nil
}
