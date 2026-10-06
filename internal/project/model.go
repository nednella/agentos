package project

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Model is the model and effort level a session's agent runs at. An empty field means unset, which leaves the choice to the agent.
type Model struct {
	Model  string `yaml:"model,omitempty"`
	Effort string `yaml:"effort,omitempty"`
}

var efforts = []string{"low", "medium", "high", "xhigh", "max"}

// modelName keeps a value from being read as a flag when it reaches the agent's command line.
var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\[\]-]*$`)

func (m Model) over(base Model) Model {
	return Model{Model: cmp.Or(m.Model, base.Model), Effort: cmp.Or(m.Effort, base.Effort)}
}

// Validate reports a model or effort the agent would not accept.
func (m Model) Validate() error {
	if m.Model != "" && !modelName.MatchString(m.Model) {
		return fmt.Errorf("model %q is not a model name", m.Model)
	}
	if m.Effort != "" && !slices.Contains(efforts, m.Effort) {
		return fmt.Errorf("effort must be one of %s, not %q", strings.Join(efforts, ", "), m.Effort)
	}
	return nil
}

// Default is the model and effort the config sets for every session.
func (c Config) Default() Model { return Model{Model: c.Model, Effort: c.Effort} }

// Pick chooses the model and effort for a session. From weakest to strongest: base (the config's default),
// the project's own, the action's, then the issue's `model:<x>` and `effort:<y>` labels. A value that fails
// Validate is skipped. A session for no issue passes an empty action and no labels.
func (p Project) Pick(base Model, action Action, labels []string) Model {
	m := Model{Model: p.Model, Effort: p.Effort}.over(base)
	m = Model{Model: action.Model, Effort: action.Effort}.over(m)
	for _, l := range slices.Backward(labels) {
		var override Model
		switch {
		case strings.HasPrefix(l, "model:"):
			override.Model = strings.TrimPrefix(l, "model:")
		case strings.HasPrefix(l, "effort:"):
			override.Effort = strings.TrimPrefix(l, "effort:")
		}
		if override.Validate() == nil {
			m = override.over(m)
		}
	}
	return m
}

func (p Project) validateModel() error { return Model{Model: p.Model, Effort: p.Effort}.Validate() }
