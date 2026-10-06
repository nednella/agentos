package project

import "testing"

func TestDefault(t *testing.T) {
	tests := []struct {
		name string
		c    Config
		want Model
	}{
		{"unset", Config{}, Model{}},
		{"model only", Config{Model: "opus"}, Model{"opus", ""}},
		{"both", Config{Model: "haiku", Effort: "low"}, Model{"haiku", "low"}},
	}
	for _, tt := range tests {
		if got := tt.c.Default(); got != tt.want {
			t.Errorf("%s: Default() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestPick(t *testing.T) {
	base := Model{"sonnet", "medium"}
	project := Project{Model: "haiku", Effort: "low"}
	plan := Action{Name: "Plan", Model: "opus"}
	tests := []struct {
		name   string
		p      Project
		base   Model
		action Action
		labels []string
		want   Model
	}{
		{"nothing set passes nothing", Project{}, Model{}, Action{}, nil, Model{}},
		{"the config's", Project{}, base, Action{}, nil, Model{"sonnet", "medium"}},
		{"project over config", project, base, Action{}, nil, Model{"haiku", "low"}},
		{"action over project", project, base, plan, nil, Model{"opus", "low"}},
		{"action over config, field by field", Project{}, base, Action{Effort: "high"}, nil, Model{"sonnet", "high"}},
		{"labels over action", project, base, plan, []string{"type:bug", "model:sonnet", "effort:max"}, Model{"sonnet", "max"}},
		{"first label of a kind wins", Project{}, Model{}, Action{}, []string{"model:opus", "model:haiku"}, Model{"opus", ""}},
		{"bad labels are skipped", Project{}, base, plan, []string{"model:--x", "effort:huge"}, Model{"opus", "medium"}},
	}
	for _, tt := range tests {
		if got := tt.p.Pick(tt.base, tt.action, tt.labels); got != tt.want {
			t.Errorf("%s: Pick = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
