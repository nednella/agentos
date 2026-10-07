package issues_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
)

// flagsOf is the --model and --effort values a session's claude was started with.
func flagsOf(t *testing.T, dir, id string) (model, effort string) {
	t.Helper()
	var args []string
	eventually(t, "claude started", func() bool {
		data, err := os.ReadFile(filepath.Join(dir, strings.ReplaceAll(id, "/", "_")+".args"))
		args = strings.Split(strings.TrimSpace(string(data)), "\n")
		return err == nil && len(data) > 0
	})
	for _, flag := range []string{"--model", "--effort"} {
		v := ""
		if i := slices.Index(args, flag); i >= 0 {
			v = args[i+1]
		}
		if flag == "--model" {
			model = v
		} else {
			effort = v
		}
	}
	return model, effort
}

func TestIssueSessionsGetTheirActionsModel(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})

	tests := []struct {
		name          string
		issue         int
		action        string
		model, effort string
	}{
		{"an action with no model passes none", 11, "", "", ""},
		{"an action's model and effort", 7, "Plan", "opus", "high"},
		{"the default action", 8, "", "opus", ""},
		{"labels win", 12, "", "haiku", "high"},
	}
	for _, tt := range tests {
		s, err := h.StartIssueWith(tt.issue, tt.action)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if s.Model != tt.model || s.Effort != tt.effort {
			t.Errorf("%s: session runs %s %s, want %s %s", tt.name, s.Model, s.Effort, tt.model, tt.effort)
		}
		if model, effort := flagsOf(t, dir, s.ID); model != tt.model || effort != tt.effort {
			t.Errorf("%s: claude got --model %q --effort %q, want %q %q", tt.name, model, effort, tt.model, tt.effort)
		}
	}
}

func TestPlainSessionGetsNoModelFlags(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	s, err := h.NewSession("plain", "")
	if err != nil {
		t.Fatal(err)
	}
	if model, effort := flagsOf(t, dir, s.ID); model != "" || effort != "" || s.Model != "" || s.Effort != "" {
		t.Errorf("claude got %q %q, session %q %q, want none", model, effort, s.Model, s.Effort)
	}
}

func TestSessionsGetClaudesOwnSettings(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	user := os.Getenv("CLAUDE_CONFIG_DIR")
	if err := os.MkdirAll(user, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "settings.json"), []byte(`{"model": "sonnet", "effortLevel": "low"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name          string
		start         func() (sessions.Session, error)
		model, effort string
	}{
		{"a plain session", func() (sessions.Session, error) { return h.NewSession("plain", "") }, "sonnet", "low"},
		{"the action's model wins", func() (sessions.Session, error) { return h.StartIssueWith(8, "") }, "opus", "low"},
		{"labels win", func() (sessions.Session, error) { return h.StartIssueWith(12, "") }, "haiku", "high"},
	}
	for _, tt := range tests {
		s, err := tt.start()
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if model, effort := flagsOf(t, dir, s.ID); model != tt.model || effort != tt.effort || s.Model != tt.model || s.Effort != tt.effort {
			t.Errorf("%s: claude got %q %q, session %q %q, want %s %s", tt.name, model, effort, s.Model, s.Effort, tt.model, tt.effort)
		}
	}
}

func TestModelSurvivesARestartAndReachesTheTally(t *testing.T) {
	apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	s, err := h.StartIssue(8)
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "Notification", `{"message":"Claude needs your permission"}`)
	eventually(t, "a wait with the model", func() bool {
		st, _ := h.Stats(7)
		return len(st.Recent) == 1 && st.Recent[0].Model == "opus"
	})

	again := h.Restart(t)
	got, ok := again.Session(s.ID)
	if !ok || got.Model != "opus" || got.Effort != "" {
		t.Errorf("after a restart: %+v, found %v", got, ok)
	}

	if err := again.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the ended row keeps its model", func() bool {
		ended, ok := again.Session(s.ID)
		return ok && ended.State == "ended" && ended.Model == "opus"
	})
}

func TestAnotherAgentIsNotGivenAModel(t *testing.T) {
	h := newHarness(t)
	s, err := h.StartIssue(8)
	if err != nil {
		t.Fatal(err)
	}
	if s.Model != "" || s.Effort != "" {
		t.Errorf("a bash session claims %q %q", s.Model, s.Effort)
	}
}
