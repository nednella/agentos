package project

import (
	"cmp"
	"fmt"
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

// PickAgent chooses the provider for a session: the issue's `agent:<name>` label, then the action's agent, "" for the
// default. A label that names no provider is skipped.
func PickAgent(action Action, labels []string) string {
	for _, l := range labels {
		if name, ok := strings.CutPrefix(l, "agent:"); ok && slices.Contains(agent.Names, name) {
			return name
		}
	}
	return action.Agent
}

// Pick chooses the model and effort for a session. From weakest to strongest: the action's, then the issue's
// `model:<x>` and `effort:<y>` labels. A label value that valid rejects is skipped, so each is checked by the
// provider PickAgent chose, and ignored says why, a sentence per label. A session for no issue passes an empty action
// and no labels.
func Pick(action Action, labels []string, valid func(model, effort string) error) (m Model, ignored []string) {
	m = Model{Model: action.Model, Effort: action.Effort}
	for _, l := range slices.Backward(labels) {
		var override Model
		switch {
		case strings.HasPrefix(l, "model:"):
			override.Model = strings.TrimPrefix(l, "model:")
		case strings.HasPrefix(l, "effort:"):
			override.Effort = strings.TrimPrefix(l, "effort:")
		default:
			continue
		}
		if err := valid(override.Model, override.Effort); err != nil {
			ignored = append(ignored, fmt.Sprintf("%s is ignored: %v", l, err))
			continue
		}
		m = override.over(m)
	}
	slices.Reverse(ignored)
	return m, ignored
}
