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

	"github.com/nednella/agentos/internal/prompts"
)

// DefaultPRPoll is how often pull requests are polled when pr_poll is not set.
const DefaultPRPoll = 30 * time.Second

// OtherSection holds the issues no queue section matches.
const OtherSection = "Other"

// Action is a way to start a session for an issue: a command to type, run with its own model and effort.
type Action struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command,omitempty"` // {n} is the issue number, {title} its title
	Model   string `yaml:"model,omitempty"`
	Effort  string `yaml:"effort,omitempty"`
}

// Section is a group of the queue. It takes the issues that have any of its labels, or the issues that have
// none when it names no labels.
type Section struct {
	Name    string   `yaml:"name"`
	Labels  []string `yaml:"labels,omitempty"`
	Actions []Action `yaml:"actions,omitempty"` // the first is the default
}

// Section is the index of the first section an issue with these labels goes in, and the section.
// An issue no section takes goes in OtherSection, one place past the last. A project with no sections
// has one nameless group of every issue.
func (p Project) Section(labels []string) (int, Section) {
	if len(p.QueueSections) == 0 {
		return 0, Section{}
	}
	for i, s := range p.QueueSections {
		if len(s.Labels) == 0 && len(labels) == 0 || slices.ContainsFunc(labels, func(l string) bool { return slices.Contains(s.Labels, l) }) {
			return i, s
		}
	}
	return len(p.QueueSections), Section{Name: OtherSection}
}

// StartAction is the only action of an issue whose section lists none: it types a line that names the issue.
func StartAction() Action { return Action{Name: "Start", Command: prompts.StartIssue()} }

// ActionList is what can be done with an issue of the section, the default first.
func (s Section) ActionList() []Action {
	if len(s.Actions) == 0 {
		return []Action{StartAction()}
	}
	return s.Actions
}

// Render is the command to type for the issue.
func (a Action) Render(number int, title string) string {
	return strings.NewReplacer("{n}", strconv.Itoa(number), "{title}", title).Replace(a.Command)
}

// NoteCommand is what a session for a note types into the agent; the note itself unless the project sets
// note_session_command.
func (p Project) NoteCommand(text string) string {
	return strings.ReplaceAll(cmp.Or(p.NoteSessionCommand, "{text}"), "{text}", text)
}

// The values of Project.SessionPromptSend.
const (
	PromptAuto   = "auto"
	PromptManual = "manual"
)

// SendsPrompt says whether the text a session is started with is sent at once, or waits for Enter.
func (p Project) SendsPrompt() bool { return p.SessionPromptSend != PromptManual }

func (p Project) validateQueue() error {
	if p.SessionPromptSend != "" && p.SessionPromptSend != PromptAuto && p.SessionPromptSend != PromptManual {
		return fmt.Errorf("session_prompt_send must be %s or %s, not %q", PromptAuto, PromptManual, p.SessionPromptSend)
	}
	var sections []string
	for _, s := range p.QueueSections {
		if s.Name == "" {
			return errors.New("queue_sections: a section needs a name")
		}
		if slices.Contains(sections, s.Name) {
			return fmt.Errorf("queue_sections: two sections are named %q", s.Name)
		}
		sections = append(sections, s.Name)
		var actions []string
		for _, a := range s.Actions {
			if a.Name == "" {
				return fmt.Errorf("queue_sections.%s: an action needs a name", s.Name)
			}
			if slices.Contains(actions, a.Name) {
				return fmt.Errorf("queue_sections.%s: two actions are named %q", s.Name, a.Name)
			}
			actions = append(actions, a.Name)
			if err := (Model{Model: a.Model, Effort: a.Effort}).Validate(); err != nil {
				return fmt.Errorf("queue_sections.%s.%s: %w", s.Name, a.Name, err)
			}
		}
	}
	return nil
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
