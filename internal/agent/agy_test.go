package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/session"
)

func TestAgyCommand(t *testing.T) {
	tests := []struct {
		name string
		l    Launch
		want []string
	}{
		{"none", Launch{}, nil},
		{"the effort joins the model", Launch{Model: "gemini-4-argon", Effort: "high"}, []string{"--model", "gemini-4-argon-high"}},
		{"an effort no id ends in", Launch{Model: "gemini-4-argon", Effort: "xhigh"}, []string{"--model", "gemini-4-argon", "--effort", "xhigh"}},
		{"an effort alone", Launch{Effort: "low"}, []string{"--effort", "low"}},
		{"resume", Launch{Resume: "abc"}, []string{"--conversation", "abc"}},
		{"chat", Launch{Chat: true}, []string{"--mode", "plan"}},
		{"start", Launch{Start: "say hi"}, []string{"-i", "say hi"}},
		{"the prompt and the browser go through the global files", Launch{Prompt: "p", MCP: MCPServer{Command: "npx"}}, nil},
		{"all, in order", Launch{Model: "m", Resume: "abc", Chat: true, Start: "s"}, []string{"--model", "m", "--conversation", "abc", "--mode", "plan", "-i", "s"}},
	}
	for _, tt := range tests {
		argv := Agy{}.Command(tt.l)
		if argv[0] != "agy" || !slices.Equal(argv[1:], tt.want) && (len(argv) != 1 || len(tt.want) != 0) {
			t.Errorf("%s: argv = %q, want agy %q", tt.name, argv, tt.want)
		}
	}
	if argv := (Agy{Dirs: []string{"/data", "/local"}}).Command(Launch{}); !slices.Equal(argv, []string{"agy", "--add-dir", "/data", "--add-dir", "/local"}) {
		t.Errorf("argv with dirs = %q", argv)
	}
}

func TestAgyEnvAndPrepareHoldThePromptForTheStartHook(t *testing.T) {
	tmp := t.TempDir()
	a := Agy{} // Writes is off: a build that may not write the global files still writes the prompt
	l := Launch{Prompt: "where you run", TmpDir: tmp}
	if err := a.Prepare(l); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(tmp, "prompt.md")
	if data, err := os.ReadFile(file); err != nil || string(data) != "where you run" {
		t.Errorf("prompt file = %q, %v", data, err)
	}
	if got := a.Env(l); !slices.Equal(got, []string{"AGENTOS_PROMPT=" + file}) {
		t.Errorf("env = %q", got)
	}
	if got := a.Env(Launch{Prompt: "p", TmpDir: tmp, Chat: true}); !slices.Equal(got, []string{"AGENTOS_PROMPT=" + file, "AGENTOS_CHAT=1"}) {
		t.Errorf("env of a chat = %q", got)
	}
	// A resumed conversation keeps the message the hook injected, so it gets no prompt again.
	resumed := Launch{Prompt: "p", TmpDir: t.TempDir(), Resume: "abc"}
	if err := a.Prepare(resumed); err != nil {
		t.Fatal(err)
	}
	if got := a.Env(resumed); got != nil {
		t.Errorf("env of a resume = %q", got)
	}
	if _, err := os.Stat(filepath.Join(resumed.TmpDir, "prompt.md")); err == nil {
		t.Error("a resume wrote a prompt file")
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	return v
}

func TestAgyPrepareWritesTheGlobalHooksAndMCPEntry(t *testing.T) {
	dir := t.TempDir()
	hooks, servers := filepath.Join(dir, "hooks.json"), filepath.Join(dir, "mcp_config.json")
	// The owner's own entries, and an empty MCP file like a fresh agy leaves, stay as they are.
	if err := os.WriteFile(hooks, []byte(`{"lint":{"PostToolUse":[{"matcher":"run_command","hooks":[{"command":"./lint.sh"}]}]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(servers, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	a := Agy{ConfigDir: dir, Installed: "/Users/me/.local/bin/agentos", Writes: true}
	if err := a.Prepare(Launch{}); err != nil {
		t.Fatal(err)
	}
	got := readJSON(t, hooks)
	if got["lint"] == nil {
		t.Error("the owner's own hook is gone")
	}
	mine, _ := got["agentos"].(map[string]any)
	for _, ev := range agyEvents {
		handlers, _ := mine[ev].([]any)
		if len(handlers) != 1 {
			t.Fatalf("hook for %s = %v", ev, mine[ev])
		}
		h, _ := handlers[0].(map[string]any)
		if ev == "PreToolUse" || ev == "PostToolUse" {
			if h["matcher"] != "*" {
				t.Errorf("%s matcher = %v", ev, h["matcher"])
			}
			group, _ := h["hooks"].([]any)
			h, _ = group[0].(map[string]any)
		}
		if h["type"] != "command" || h["command"] != "'/Users/me/.local/bin/agentos' hook "+ev {
			t.Errorf("%s handler = %v", ev, h)
		}
	}
	mcp := readJSON(t, servers)["mcpServers"].(map[string]any)["browser"].(map[string]any)
	want := map[string]any{"command": "/Users/me/.local/bin/agentos", "args": []any{"mcp-browser"}, "disabledTools": []any{"new_page", "close_page"}}
	if !reflect.DeepEqual(mcp, want) {
		t.Errorf("browser server = %v, want %v", mcp, want)
	}

	// A second start leaves the files alone; an entry that differs is written again.
	before, _ := os.Stat(hooks)
	if err := a.Prepare(Launch{}); err != nil {
		t.Fatal(err)
	}
	if after, _ := os.Stat(hooks); !after.ModTime().Equal(before.ModTime()) {
		t.Error("an unchanged hooks.json was written again")
	}
	moved := Agy{ConfigDir: dir, Installed: "/opt/agentos", Writes: true}
	if err := moved.Prepare(Launch{}); err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(hooks); !strings.Contains(string(data), "'/opt/agentos' hook Stop") || strings.Contains(string(data), "/Users/me") {
		t.Errorf("hooks.json after the command moved = %s", data)
	}
}

func TestAgyPrepareLeavesTheOwnersFilesToTheInstalledApp(t *testing.T) {
	dir := t.TempDir()
	a := Agy{ConfigDir: dir, Installed: "/x", Writes: false}
	if err := a.Prepare(Launch{}); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("a build that may not write agy's config wrote %v", entries)
	}
}

func TestNewAgyWritesOnlyWhereTheEnvironmentPoints(t *testing.T) {
	t.Setenv(agyConfigDirEnv, "/tmp/agy-test")
	if a := NewAgy(nil); a.ConfigDir != "/tmp/agy-test" || !a.Writes {
		t.Errorf("with the env set: %+v", a)
	}
	t.Setenv(agyConfigDirEnv, "")
	home, _ := os.UserHomeDir()
	if a := NewAgy(nil); a.ConfigDir != filepath.Join(home, ".gemini", "config") || a.Writes || a.Installed != filepath.Join(home, ".local", "bin", "agentos") {
		t.Errorf("a test binary without the env: %+v", a)
	}
}

func TestAgyFill(t *testing.T) {
	models := func() []agyModel {
		return []agyModel{{"gemini-4-argon-medium", "Gemini 4 Argon (Medium)"}, {"gemini-3.8-flash-low", "Gemini 3.8 Flash (Low)"}, {"claude-opus-5-5-low", "Claude Opus 5.5 (Low)"}}
	}
	settings := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(settings, []byte(`{"model": "Gemini 4 Argon (Medium)"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	claude := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(claude, []byte(`{"model": "Claude Opus 5.5 (Low)"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	unlisted := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(unlisted, []byte(`{"model": "Gemini 9"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		a    Agy
		l    Launch
		want Launch
	}{
		{"agy's default", Agy{Settings: settings, Models: models}, Launch{}, Launch{Model: "gemini-4-argon", Effort: "medium"}},
		{"a set model keeps the default's effort", Agy{Settings: settings, Models: models}, Launch{Model: "gemini-3.8-flash"}, Launch{Model: "gemini-3.8-flash", Effort: "medium"}},
		{"a set effort stays", Agy{Settings: settings, Models: models}, Launch{Effort: "xhigh", Resume: "abc"}, Launch{Model: "gemini-4-argon", Effort: "xhigh", Resume: "abc"}},
		{"a full id is split", Agy{Settings: settings, Models: models}, Launch{Model: "gemini-3.8-flash-low"}, Launch{Model: "gemini-3.8-flash", Effort: "low"}},
		{"a set effort wins over the id's", Agy{Settings: settings, Models: models}, Launch{Model: "gemini-3.8-flash-low", Effort: "high"}, Launch{Model: "gemini-3.8-flash", Effort: "high"}},
		{"a Claude model", Agy{Settings: claude, Models: models}, Launch{}, Launch{}},
		{"a name agy does not list", Agy{Settings: unlisted, Models: models}, Launch{}, Launch{}},
		{"agy cannot say", Agy{Settings: settings, Models: func() []agyModel { return nil }}, Launch{}, Launch{}},
		{"no settings", Agy{Settings: filepath.Join(t.TempDir(), "none.json"), Models: models}, Launch{}, Launch{}},
	}
	for _, tt := range tests {
		if got := tt.a.Fill(t.TempDir(), tt.l); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Fill = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestNewAgyFindsTheSettingsBesideTheConfig(t *testing.T) {
	t.Setenv(agyConfigDirEnv, "/tmp/agy-test/config")
	if a := NewAgy(nil); a.Settings != "/tmp/agy-test/antigravity-cli/settings.json" {
		t.Errorf("settings = %q", a.Settings)
	}
}

func TestAgyValidModel(t *testing.T) {
	known := Agy{Models: func() []agyModel {
		return []agyModel{{"gemini-4-argon-medium", "Gemini 4 Argon (Medium)"}, {"claude-opus-5-5-low", "Claude Opus 5.5 (Low)"}}
	}}
	unknown := Agy{Models: func() []agyModel { return nil }}
	tests := []struct {
		name          string
		a             Agy
		model, effort string
		ok            bool
	}{
		{"a listed id", known, "gemini-4-argon-medium", "", true},
		{"a listed model and its effort", known, "gemini-4-argon", "medium", true},
		{"a model listed at another effort", known, "gemini-4-argon", "high", false},
		{"a listed model alone", known, "gemini-4-argon", "", true},
		{"a listed model at an effort no id ends in", known, "gemini-4-argon", "max", true},
		{"an unlisted model", known, "gemini-9", "", false},
		{"a Claude model agy lists", known, "claude-opus-5-5-low", "", false},
		{"a Claude model and its effort", known, "claude-opus-5-5", "low", false},
		{"any model when agy cannot say", unknown, "gemini-9", "", true},
		{"no Claude model when agy cannot say", unknown, "claude-opus-5-5", "", false},
		{"no GPT-OSS model", unknown, "gpt-oss-120b-medium", "", false},
		{"an effort agy takes", known, "", "xhigh", true},
		{"an effort it does not", known, "", "extreme", false},
		{"nothing", known, "", "", true},
	}
	for _, tt := range tests {
		if err := tt.a.ValidModel(tt.model, tt.effort); (err == nil) != tt.ok {
			t.Errorf("%s: ValidModel(%q, %q) = %v", tt.name, tt.model, tt.effort, err)
		}
	}
}

func TestListAgyModelsReadsTheIdsAndNames(t *testing.T) {
	dir := t.TempDir()
	script := "#!/bin/sh\necho 'Fetching available models...' >&2\nprintf 'gemini-4-argon-medium\\tGemini 4 Argon (Medium)\\nclaude-opus-5-5-low\\tClaude Opus 5.5 (Low)\\n'\n"
	if err := os.WriteFile(filepath.Join(dir, "agy"), []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if got := listAgyModels(); !slices.Equal(got, []agyModel{{"gemini-4-argon-medium", "Gemini 4 Argon (Medium)"}, {"claude-opus-5-5-low", "Claude Opus 5.5 (Low)"}}) {
		t.Errorf("models = %q", got)
	}
	t.Setenv("PATH", t.TempDir())
	if got := listAgyModels(); got != nil {
		t.Errorf("models without agy = %q", got)
	}
}

const agyCommon = `"conversationId":"c1","workspacePaths":["/work/tree"],"transcriptPath":"/t.jsonl","modelName":"gemini-4-argon-medium"`

func TestAgyParseEvent(t *testing.T) {
	transcript := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	lines := `{"step_index":0,"type":"USER_INPUT","content":"hi"}
{"step_index":1,"type":"PLANNER_RESPONSE","input_tokens":1,"tool_calls":[{"name":"run_command","args":{"CommandLine":"ls"}}]}
{"step_index":2,"type":"GENERIC","content":"out"}
{"step_index":3,"type":"PLANNER_RESPONSE","input_tokens":2,"content":"I listed it.\nWhat next?"}
`
	if err := os.WriteFile(transcript, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	stop := `{` + strings.Replace(agyCommon, "/t.jsonl", transcript, 1) + `,"terminationReason":"NO_TOOL_CALL","fullyIdle":true}`
	base := session.Event{Cwd: "/work/tree", Conversation: "c1", Transcript: "/t.jsonl", Model: "gemini-4-argon-medium"}
	with := func(ev session.Event, f func(*session.Event)) session.Event { f(&ev); return ev }
	tests := []struct {
		name, event, body string
		want              session.Event
		reply             string
		ok                bool
	}{
		{"start", "SessionStart", `{` + agyCommon + `}`, with(base, func(e *session.Event) { e.Name = "SessionStart" }), "{}", true},
		{"the first model call is the prompt", "PreInvocation", `{` + agyCommon + `,"invocationNum":0}`, with(base, func(e *session.Event) { e.Name = "UserPromptSubmit" }), "{}", true},
		{"later model calls are not recorded", "PreInvocation", `{` + agyCommon + `,"invocationNum":2}`, with(base, func(e *session.Event) { e.Name = "PreInvocation" }), "{}", false},
		{"a command", "PreToolUse", `{` + agyCommon + `,"stepIdx":2,"toolCall":{"name":"run_command","args":{"CommandLine":"echo one","WaitMsBeforeAsync":5000}}}`,
			with(base, func(e *session.Event) {
				e.Name, e.Tool, e.Command, e.Detail = "PreToolUse", "run_command", "echo one", "run_command echo one"
			}), "", true},
		{"a tool without a known argument", "PostToolUse", `{` + agyCommon + `,"toolCall":{"name":"view_file","args":{"AbsolutePath":"/work/a.go"}}}`,
			with(base, func(e *session.Event) { e.Name, e.Tool, e.Detail = "PostToolUse", "view_file", "view_file /work/a.go" }), "{}", true},
		{"stop reads the reply from the transcript", "Stop", stop,
			with(base, func(e *session.Event) {
				e.Name, e.Transcript, e.Detail, e.Asks = "Stop", transcript, "I listed it. What next?", true
			}), "{}", true},
		{"an event agentos does not take", "PostInvocation", `{` + agyCommon + `}`, with(base, func(e *session.Event) { e.Name = "PostInvocation" }), "{}", false},
		{"no payload", "Stop", ``, session.Event{Name: "Stop"}, "{}", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reply, ok := (Agy{}).ParseEvent(tt.event, strings.NewReader(tt.body))
			if got != tt.want || reply != tt.reply || ok != tt.ok {
				t.Errorf("ParseEvent = %+v, %q, %v\nwant        %+v, %q, %v", got, reply, ok, tt.want, tt.reply, tt.ok)
			}
		})
	}
}

func TestAgyStartHookInjectsThePrompt(t *testing.T) {
	file := filepath.Join(t.TempDir(), "prompt.md")
	if err := os.WriteFile(file, []byte("You run in agentos.\nAsk with a \"question\"."), 0o600); err != nil {
		t.Fatal(err)
	}
	_, reply, ok := (Agy{HookPrompt: file}).ParseEvent("SessionStart", strings.NewReader(`{`+agyCommon+`}`))
	var got struct {
		InjectSteps []struct {
			SystemMessage struct{ SystemMessage string }
		}
	}
	if err := json.Unmarshal([]byte(reply), &got); err != nil || !ok || len(got.InjectSteps) != 1 || got.InjectSteps[0].SystemMessage.SystemMessage != "You run in agentos.\nAsk with a \"question\"." {
		t.Errorf("reply = %s, %v, %v", reply, err, ok)
	}
	if _, reply, _ := (Agy{HookPrompt: filepath.Join(t.TempDir(), "missing")}).ParseEvent("SessionStart", strings.NewReader(`{}`)); reply != "{}" {
		t.Errorf("reply with no prompt file = %s", reply)
	}
}

func TestAgyChatHookKeepsTheChatReadOnly(t *testing.T) {
	tests := []struct {
		tool, command, decision string
	}{
		{"write_to_file", "", "deny"},
		{"replace_file_content", "", "deny"},
		{"multi_replace_file_content", "", "deny"},
		{"delete_file", "", "deny"},
		{"move_file", "", "deny"},
		{"run_command", "agentos issue 12 --prompt -", "allow"},
		{"run_command", "gh issue comment 12 --body-file -", "allow"},
		{"run_command", "agentos browser open http://x", "allow"},
		{"run_command", "agentos issues", "ask"},
		{"run_command", "git commit -m x", "ask"},
		{"run_command", "gh pr create", "ask"},
		{"view_file", "", ""},
		{"call_mcp_tool", "", ""},
	}
	for _, tt := range tests {
		body := `{"toolCall":{"name":"` + tt.tool + `","args":{"CommandLine":` + strconv(tt.command) + `}}}`
		_, reply, _ := (Agy{HookChat: true}).ParseEvent("PreToolUse", strings.NewReader(body))
		var got struct{ Decision string }
		_ = json.Unmarshal([]byte(reply), &got)
		if got.Decision != tt.decision || (tt.decision == "" && reply != "") {
			t.Errorf("%s %q in a chat: reply = %q, want decision %q", tt.tool, tt.command, reply, tt.decision)
		}
		if _, reply, _ := (Agy{}).ParseEvent("PreToolUse", strings.NewReader(body)); reply != "" {
			t.Errorf("%s %q in a task: reply = %q, want agy's own decision", tt.tool, tt.command, reply)
		}
	}
}

func strconv(s string) string {
	q, _ := json.Marshal(s)
	return string(q)
}

func TestAgyNoReply(t *testing.T) {
	if AgyNoReply("PreToolUse") != "" || AgyNoReply("Stop") != "{}" || AgyNoReply("SessionStart") != "{}" {
		t.Error("the empty reply is wrong for an event")
	}
}

// The panes below are what agy 1.4.4 drew, trimmed of blank lines and the logo.
const (
	agyAsksToRun = `> Run the shell command: touch made.txt
● Ran (touch made.txt) (ctrl+o to expand)
Command
────────────────────────────────────────────────────────────
Requesting permission for:
   touch made.txt
Run this command?
> 1. Yes, run command
  2. Yes, and always allow in this conversation for commands that start with 'touch'
  3. Yes, and always allow for commands that start with 'touch' (Persist to settings.json)
  4. No, cancel
  ↑/↓ Navigate · tab Amend · ctrl+g edit/expand command
esc to cancel                                                    Gemini 4 Argon · medium
`
	agyAsksToRead = `> Read the file /etc/shells with your file read tool.
● Used ReadFile (ctrl+o to expand)
File access
────────────────────────────────────────────────────────────
Read: /private/etc/shells
Reason: outside workspace
Allow access to this file?
  1. Yes, allow access
  2. Yes, and always allow non-workspace access
> 3. No, deny access
  ↑/↓ Navigate
esc to cancel                                                    Gemini 4 Argon · medium
`
	agyRuns = `> Run the shell command: sleep 8 && touch made.txt
● Running (sleep 8 && touch made.txt) (ctrl+o to expand)
⣻  Running command...
────────────────────────────────────────────────────────────
>
────────────────────────────────────────────────────────────
esc to cancel                                                    Gemini 4 Argon · medium
`
	agyRepliedWithChoices = `> Should I run it?
  Run this command?
  1. Yes, run it
  2. No, leave it
────────────────────────────────────────────────────────────
>
────────────────────────────────────────────────────────────
? for shortcuts                                                  Gemini 4 Argon · medium
`
)

func TestAgyPermissionPrompt(t *testing.T) {
	tests := []struct {
		name, pane, want string
		ok               bool
	}{
		{"a command", agyAsksToRun, "Run this command?", true},
		{"a file outside the workspace, the cursor on no", agyAsksToRead, "Allow access to this file?", true},
		{"a tool running", agyRuns, "", false},
		{"choices in a reply above the input box", agyRepliedWithChoices, "", false},
		{"an empty pane", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := Agy{}.PermissionPrompt(tt.pane)
			if got != tt.want || ok != tt.ok {
				t.Errorf("PermissionPrompt = %q, %v, want %q, %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}
