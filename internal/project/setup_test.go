package project

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSetUpFillsOnlyEmptyKeys(t *testing.T) {
	p := Project{Name: "a", Dir: "/srv/a", Branch: "feat/{n}", OnChecks: "/fix {n}"}
	if !p.NeedsSetup() {
		t.Fatal("a project with no queue sections does not need setting up")
	}
	p.SetUp()
	if p.NeedsSetup() || p.Branch != "feat/{n}" || p.OnChecks != "/fix {n}" {
		t.Errorf("SetUp changed what the project set: %+v", p)
	}
	if p.CleanupCommand == "" || !strings.Contains(p.OnReview, "PR #{n}") || !strings.Contains(p.OnConflict, "PR #{n}") {
		t.Errorf("SetUp left keys empty: %+v", p)
	}
	inbox := Section{Name: "Inbox", Labels: []string{"*"}, Actions: []Action{{Name: "Work", Command: "/work {n}"}}}
	if !reflect.DeepEqual(p.QueueSections, []Section{inbox}) {
		t.Errorf("queue sections = %+v", p.QueueSections)
	}

	own := []Section{{Name: "Mine", Labels: []string{"bug"}}}
	q := Project{QueueSections: own}
	q.SetUp()
	if !reflect.DeepEqual(q.QueueSections, own) {
		t.Errorf("SetUp replaced the project's sections: %+v", q.QueueSections)
	}
}

func TestSaveSetUpKeepsTheEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	original := `agent_command: claude
projects:
  # the main one
  - name: api
    directory: /srv/api # work
    session_branch_fallback: "feat/{n}" # ours
`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Projects[0].SetUp()
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	for _, want := range []string{"# the main one", "# work", `"feat/{n}" # ours`, "queue_sections:"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("the saved file lacks %q:\n%s", want, got)
		}
	}
	reloaded, err := Load(path)
	if err != nil || !reflect.DeepEqual(reloaded, cfg) {
		t.Errorf("reload = %+v, %v\nwant %+v", reloaded, err, cfg)
	}

	cfg.Projects[0].SetupDismissed = true
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	if reloaded, err = Load(path); err != nil || !reloaded.Projects[0].SetupDismissed {
		t.Errorf("dismissing was not saved: %+v, %v", reloaded.Projects[0], err)
	}
	got, _ = os.ReadFile(path)
	if !strings.Contains(string(got), "# work") {
		t.Errorf("dismissing rewrote the entry:\n%s", got)
	}
}

func TestSetUpOpensAOneLineEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("projects:\n  - {name: api, directory: /srv/api}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Projects[0].SetUp()
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "\n    queue_sections:\n") {
		t.Errorf("the entry stayed on one line:\n%s", got)
	}
}

func TestSaveSetUpFillsAnEmptyQueueSections(t *testing.T) {
	for _, tt := range []struct{ name, queue string }{
		{"empty list", "    queue_sections: [] # none yet\n"},
		{"null", "    queue_sections:\n"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			original := "projects:\n  # the main one\n  - name: api\n    directory: ~/code/api # work\n" + tt.queue + "    note_session_command: x # ours\n"
			if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if err != nil {
				t.Fatal(err)
			}
			cfg.Projects[0].SetUp()
			if err := Save(path, cfg); err != nil {
				t.Fatal(err)
			}
			got, _ := os.ReadFile(path)
			for _, want := range []string{"# the main one", "directory: ~/code/api # work", "x # ours", "- name: Inbox"} {
				if !strings.Contains(string(got), want) {
					t.Errorf("the saved file lacks %q:\n%s", want, got)
				}
			}
			if strings.Count(string(got), "queue_sections:") != 1 {
				t.Errorf("queue_sections appears more than once:\n%s", got)
			}
			if reloaded, err := Load(path); err != nil || len(reloaded.Projects[0].QueueSections) != 1 {
				t.Errorf("reload = %+v, %v", reloaded.Projects, err)
			}
		})
	}
}

func TestSaveLeavesOutAnEmptyAgentCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := Save(path, Config{Projects: []Project{{Name: "api", Dir: "/srv/api"}}}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if strings.Contains(string(got), "agent_command") {
		t.Errorf("a fresh config holds an empty agent_command:\n%s", got)
	}
}
