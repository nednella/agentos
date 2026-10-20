package project

import (
	"cmp"
	"slices"
	"strings"

	"github.com/nednella/agentos/internal/agent"
)

// Model is the model and effort level a session's agent runs at. An empty field means unset, which leaves the choice to the agent.
type Model struct {
	Model  string `yaml:"model,omitempty"`
	Effort string `yaml:"effort,omitempty"`
}

func (m Model) over(base Model) Model {
	return Model{Model: cmp.Or(m.Model, base.Model), Effort: cmp.Or(m.Effort, base.Effort)}
}

// Validate reports a model or effort no agent would accept.
func (m Model) Validate() error { return agent.CheckModel(m.Model, m.Effort) }

// Pick chooses the model and effort for a session. From weakest to strongest: the action's, then the issue's
// `model:<x>` and `effort:<y>` labels. A value that fails Validate is skipped. A session for no issue passes an
// empty action and no labels.
func Pick(action Action, labels []string) Model {
	m := Model{Model: action.Model, Effort: action.Effort}
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
