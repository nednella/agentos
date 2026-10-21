package issues_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
)

const agentQueue = `    queue_sections:
      - name: Ready
        labels: [ready]
        actions:
          - {name: Work, command: "/work {n}", agent: agy}
          - {name: Claude, command: "/work {n}", agent: claude, model: opus}
`

func TestIssueSessionsGetTheirActionsAgent(t *testing.T) {
	claude := apptest.ClaudeOnPath(t)
	agy := apptest.AgyOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude", Queue: agentQueue})
	h.GH.SetIssues(`[
		{"number":7,"title":"By the action","url":"https://x/7","labels":[{"name":"ready"}]},
		{"number":8,"title":"By the label","url":"https://x/8","labels":[{"name":"ready"},{"name":"agent:claude"}]},
		{"number":9,"title":"Unknown label","url":"https://x/9","labels":[{"name":"ready"},{"name":"agent:codex"}]},
		{"number":10,"title":"Model the provider takes","url":"https://x/10","labels":[{"name":"ready"},{"name":"agent:agy"},{"name":"model:gemini-4-argon-medium"}]},
		{"number":11,"title":"Model the provider refuses","url":"https://x/11","labels":[{"name":"ready"},{"name":"model:opus"}]}
	]`)

	tests := []struct {
		name          string
		issue         int
		action        string
		agent         string
		model, effort string
		flag          string
	}{
		{"the action's agent", 7, "", "agy", "", "", ""},
		{"the label over the action", 8, "", "claude", "", "", ""},
		{"a label naming no provider is ignored", 9, "", "agy", "", "", ""},
		{"a model label the provider takes", 10, "", "agy", "gemini-4-argon", "medium", "gemini-4-argon-medium"},
		{"a model label the provider refuses is skipped", 11, "", "agy", "", "", ""},
		{"another action's agent and model", 11, "Claude", "claude", "opus", "", "opus"},
	}
	for _, tt := range tests {
		s, err := h.StartIssueWith(tt.issue, tt.action)
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if s.Agent != tt.agent || s.Model != tt.model || s.Effort != tt.effort {
			t.Errorf("%s: session runs %s %s %s, want %s %s %s", tt.name, s.Agent, s.Model, s.Effort, tt.agent, tt.model, tt.effort)
		}
		dir := claude
		if tt.agent == "agy" {
			dir = agy
		}
		if model, effort := flagsOf(t, dir, s.ID); model != tt.flag || effort != "" {
			t.Errorf("%s: %s got --model %q --effort %q, want %q and none", tt.name, tt.agent, model, effort, tt.flag)
		}
		if err := h.KillSession(s.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAgySessionsGetAgysOwnSettings(t *testing.T) {
	dir := apptest.AgyOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude", Queue: agentQueue})
	// agy keeps its settings one folder up from the config folder the app writes its hooks to.
	settings := filepath.Join(filepath.Dir(os.Getenv("AGENTOS_AGY_CONFIG_DIR")), "antigravity-cli", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"model": "Gemini 3.8 Flash (Low)"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	h.GH.SetIssues(`[
		{"number":7,"title":"By the settings","url":"https://x/7","labels":[{"name":"ready"}]},
		{"number":10,"title":"Model label","url":"https://x/10","labels":[{"name":"ready"},{"name":"model:gemini-4-argon-medium"}]},
		{"number":12,"title":"Effort label","url":"https://x/12","labels":[{"name":"ready"},{"name":"effort:xhigh"}]}
	]`)

	tests := []struct {
		name                  string
		issue                 int
		model, effort         string
		modelFlag, effortFlag string
	}{
		{"agy's default", 7, "gemini-3.8-flash", "low", "gemini-3.8-flash-low", ""},
		{"the label's model and its effort", 10, "gemini-4-argon", "medium", "gemini-4-argon-medium", ""},
		{"an effort no id ends in", 12, "gemini-3.8-flash", "xhigh", "gemini-3.8-flash", "xhigh"},
	}
	for _, tt := range tests {
		s, err := h.StartIssueWith(tt.issue, "")
		if err != nil {
			t.Fatalf("%s: %v", tt.name, err)
		}
		if s.Model != tt.model || s.Effort != tt.effort {
			t.Errorf("%s: session runs %q %q, want %q %q", tt.name, s.Model, s.Effort, tt.model, tt.effort)
		}
		if model, effort := flagsOf(t, dir, s.ID); model != tt.modelFlag || effort != tt.effortFlag {
			t.Errorf("%s: agy got --model %q --effort %q, want %q %q", tt.name, model, effort, tt.modelFlag, tt.effortFlag)
		}
	}
}
