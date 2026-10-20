package project

import (
	"errors"
	"testing"

	"github.com/nednella/agentos/internal/agent"
)

func TestPick(t *testing.T) {
	plan := Action{Name: "Plan", Model: "opus"}
	noHaiku := func(model, effort string) error {
		if model == "haiku" {
			return errors.New("no haiku")
		}
		return agent.CheckModel(model, effort)
	}
	tests := []struct {
		name   string
		action Action
		labels []string
		valid  func(model, effort string) error
		want   Model
	}{
		{"nothing set passes nothing", Action{}, nil, agent.CheckModel, Model{}},
		{"the action's", plan, nil, agent.CheckModel, Model{"opus", ""}},
		{"labels over action, field by field", Action{Name: "Plan", Model: "opus", Effort: "low"}, []string{"type:bug", "model:sonnet", "effort:max"}, agent.CheckModel, Model{"sonnet", "max"}},
		{"a label fills what the action leaves unset", plan, []string{"effort:high"}, agent.CheckModel, Model{"opus", "high"}},
		{"first label of a kind wins", Action{}, []string{"model:opus", "model:haiku"}, agent.CheckModel, Model{"opus", ""}},
		{"bad labels are skipped", plan, []string{"model:--x", "effort:huge"}, agent.CheckModel, Model{"opus", ""}},
		{"a label the provider rejects is skipped", plan, []string{"model:haiku", "effort:high"}, noHaiku, Model{"opus", "high"}},
	}
	for _, tt := range tests {
		if got := Pick(tt.action, tt.labels, tt.valid); got != tt.want {
			t.Errorf("%s: Pick = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestPickAgent(t *testing.T) {
	onAgy := Action{Name: "Work", Agent: "agy"}
	tests := []struct {
		name   string
		action Action
		labels []string
		want   string
	}{
		{"nothing set leaves the default", Action{}, nil, ""},
		{"the action's", onAgy, []string{"type:bug"}, "agy"},
		{"label over action", onAgy, []string{"agent:claude"}, "claude"},
		{"first label wins", Action{}, []string{"agent:agy", "agent:claude"}, "agy"},
		{"a label naming no provider is skipped", onAgy, []string{"agent:codex"}, "agy"},
	}
	for _, tt := range tests {
		if got := PickAgent(tt.action, tt.labels); got != tt.want {
			t.Errorf("%s: PickAgent = %q, want %q", tt.name, got, tt.want)
		}
	}
}
