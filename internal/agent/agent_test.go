package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/prompts"
)

func TestClaudeCommand(t *testing.T) {
	argv := Claude{Exe: "/opt/it's here/agentos"}.Command("p/1", Launch{})
	if len(argv) != 3 || argv[0] != "claude" || argv[1] != "--settings" {
		t.Fatalf("argv = %q", argv)
	}
	var settings struct {
		Hooks map[string][]struct {
			Hooks []struct{ Type, Command string }
		}
	}
	if err := json.Unmarshal([]byte(argv[2]), &settings); err != nil {
		t.Fatal(err)
	}
	if len(settings.Hooks) != len(claudeEvents) {
		t.Errorf("hooks for %d events, want %d", len(settings.Hooks), len(claudeEvents))
	}
	got := settings.Hooks["Stop"][0].Hooks[0]
	if got.Type != "command" || !strings.HasSuffix(got.Command, " hook Stop") || !strings.HasPrefix(got.Command, `'/opt/it'\''s here/agentos'`) {
		t.Errorf("Stop hook = %+v", got)
	}
}

func TestClaudeGrantsDirs(t *testing.T) {
	argv := Claude{Exe: "/x", Dirs: []string{"/data", "/local"}}.Command("p/1", Launch{})
	var settings struct {
		Permissions struct{ AdditionalDirectories []string }
	}
	if err := json.Unmarshal([]byte(argv[2]), &settings); err != nil {
		t.Fatal(err)
	}
	if got := settings.Permissions.AdditionalDirectories; !slices.Equal(got, []string{"/data", "/local"}) {
		t.Errorf("additionalDirectories = %q", got)
	}
}

func TestClaudeLaunch(t *testing.T) {
	tests := []struct {
		name string
		l    Launch
		want []string
	}{
		{"none", Launch{}, nil},
		{"model", Launch{Model: "opus"}, []string{"--model", "opus"}},
		{"effort", Launch{Effort: "high"}, []string{"--effort", "high"}},
		{"both", Launch{Model: "sonnet", Effort: "medium"}, []string{"--model", "sonnet", "--effort", "medium"}},
		{"resume", Launch{Resume: "abc"}, []string{"--resume", "abc"}},
		{"chat", Launch{Chat: true}, []string{"--permission-mode", "plan", "--allowedTools", "Bash(agentos new:*),Bash(agentos note:*),Bash(gh issue create:*)"}},
	}
	for _, tt := range tests {
		argv := Claude{Exe: "/x"}.Command("p/1", tt.l)
		if got := argv[3:]; !slices.Equal(got, tt.want) && (len(got) != 0 || len(tt.want) != 0) {
			t.Errorf("%s: flags = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPlainIgnoresLaunch(t *testing.T) {
	argv := Plain{Argv: []string{"bash"}}.Command("p/1", Launch{Model: "opus", Effort: "high", Resume: "abc", Chat: true})
	if !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{{"", "claude"}, {"claude", "claude"}, {"bash", "bash"}, {"bash -l", "bash"}}
	for _, tt := range tests {
		if got := New(tt.name, "/x").Command("p/1", Launch{})[0]; got != tt.want {
			t.Errorf("New(%q) runs %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestBrowserPromptNamesTheCommands(t *testing.T) {
	for _, want := range []string{"agentos browser help", "agentos browser open", "agentos browser screenshot", "agentos browser close"} {
		if !strings.Contains(prompts.BrowserSession(), want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
}

func TestClaudeHeadless(t *testing.T) {
	argv := ClaudeHeadless.Argv("find things")
	if want := []string{"claude", "-p", "find things", "--allowedTools", "WebSearch WebFetch Bash(agentos digest add:*)"}; !slices.Equal(argv, want) {
		t.Errorf("argv = %q, want %q", argv, want)
	}
	env := ClaudeHeadless.Env([]string{"PATH=/bin", "ANTHROPIC_API_KEY=k", "DEPLOY_TOKEN=s", "AGENTOS_SESSION=main/9"})
	if want := []string{"PATH=/bin", "ANTHROPIC_API_KEY=k"}; !slices.Equal(env, want) {
		t.Errorf("env = %q, want %q", env, want)
	}
}

func TestClaudeFill(t *testing.T) {
	write := func(t *testing.T, path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	tests := []struct {
		name                string
		user, shared, local string
		l, want             Launch
	}{
		{"no settings", "", "", "", Launch{}, Launch{}},
		{"the user's", `{"model": "opus", "effortLevel": "medium"}`, "", "", Launch{}, Launch{Model: "opus", Effort: "medium"}},
		{"the project's over the user's", `{"model": "opus", "effortLevel": "medium"}`, `{"model": "sonnet"}`, "", Launch{}, Launch{Model: "sonnet", Effort: "medium"}},
		{"local over shared", "", `{"model": "sonnet"}`, `{"model": "haiku"}`, Launch{}, Launch{Model: "haiku"}},
		{"a set value stays", `{"model": "opus", "effortLevel": "medium"}`, "", "", Launch{Effort: "max", Resume: "abc"}, Launch{Model: "opus", Effort: "max", Resume: "abc"}},
		{"the model's own effort", `{"model": "claude-opus-5-5", "effortLevel": "medium", "modelSettings": {"claude-opus-5-5": {"effortLevel": "high"}}}`, "", "", Launch{}, Launch{Model: "claude-opus-5-5", Effort: "high"}},
		{"another model's effort is not used", `{"model": "opus", "effortLevel": "medium", "modelSettings": {"claude-sonnet-5-5": {"effortLevel": "high"}}}`, "", "", Launch{}, Launch{Model: "opus", Effort: "medium"}},
		{"a broken file is skipped", `{"model": "opus"}`, `{not json`, "", Launch{}, Launch{Model: "opus"}},
	}
	for _, tt := range tests {
		user, dir := t.TempDir(), t.TempDir()
		for path, body := range map[string]string{
			filepath.Join(user, "settings.json"):                 tt.user,
			filepath.Join(dir, ".claude", "settings.json"):       tt.shared,
			filepath.Join(dir, ".claude", "settings.local.json"): tt.local,
		} {
			if body != "" {
				write(t, path, body)
			}
		}
		if got := (Claude{ConfigDir: user}).Fill(dir, tt.l); got != tt.want {
			t.Errorf("%s: Fill = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestPlainFillsNothing(t *testing.T) {
	if got := (Plain{}).Fill(t.TempDir(), Launch{Effort: "high"}); got != (Launch{Effort: "high"}) {
		t.Errorf("Fill = %+v", got)
	}
}
