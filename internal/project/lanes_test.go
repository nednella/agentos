package project

import "testing"

func TestLane(t *testing.T) {
	custom := Project{Lanes: map[string]string{"go": "ready", "later": "idea", "blocked": "you"}}
	tests := []struct {
		name   string
		p      Project
		labels []string
		want   string
	}{
		{"default ready wins", Project{}, []string{"idea", "ready", "type:bug"}, "ready"},
		{"default plan", Project{}, []string{"needs-plan"}, "plan"},
		{"default you", Project{}, []string{"needs-human"}, "you"},
		{"no labels", Project{}, nil, "inbox"},
		{"unmapped label", Project{}, []string{"roadmap"}, "idea"},
		{"custom", custom, []string{"later", "go"}, "ready"},
		{"custom ignores defaults", custom, []string{"ready"}, "idea"},
	}
	for _, tt := range tests {
		if got := tt.p.Lane(tt.labels); got != tt.want {
			t.Errorf("%s: Lane(%v) = %q, want %q", tt.name, tt.labels, got, tt.want)
		}
	}
}

func TestCommands(t *testing.T) {
	custom := Project{Commands: map[string]string{"ready": "/ship #{n}"}}
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"default ready", Project{}.IssueCommand("ready", 4), "/work 4"},
		{"default plan", Project{}.IssueCommand("plan", 5), "/investigate 5"},
		{"idea has no default", Project{}.IssueCommand("idea", 5), ""},
		{"inbox has no default", Project{}.IssueCommand("inbox", 5), ""},
		{"you has none", Project{Commands: map[string]string{"you": "x"}}.IssueCommand("you", 5), ""},
		{"custom inbox", Project{Commands: map[string]string{"inbox": "/investigate {n}"}}.IssueCommand("inbox", 9), "/investigate 9"},
		{"default note", Project{}.NoteCommand("buy milk"), "buy milk"},
		{"custom empty note", Project{Commands: map[string]string{"note": ""}}.NoteCommand("buy milk"), ""},
		{"note template", Project{Commands: map[string]string{"note": "/plan {text}"}}.NoteCommand("x"), "/plan x"},
		{"custom idea", Project{Commands: map[string]string{"idea": "/think {n}"}}.IssueCommand("idea", 3), "/think 3"},
		{"custom ready", custom.IssueCommand("ready", 4), "/ship #4"},
		{"unset plan keeps default", custom.IssueCommand("plan", 6), "/investigate 6"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestBranchFor(t *testing.T) {
	if got := (Project{}).BranchFor(394); got != "issue-394" {
		t.Errorf("default branch = %q", got)
	}
	if got := (Project{Branch: "ned/{n}-fix"}).BranchFor(7); got != "ned/7-fix" {
		t.Errorf("custom branch = %q", got)
	}
}
