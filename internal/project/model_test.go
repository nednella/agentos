package project

import "testing"

func TestPick(t *testing.T) {
	plan := Action{Name: "Plan", Model: "opus"}
	tests := []struct {
		name   string
		action Action
		labels []string
		want   Model
	}{
		{"nothing set passes nothing", Action{}, nil, Model{}},
		{"the action's", plan, nil, Model{"opus", ""}},
		{"labels over action, field by field", Action{Name: "Plan", Model: "opus", Effort: "low"}, []string{"type:bug", "model:sonnet", "effort:max"}, Model{"sonnet", "max"}},
		{"a label fills what the action leaves unset", plan, []string{"effort:high"}, Model{"opus", "high"}},
		{"first label of a kind wins", Action{}, []string{"model:opus", "model:haiku"}, Model{"opus", ""}},
		{"bad labels are skipped", plan, []string{"model:--x", "effort:huge"}, Model{"opus", ""}},
	}
	for _, tt := range tests {
		if got := Pick(tt.action, tt.labels); got != tt.want {
			t.Errorf("%s: Pick = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
