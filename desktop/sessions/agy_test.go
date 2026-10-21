package sessions_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/bus"
	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/prompts"
)

// agyEnv is what the stand-in agy was started with: AGENTOS_PROMPT and AGENTOS_CHAT, by name.
func agyEnv(t *testing.T, dir, id string) map[string]string {
	t.Helper()
	claudeArgs(t, dir, id) // the env file comes first
	data, err := os.ReadFile(filepath.Join(dir, strings.ReplaceAll(id, "/", "_")+".env"))
	if err != nil {
		t.Fatal(err)
	}
	env := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		k, v, _ := strings.Cut(line, "=")
		env[k] = v
	}
	return env
}

func agyPayload(h *apptest.Harness, transcript string, extra string) string {
	return agyPayloadOn(h, transcript, "gemini-4-argon-medium", extra)
}

func agyPayloadOn(h *apptest.Harness, transcript, model, extra string) string {
	return fmt.Sprintf(`{"conversationId":"conv-1","workspacePaths":[%q],"transcriptPath":%q,"modelName":%q%s}`, h.Dir, transcript, model, extra)
}

func TestAgySessionStartsWithItsPromptInAFileAndItsHooksInAgysConfig(t *testing.T) {
	dir := apptest.AgyOnPath(t)
	h := newHarness(t)
	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"with agy"}, Opts: map[string]string{"agent": "agy", "prompt": "say hi"}}); !resp.OK {
		t.Fatalf("new --agent agy = %+v", resp)
	}
	s := sessionNumbered(t, h, 1)
	if s.Agent != "agy" || s.Chat {
		t.Errorf("session = %+v", s)
	}
	args := claudeArgs(t, dir, s.ID)
	if i := slices.Index(args, "-i"); i < 0 || args[i+1] != "say hi" || slices.Contains(args, "--mode") || strings.Contains(strings.Join(args, " "), prompts.Task()) {
		t.Errorf("args = %q", args)
	}
	env := agyEnv(t, dir, s.ID)
	prompt, err := os.ReadFile(env["AGENTOS_PROMPT"])
	if err != nil || !strings.Contains(string(prompt), prompts.Session()) || !strings.Contains(string(prompt), prompts.Task()) || strings.Contains(string(prompt), prompts.Chat()) {
		t.Errorf("prompt file %q = %q, %v", env["AGENTOS_PROMPT"], prompt, err)
	}
	if env["AGENTOS_CHAT"] != "" {
		t.Errorf("a task was marked as a chat: %v", env)
	}
	hooks, err := os.ReadFile(filepath.Join(os.Getenv("AGENTOS_AGY_CONFIG_DIR"), "hooks.json"))
	home, _ := os.UserHomeDir()
	if err != nil || !strings.Contains(string(hooks), "'"+filepath.Join(home, ".local", "bin", "agentos")+"' hook Stop") {
		t.Errorf("agy's hooks.json = %s, %v", hooks, err)
	}
	if _, err := os.Stat(filepath.Join(home, ".gemini", "config", "hooks.json")); err == nil {
		if data, _ := os.ReadFile(filepath.Join(home, ".gemini", "config", "hooks.json")); strings.Contains(string(data), "agentos") {
			t.Fatal("the test wrote the owner's own agy hooks")
		}
	}

	chat, err := h.Sessions().CreateChat("ask agy", "", false, "agy")
	if err != nil {
		t.Fatal(err)
	}
	if args := claudeArgs(t, dir, chat.ID); !slices.Contains(args, "--mode") {
		t.Errorf("args of a chat = %q", args)
	}
	env = agyEnv(t, dir, chat.ID)
	if prompt, _ := os.ReadFile(env["AGENTOS_PROMPT"]); env["AGENTOS_CHAT"] != "1" || !strings.Contains(string(prompt), prompts.Chat()) {
		t.Errorf("chat env = %v, prompt = %q", env, prompt)
	}
}

func TestAgyHooksMoveTheRowAndCountItsUsage(t *testing.T) {
	apptest.AgyOnPath(t)
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4, "agy")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	reply := h.HookReply(t, s.ID, "SessionStart", agyPayload(h, transcript, ""))
	if reply != "{}" {
		t.Errorf("SessionStart reply = %q", reply)
	}
	h.Hook(t, s.ID, "PreInvocation", agyPayload(h, transcript, `,"invocationNum":0`))
	eventually(t, "the row working", func() bool { row, _ := h.Session(s.ID); return row.State == "working" })
	// Written to the second, so a reply must sit well after the session was created to count.
	at := time.Now().Add(2 * time.Second).UTC().Format(time.RFC3339)
	appendLines(t, transcript,
		`{"step_index":0,"type":"USER_INPUT","created_at":"`+at+`","content":"go"}`+"\n",
		`{"step_index":1,"type":"PLANNER_RESPONSE","created_at":"`+at+`","input_tokens":1000,"cache_read_tokens":0,"output_tokens":100,"tool_calls":[{"name":"run_command","args":{"CommandLine":"ls"}}]}`+"\n")
	if reply := h.HookReply(t, s.ID, "PreToolUse", agyPayload(h, transcript, `,"toolCall":{"name":"run_command","args":{"CommandLine":"ls"}}`)); reply != "" {
		t.Errorf("PreToolUse reply in a task = %q", reply)
	}
	eventually(t, "the tool on the row", func() bool { row, _ := h.Session(s.ID); return row.Detail == "run_command ls" })
	appendLines(t, transcript,
		`{"step_index":2,"type":"GENERIC","created_at":"`+at+`","content":"out"}`+"\n",
		`{"step_index":3,"type":"PLANNER_RESPONSE","created_at":"`+at+`","input_tokens":10,"cache_read_tokens":1000,"output_tokens":1,"content":"Done. Shall I push?"}`+"\n")
	h.Hook(t, s.ID, "Stop", agyPayload(h, transcript, `,"terminationReason":"NO_TOOL_CALL"`))
	eventually(t, "the row waiting on the question", func() bool {
		row, _ := h.Session(s.ID)
		return row.State == "waiting" && row.Detail == "Done. Shall I push?"
	})
	eventually(t, "the turn's tokens on the row", func() bool { row, _ := h.Session(s.ID); return row.Tokens == 2111 })
	row, _ := h.Session(s.ID)
	if row.Cost != 0.00313 {
		t.Errorf("cost = %v, want 1010 input at $2, 101 output at $10 and 1000 cached at $0.10 per million", row.Cost)
	}
	eventually(t, "the usage in the event log", func() bool {
		events := h.Events(t, "usage")
		return len(events) == 1 && events[0].Model == "gemini-4-argon-medium" && events[0].Tokens.Total() == 2111
	})
	if row.Model != "gemini-4-argon" || row.Effort != "medium" {
		t.Errorf("row = %s %s, want the model and effort the hooks report", row.Model, row.Effort)
	}
	h.Hook(t, s.ID, "PreInvocation", agyPayloadOn(h, transcript, "gemini-3.8-flash-low", `,"invocationNum":0`))
	eventually(t, "the row following a change of model", func() bool {
		row, _ := h.Session(s.ID)
		return row.Model == "gemini-3.8-flash" && row.Effort == "low"
	})
}

func TestAgyChatHookRefusesAWrite(t *testing.T) {
	apptest.AgyOnPath(t)
	h := newHarness(t)
	chat, err := h.Sessions().CreateChat("ask", "", false, "agy")
	if err != nil {
		t.Fatal(err)
	}
	write := h.HookReply(t, chat.ID, "PreToolUse", agyPayload(h, "", `,"toolCall":{"name":"write_to_file","args":{"TargetFile":"a.go"}}`), "AGENTOS_CHAT=1")
	if !strings.Contains(write, `"decision":"deny"`) {
		t.Errorf("a chat's write got %q", write)
	}
	hand := h.HookReply(t, chat.ID, "PreToolUse", agyPayload(h, "", `,"toolCall":{"name":"run_command","args":{"CommandLine":"agentos note hi"}}`), "AGENTOS_CHAT=1")
	if !strings.Contains(hand, `"decision":"allow"`) {
		t.Errorf("a chat's agentos note got %q", hand)
	}
}

func TestResumedAgySessionContinuesItsConversation(t *testing.T) {
	dir := apptest.AgyOnPath(t)
	h := newHarness(t)
	old, err := h.Sessions().Create("#12 work", "", false, 12, "agy")
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, old.ID, "Stop", agyPayload(h, "", ""))
	if err := h.KillSession(old.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the session ended", func() bool { got, _ := h.Session(old.ID); return got.State == "ended" && got.Resumable })
	fresh, err := h.ResumeSession(old.ID)
	if err != nil {
		t.Fatal(err)
	}
	args := claudeArgs(t, dir, fresh.ID)
	if i := slices.Index(args, "--conversation"); i < 0 || args[i+1] != "conv-1" || fresh.Agent != "agy" {
		t.Errorf("resumed with %q as %+v", args, fresh)
	}
	if env := agyEnv(t, dir, fresh.ID); env["AGENTOS_PROMPT"] != "" {
		t.Errorf("a resumed session got a prompt to inject again: %v", env)
	}
}

func TestAgySessionWaitsWhileItsPaneAsksForPermission(t *testing.T) {
	apptest.AgyAskingOnPath(t)
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4, "agy")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	h.Hook(t, s.ID, "SessionStart", agyPayload(h, transcript, ""))
	eventually(t, "the prompt in the pane", func() bool { return strings.Contains(h.Pane(t, s.ID), "Run this command?") })
	tool := `,"toolCall":{"name":"run_command","args":{"CommandLine":"touch made.txt"}}`
	h.Hook(t, s.ID, "PreToolUse", agyPayload(h, transcript, tool))
	eventually(t, "the row waiting on the prompt", func() bool {
		row, _ := h.Session(s.ID)
		return row.State == "waiting" && row.Detail == "Run this command?"
	})
	rec, _ := bus.ReadState(h.State, s.ID)
	if rec.Notify != "permission_prompt" || rec.Tool != "run_command" || rec.Command != "touch made.txt" {
		t.Errorf("record = %+v, want the permission prompt about the tool call", rec)
	}
	h.Hook(t, s.ID, "PostToolUse", agyPayload(h, transcript, tool))
	eventually(t, "the row working once the tool ran", func() bool { row, _ := h.Session(s.ID); return row.State == "working" })
}

func TestAgySessionKeepsWorkingWhenItsPaneAsksNothing(t *testing.T) {
	apptest.AgyOnPath(t)
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4, "agy")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	h.Hook(t, s.ID, "PreToolUse", agyPayload(h, transcript, `,"toolCall":{"name":"run_command","args":{"CommandLine":"go test ./..."}}`))
	eventually(t, "the row working", func() bool { row, _ := h.Session(s.ID); return row.State == "working" })
	time.Sleep(agent.Agy{}.PromptWait() + time.Second)
	if row, _ := h.Session(s.ID); row.State != "working" {
		t.Errorf("state = %s after a long tool call that asked nothing", row.State)
	}
}
