package project

import (
	"errors"
	"slices"
	"testing"

	"github.com/nednella/agentos/internal/agent"
)

func TestPick(t *testing.T) {
	plan := Action{Name: "Plan", Model: "opus"}
	claude := agent.Claude{}.ValidModel
	noHaiku := func(model, effort string) error {
		if model == "haiku" {
			return errors.New("no haiku")
		}
		return claude(model, effort)
	}
	tests := []struct {
		name    string
		action  Action
		labels  []string
		valid   func(model, effort string) error
		want    Model
		ignored []string
	}{
		{"nothing set passes nothing", Action{}, nil, claude, Model{}, nil},
		{"the action's", plan, nil, claude, Model{"opus", ""}, nil},
		{"labels over action, field by field", Action{Name: "Plan", Model: "opus", Effort: "low"}, []string{"type:bug", "model:sonnet", "effort:max"}, claude, Model{"sonnet", "max"}, nil},
		{"a label fills what the action leaves unset", plan, []string{"effort:high"}, claude, Model{"opus", "high"}, nil},
		{"first label of a kind wins", Action{}, []string{"model:opus", "model:haiku"}, claude, Model{"opus", ""}, nil},
		{"bad labels are skipped", plan, []string{"model:--x", "effort:huge"}, claude, Model{"opus", ""}, []string{
			`model:--x is ignored: model "--x" is not a model name`,
			`effort:huge is ignored: claude takes an effort of low, medium, high, xhigh, max, not "huge"`,
		}},
		{"a label the provider rejects is skipped", plan, []string{"model:haiku", "effort:high"}, noHaiku, Model{"opus", "high"}, []string{"model:haiku is ignored: no haiku"}},
		{"a model of another provider is skipped", plan, []string{"model:gemini-4-argon"}, claude, Model{"opus", ""}, []string{`model:gemini-4-argon is ignored: claude has no model "gemini-4-argon"`}},
	}
	for _, tt := range tests {
		got, ignored := Pick(tt.action, tt.labels, tt.valid)
		if got != tt.want {
			t.Errorf("%s: Pick = %+v, want %+v", tt.name, got, tt.want)
		}
		if !slices.Equal(ignored, tt.ignored) {
			t.Errorf("%s: ignored = %q, want %q", tt.name, ignored, tt.ignored)
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
