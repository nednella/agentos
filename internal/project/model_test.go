package project

import "testing"

func TestDefault(t *testing.T) {
	tests := []struct {
		name string
		c    Config
		want Model
	}{
		{"unset", Config{}, Model{"sonnet", "medium"}},
		{"model only", Config{Model: "opus"}, Model{"opus", "medium"}},
		{"both", Config{Model: "haiku", Effort: "low"}, Model{"haiku", "low"}},
	}
	for _, tt := range tests {
		if got := tt.c.Default(); got != tt.want {
			t.Errorf("%s: Default() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestPick(t *testing.T) {
	base := Config{}.Default()
	custom := Project{
		Model: "haiku", Effort: "low",
		Models: map[string]Model{"ready": {Model: "opus"}, "plan": {Effort: "high"}},
	}
	tests := []struct {
		name   string
		p      Project
		lane   string
		labels []string
		want   Model
	}{
		{"no lane takes the default", Project{}, "", nil, Model{"sonnet", "medium"}},
		{"ready", Project{}, "ready", nil, Model{"sonnet", "medium"}},
		{"inbox", Project{}, "inbox", nil, Model{"sonnet", "medium"}},
		{"idea", Project{}, "idea", nil, Model{"sonnet", "medium"}},
		{"plan runs on opus", Project{}, "plan", nil, Model{"opus", "medium"}},
		{"project overrides the default", custom, "", nil, Model{"haiku", "low"}},
		{"project lane overrides the project", custom, "ready", nil, Model{"opus", "low"}},
		{"project lane keeps the built-in model", custom, "plan", nil, Model{"opus", "high"}},
		{"labels win", custom, "ready", []string{"type:bug", "model:sonnet", "effort:max"}, Model{"sonnet", "max"}},
		{"first label of a kind wins", Project{}, "ready", []string{"model:opus", "model:haiku"}, Model{"opus", "medium"}},
		{"bad labels are skipped", Project{}, "ready", []string{"model:--x", "effort:huge"}, Model{"sonnet", "medium"}},
		{"unknown lane", Project{}, "you", nil, Model{"sonnet", "medium"}},
	}
	for _, tt := range tests {
		if got := tt.p.Pick(base, tt.lane, tt.labels); got != tt.want {
			t.Errorf("%s: Pick(%q, %v) = %+v, want %+v", tt.name, tt.lane, tt.labels, got, tt.want)
		}
	}
}
