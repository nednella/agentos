package project

import (
	"testing"
	"time"
)

func TestSection(t *testing.T) {
	p := Project{QueueSections: []Section{
		{Name: "Inbox"},
		{Name: "Ready", Labels: []string{"ready", "go"}},
		{Name: "Plan", Labels: []string{"needs-plan"}},
	}}
	tests := []struct {
		name   string
		p      Project
		labels []string
		want   string
		at     int
	}{
		{"no labels take the section without labels", p, nil, "Inbox", 0},
		{"any label of a section", p, []string{"type:bug", "go"}, "Ready", 1},
		{"first section wins", p, []string{"needs-plan", "ready"}, "Ready", 1},
		{"no section takes it", p, []string{"roadmap"}, OtherSection, 3},
		{"labels never take the section without labels", Project{QueueSections: p.QueueSections[:1]}, []string{"ready"}, OtherSection, 1},
		{"no sections", Project{}, []string{"ready"}, "", 0},
		{"no sections, no labels", Project{}, nil, "", 0},
	}
	for _, tt := range tests {
		if at, s := tt.p.Section(tt.labels); s.Name != tt.want || at != tt.at {
			t.Errorf("%s: Section(%v) = %d %q, want %d %q", tt.name, tt.labels, at, s.Name, tt.at, tt.want)
		}
	}
}

func TestActionList(t *testing.T) {
	own := Section{Actions: []Action{{Name: "Plan"}, {Name: "Work"}}}
	if got := own.ActionList(); len(got) != 2 || got[0].Name != "Plan" {
		t.Errorf("a section's own actions = %+v", got)
	}
	got := Section{}.ActionList()
	if len(got) != 1 || got[0].Name != "Start" || got[0].Render(7, "Fix it") != "Work on issue #7: Fix it" {
		t.Errorf("without actions: %+v", got)
	}
}

func TestRender(t *testing.T) {
	tests := []struct{ command, want string }{
		{"/work {n}", "/work 4"},
		{"/plan {n}: {title}", "/plan 4: Add {n} things"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := (Action{Command: tt.command}).Render(4, "Add {n} things"); got != tt.want {
			t.Errorf("Render(%q) = %q, want %q", tt.command, got, tt.want)
		}
	}
}

func TestNoteCommand(t *testing.T) {
	if got := (Project{}).NoteCommand("buy milk"); got != "buy milk" {
		t.Errorf("default = %q", got)
	}
	if got := (Project{NoteSessionCommand: "/plan {text}"}).NoteCommand("x"); got != "/plan x" {
		t.Errorf("template = %q", got)
	}
}

func TestSendsPrompt(t *testing.T) {
	for _, tt := range []struct {
		mode string
		want bool
	}{{"", true}, {"auto", true}, {"manual", false}} {
		if got := (Project{SessionPromptSend: tt.mode}).SendsPrompt(); got != tt.want {
			t.Errorf("SendsPrompt(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestBranchFor(t *testing.T) {
	if got := (Project{}).BranchFor(394); got != "" {
		t.Errorf("default branch = %q, want none", got)
	}
	if got := (Project{Branch: "ned/{n}-fix"}).BranchFor(7); got != "ned/7-fix" {
		t.Errorf("custom branch = %q", got)
	}
}
func TestKeepsAwake(t *testing.T) {
	off, on := false, true
	for _, tt := range []struct {
		name string
		c    Config
		p    Project
		want bool
	}{
		{"default", Config{}, Project{}, true},
		{"config off", Config{KeepAwake: &off}, Project{}, false},
		{"project off", Config{}, Project{KeepAwake: &off}, false},
		{"project on beats config off", Config{KeepAwake: &off}, Project{KeepAwake: &on}, true},
	} {
		if got := tt.c.KeepsAwake(tt.p); got != tt.want {
			t.Errorf("%s: KeepsAwake = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBrowserOn(t *testing.T) {
	off, on := false, true
	for _, tt := range []struct {
		p    Project
		want bool
	}{{Project{}, true}, {Project{Browser: &off}, false}, {Project{Browser: &on}, true}} {
		if got := tt.p.BrowserOn(); got != tt.want {
			t.Errorf("BrowserOn(%+v) = %v, want %v", tt.p, got, tt.want)
		}
	}
}

func TestDigestOn(t *testing.T) {
	for _, tt := range []struct {
		digest string
		want   bool
	}{{"", true}, {"weekly", true}, {"off", false}} {
		if got := (Project{Digest: tt.digest}).DigestOn(); got != tt.want {
			t.Errorf("DigestOn(%q) = %v, want %v", tt.digest, got, tt.want)
		}
	}
}

func TestPRPollEvery(t *testing.T) {
	tests := []struct {
		in      string
		want    time.Duration
		wantErr bool
	}{
		{"", 10 * time.Second, false},
		{"45s", 45 * time.Second, false},
		{"2m", 2 * time.Minute, false},
		{"500ms", 0, true},
		{"soon", 0, true},
	}
	for _, tt := range tests {
		got, err := (Project{PRPoll: tt.in}).PRPollEvery()
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("PRPollEvery(%q) = %v, %v; want %v, err %v", tt.in, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestPRCommands(t *testing.T) {
	p := Project{OnReview: "/address-review {n}", OnChecks: "/fix {n} now"}
	for _, tt := range []struct{ name, got, want string }{
		{"review", p.ReviewCommand(41), "/address-review 41"},
		{"checks", p.ChecksCommand(41), "/fix 41 now"},
		{"review unset", Project{}.ReviewCommand(41), ""},
		{"checks unset", Project{}.ChecksCommand(41), ""},
	} {
		if tt.got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, tt.got, tt.want)
		}
	}
}
