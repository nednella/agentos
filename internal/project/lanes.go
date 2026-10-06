package project

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
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

// ReviewCommand is what a session whose PR got a review or a comment is sent, or "".
func (p Project) ReviewCommand(pr int) string {
	return strings.ReplaceAll(p.OnReview, "{n}", strconv.Itoa(pr))
}

// ChecksCommand is what a session whose PR has failing checks is sent, or "".
func (p Project) ChecksCommand(pr int) string {
	return strings.ReplaceAll(p.OnChecks, "{n}", strconv.Itoa(pr))
}

// BranchFor is the branch the work on an issue happens on, or "" when the project names none.
func (p Project) BranchFor(number int) string {
	return strings.ReplaceAll(p.Branch, "{n}", strconv.Itoa(number))
}

// Cleanup says, per event, whether the app cleans up after a session by itself. An empty
// field takes the default: auto after a merge, manual after a close.
type Cleanup struct {
	Merge string `yaml:"merge,omitempty" json:"merge"`
	Close string `yaml:"close,omitempty" json:"close"`
}

// Modes are the values of Cleanup.Merge and Cleanup.Close.
var Modes = []string{"auto", "manual"}

// UnmarshalYAML turns the old cleanup command into an error that says where it went.
func (c *Cleanup) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind == yaml.ScalarNode {
		return errors.New("cleanup is now a map of merge and close; the command that removes a worktree is remove_worktree")
	}
	type plain Cleanup
	return n.Decode((*plain)(c))
}

func (c Cleanup) validate() error {
	if c.Merge != "" && !slices.Contains(Modes, c.Merge) {
		return fmt.Errorf("cleanup merge must be auto or manual, not %q", c.Merge)
	}
	if c.Close != "" && !slices.Contains(Modes, c.Close) {
		return fmt.Errorf("cleanup close must be auto or manual, not %q", c.Close)
	}
	return nil
}

// OnMerge is the mode after a merge.
func (c Cleanup) OnMerge() string { return cmp.Or(c.Merge, "auto") }

// OnClose is the mode after a close.
func (c Cleanup) OnClose() string { return cmp.Or(c.Close, "manual") }

// Resolved is c with the defaults filled in.
func (c Cleanup) Resolved() Cleanup { return Cleanup{Merge: c.OnMerge(), Close: c.OnClose()} }

// Set changes the mode of an event, "merge" or "close".
func (c *Cleanup) Set(event, mode string) error {
	if !slices.Contains(Modes, mode) {
		return fmt.Errorf("cleanup mode must be auto or manual, not %q", mode)
	}
	switch event {
	case "merge":
		c.Merge = mode
	case "close":
		c.Close = mode
	default:
		return fmt.Errorf("cleanup event must be merge or close, not %q", event)
	}
	return nil
}

// BrowserOn says whether sessions get the browser and evidence commands.
func (p Project) BrowserOn() bool { return p.Browser == nil || *p.Browser }

// KeepsAwake says whether a working session of the project holds off idle sleep: the project's own
// setting, else the config's, else on.
func (c Config) KeepsAwake(p Project) bool {
	switch {
	case p.KeepAwake != nil:
		return *p.KeepAwake
	case c.KeepAwake != nil:
		return *c.KeepAwake
	}
	return true
}

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
