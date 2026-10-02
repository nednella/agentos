package project

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
	custom := Project{Commands: map[string]string{"ready": "/ship #{n}", "note": ""}}
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
		{"custom idea", Project{Commands: map[string]string{"idea": "/think {n}"}}.IssueCommand("idea", 3), "/think 3"},
		{"custom ready", custom.IssueCommand("ready", 4), "/ship #4"},
		{"unset plan keeps default", custom.IssueCommand("plan", 6), "/investigate 6"},
		{"default note", Project{}.NoteCommand("buy milk"), "buy milk"},
		{"custom empty note", custom.NoteCommand("buy milk"), ""},
		{"note template", Project{Commands: map[string]string{"note": "/plan {text}"}}.NoteCommand("x"), "/plan x"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}

func TestSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "config.yaml")
	want := Config{Agent: "claude", Projects: []Project{
		{Name: "a", Dir: "/srv/a", Commands: map[string]string{"ready": "/x {n}"}, Lanes: map[string]string{"go": "ready"}},
		{Name: "b", Dir: "/srv/b"},
	}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestBranchAndGuard(t *testing.T) {
	if got := (Project{}).BranchFor(394); got != "issue-394" {
		t.Errorf("default branch = %q", got)
	}
	if got := (Project{Branch: "ned/{n}-fix"}).BranchFor(7); got != "ned/7-fix" {
		t.Errorf("custom branch = %q", got)
	}
	if len(Project{}.GuardPatterns()) == 0 {
		t.Error("no default guard")
	}
	if got := (Project{Guard: []string{"x"}}).GuardPatterns(); len(got) != 1 || got[0] != "x" {
		t.Errorf("custom guard = %v", got)
	}
}

func TestBrowserAndDigest(t *testing.T) {
	off := false
	tests := []struct {
		name            string
		p               Project
		browser, digest bool
	}{
		{"defaults", Project{}, true, true},
		{"browser off", Project{Browser: &off}, false, true},
		{"digest off", Project{Digest: "off"}, true, false},
		{"digest weekly", Project{Digest: "weekly"}, true, true},
	}
	for _, tt := range tests {
		if tt.p.BrowserOn() != tt.browser || tt.p.DigestOn() != tt.digest {
			t.Errorf("%s: browser %v digest %v", tt.name, tt.p.BrowserOn(), tt.p.DigestOn())
		}
	}
}

func TestLoadDataDir(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	path := filepath.Join(t.TempDir(), "c.yaml")
	if err := os.WriteFile(path, []byte("data_dir: ~/Notes/agentos\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil || cfg.DataDir != filepath.Join(home, "Notes/agentos") {
		t.Errorf("data_dir = %q, %v", cfg.DataDir, err)
	}
}
