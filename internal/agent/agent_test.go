package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/prompts"
)

func TestClaudeCommand(t *testing.T) {
	argv := Claude{Exe: "/opt/it's here/agentos"}.Command("p/1")
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

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{{"", "claude"}, {"claude", "claude"}, {"bash", "bash"}, {"bash -l", "bash"}}
	for _, tt := range tests {
		if got := New(tt.name, "/x").Command("p/1")[0]; got != tt.want {
			t.Errorf("New(%q) runs %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestBrowserPromptNamesTheCommands(t *testing.T) {
	for _, want := range []string{"agentos browser help", "agentos browser screenshot", "agentos show"} {
		if !strings.Contains(prompts.BrowserSession(), want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
}
