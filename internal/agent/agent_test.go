package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/session"
)

func TestClaudeCommand(t *testing.T) {
	argv := Claude{Exe: "/opt/it's here/agentos"}.Command(Launch{})
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

func TestClaudeLetsAChatUseTheBrowser(t *testing.T) {
	tests := []struct {
		name string
		l    Launch
		want bool
	}{
		{"chat with a browser", Launch{Chat: true, MCP: MCPServer{Command: "npx"}}, true},
		{"chat without one", Launch{Chat: true}, false},
		{"task with a browser", Launch{MCP: MCPServer{Command: "npx"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argv := Claude{Exe: "/x"}.Command(tt.l)
			type group struct {
				Matcher string
				Hooks   []struct{ Command string }
			}
			var settings struct{ Hooks map[string][]group }
			if err := json.Unmarshal([]byte(argv[2]), &settings); err != nil {
				t.Fatal(err)
			}
			groups := settings.Hooks["PermissionRequest"]
			i := slices.IndexFunc(groups, func(g group) bool { return g.Matcher == "mcp__browser__.*" })
			if (i >= 0) != tt.want {
				t.Fatalf("PermissionRequest = %+v", groups)
			}
			if i < 0 {
				return
			}
			out, err := exec.Command("sh", "-c", groups[i].Hooks[0].Command).Output()
			if err != nil {
				t.Fatal(err)
			}
			var answer struct {
				HookSpecificOutput struct {
					HookEventName string
					Decision      struct{ Behavior string }
				}
			}
			if err := json.Unmarshal(out, &answer); err != nil || answer.HookSpecificOutput.HookEventName != "PermissionRequest" || answer.HookSpecificOutput.Decision.Behavior != "allow" {
				t.Errorf("hook output = %s", out)
			}
		})
	}
}

func TestClaudeGrantsDirs(t *testing.T) {
	argv := Claude{Exe: "/x", Dirs: []string{"/data", "/local"}}.Command(Launch{})
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
		{"chat", Launch{Chat: true}, []string{"--permission-mode", "plan", "--allowedTools", "Bash(agentos issue:*),Bash(agentos new:*),Bash(agentos note:*),Bash(agentos refresh:*),Bash(agentos browser:*),Bash(gh issue create:*),Bash(gh issue comment:*),Bash(gh issue edit:*)"}},
		{"browser", Launch{MCP: MCPServer{Command: "npx", Args: []string{"-y", "x"}, Env: map[string]string{"A": "1"}, Denied: []string{"new_page", "close_page"}}},
			[]string{"--mcp-config", `{"mcpServers":{"browser":{"command":"/x","args":["mcp-browser","--","npx","-y","x"],"env":{"A":"1"}}}}`, "--disallowedTools", "mcp__browser__new_page mcp__browser__close_page"}},
		{"prompt", Launch{Prompt: "where you run"}, []string{"--append-system-prompt", "where you run"}},
		{"start", Launch{Start: "-v is a flag"}, []string{"--", "-v is a flag"}},
		{"all, in order", Launch{Chat: true, MCP: MCPServer{Command: "npx"}, Prompt: "p", Start: "s"},
			[]string{"--permission-mode", "plan", "--allowedTools", "Bash(agentos issue:*),Bash(agentos new:*),Bash(agentos note:*),Bash(agentos refresh:*),Bash(agentos browser:*),Bash(gh issue create:*),Bash(gh issue comment:*),Bash(gh issue edit:*)",
				"--mcp-config", `{"mcpServers":{"browser":{"command":"/x","args":["mcp-browser","--","npx"],"env":null}}}`, "--disallowedTools", "", "--append-system-prompt", "p", "--", "s"}},
	}
	for _, tt := range tests {
		argv := Claude{Exe: "/x"}.Command(tt.l)
		if got := argv[3:]; !slices.Equal(got, tt.want) && (len(got) != 0 || len(tt.want) != 0) {
			t.Errorf("%s: flags = %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestPlainIgnoresLaunch(t *testing.T) {
	l := Launch{Model: "opus", Effort: "high", Resume: "abc", Chat: true, Prompt: "p", Start: "s", MCP: MCPServer{Command: "npx"}, TmpDir: "/tmp/s"}
	if argv := (Plain{Argv: []string{"bash"}}).Command(l); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
	if env := (Plain{Argv: []string{"bash"}}).Env(l); env != nil {
		t.Errorf("env = %q", env)
	}
}

func TestPlainCanNothing(t *testing.T) {
	for _, c := range []Capability{Resume, StartArg, ModelEffort, Browser} {
		if (Plain{}).Can(c) {
			t.Errorf("a plain agent can %d", c)
		}
		if !(Claude{}).Can(c) {
			t.Errorf("claude cannot %d", c)
		}
	}
	if _, ok := (Plain{}).ParseEvent("Stop", strings.NewReader("{}")); ok {
		t.Error("a plain agent's hook was recorded")
	}
}

func TestClaudeEnv(t *testing.T) {
	if got := (Claude{}).Env(Launch{TmpDir: "/tmp/s"}); !slices.Equal(got, []string{"CLAUDE_CODE_TMPDIR=/tmp/s"}) {
		t.Errorf("env = %q", got)
	}
}

func TestNames(t *testing.T) {
	for _, name := range []string{"claude", "bash", "bash -l"} {
		if got := New(name, "/x").Name(); got != name {
			t.Errorf("New(%q).Name() = %q", name, got)
		}
	}
}

func TestClaudeParseEvent(t *testing.T) {
	tests := []struct {
		name  string
		event string
		body  string
		want  session.Event
	}{
		{"permission notification", "Notification", `{"message":"Claude needs your permission to use Bash","notification_type":"permission_prompt"}`,
			session.Event{Name: "Notification", NotificationType: "permission_prompt", Message: "Claude needs your permission to use Bash", Detail: "Claude needs your permission to use Bash"}},
		{"tool with command", "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"echo hi","description":"Print hi"}}`,
			session.Event{Name: "PreToolUse", Detail: "Bash echo hi", Tool: "Bash", Command: "echo hi"}},
		{"tool without known input", "PostToolUse", `{"tool_name":"Task","tool_input":{"x":1}}`,
			session.Event{Name: "PostToolUse", Detail: "Task", Tool: "Task"}},
		{"prompt on many lines", "UserPromptSubmit", `{"prompt":"fix\n  the   bug"}`,
			session.Event{Name: "UserPromptSubmit", Detail: "fix the bug"}},
		{"working directory", "Stop", `{"cwd":"/tmp/p"}`, session.Event{Name: "Stop", Cwd: "/tmp/p"}},
		{"conversation", "Stop", `{"session_id":"abc"}`, session.Event{Name: "Stop", Conversation: "abc"}},
		{"transcript", "Stop", `{"transcript_path":"/t/abc.jsonl"}`, session.Event{Name: "Stop", Transcript: "/t/abc.jsonl"}},
		{"not json", "Stop", `garbage`, session.Event{Name: "Stop"}},
		{"reply ends in a question", "Stop", `{"last_assistant_message":"Done.\n\n**Do you want me to push?**\n"}`,
			session.Event{Name: "Stop", Detail: "Done. **Do you want me to push?**", Asks: true}},
		{"question inside the reply", "Stop", `{"last_assistant_message":"Why? The test was flaky. I fixed it."}`,
			session.Event{Name: "Stop", Detail: "Why? The test was flaky. I fixed it."}},
		{"empty", "SessionStart", ``, session.Event{Name: "SessionStart"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := (Claude{}).ParseEvent(tt.event, strings.NewReader(tt.body))
			if !ok || got != tt.want {
				t.Errorf("got %+v, %v, want %+v", got, ok, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{{"", "claude"}, {"claude", "claude"}, {"bash", "bash"}, {"bash -l", "bash"}}
	for _, tt := range tests {
		if got := New(tt.name, "/x").Command(Launch{})[0]; got != tt.want {
			t.Errorf("New(%q) runs %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestRegistryProvider(t *testing.T) {
	tests := []struct {
		exe, command string
		want         Provider
	}{
		{"/x", "", Claude{Exe: "/x", ConfigDir: claudeConfigDir(), Dirs: []string{"/d"}}},
		{"/x", "claude", Claude{Exe: "/x", ConfigDir: claudeConfigDir(), Dirs: []string{"/d"}}},
		{"/x", "bash -l", Plain{Argv: []string{"bash", "-l"}}},
		{"", "", Plain{Argv: []string{"claude"}}},
		{"", "claude", Plain{Argv: []string{"claude"}}},
		{"", "bash", Plain{Argv: []string{"bash"}}},
	}
	for _, tt := range tests {
		if got := (Registry{Exe: tt.exe, Dirs: []string{"/d"}}).Provider(tt.command); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Registry{Exe: %q}.Provider(%q) = %#v, want %#v", tt.exe, tt.command, got, tt.want)
		}
	}
}

func TestCheckName(t *testing.T) {
	for _, name := range []string{"", "claude"} {
		if err := CheckName(name); err != nil {
			t.Errorf("CheckName(%q) = %v", name, err)
		}
	}
	for _, name := range []string{"agy", "bash", "claude --resume"} {
		if err := CheckName(name); err == nil || !strings.Contains(err.Error(), "the agents are claude") {
			t.Errorf("CheckName(%q) = %v", name, err)
		}
	}
}

func TestBrowserPromptNamesTheCommands(t *testing.T) {
	for _, want := range []string{"agentos browser help", "agentos browser open", "agentos browser screenshot", "agentos browser close"} {
		if !strings.Contains(prompts.Browser(), want) {
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
		if got := (Claude{ConfigDir: user}).Fill(dir, tt.l); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Fill = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestPlainFillsNothing(t *testing.T) {
	if got := (Plain{}).Fill(t.TempDir(), Launch{Effort: "high"}); !reflect.DeepEqual(got, Launch{Effort: "high"}) {
		t.Errorf("Fill = %+v", got)
	}
}
