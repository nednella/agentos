package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/usage"
)

// scenario is something a session does, which every provider is recorded doing and must end with the same row for.
// Cannot names each provider that cannot do it, and why: it is recorded and tested for the others only.
type scenario struct {
	name   string
	steps  []step
	state  session.State
	reply  string // a word the row's last reply holds, "" when the scenario ends with no reply
	tool   bool   // the row names a tool
	usage  bool   // the row shows tokens and a cost
	cannot map[string]string
}

// step is what the recorder does to the agent: type a prompt and wait for the turn to end, exit, or start the agent
// again on its conversation.
type step struct {
	prompt       string
	exit, resume bool
}

var pong = step{prompt: "Reply with the one word pong, in lower case, and nothing else."}

var scenarios = []scenario{
	{name: "prompt", steps: []step{pong}, state: session.Idle, reply: "pong", usage: true},
	{name: "tool", steps: []step{{prompt: "Read the file note.txt and reply with its contents only."}}, state: session.Idle, reply: "kiwi", tool: true, usage: true},
	{name: "question", steps: []step{{prompt: "Reply with this question and nothing else: Shall I go on?"}}, state: session.Waiting, reply: "go on?", usage: true},
	{name: "permission", steps: []step{{prompt: "Run the shell command: touch made.txt"}}, state: session.Waiting, tool: true, usage: true,
		cannot: map[string]string{"agy": "agy fires no hook when it asks for a permission (#490)"}},
	{name: "exit", steps: []step{pong, {exit: true}}, state: session.Ended, usage: true,
		cannot: map[string]string{"agy": "agy fires no hook when it exits: the app sees its process end"}},
	{name: "resume", steps: []step{pong, {exit: true}, {resume: true}, {prompt: "Reply with the one word plum, in lower case, and nothing else."}}, state: session.Idle, reply: "plum", usage: true},
}

// TestScenarios replays each provider's recordings in testdata/<provider>/<scenario>, as the app reads hooks and
// transcripts, and checks the row each scenario must end with. After an update of an agent, record it again (see
// TestRecord): the diff of testdata shows what the agent changed, and this test what that breaks.
func TestScenarios(t *testing.T) {
	for _, name := range Names {
		version, err := os.ReadFile(filepath.Join("testdata", name, "version"))
		if err != nil {
			t.Errorf("%s has no recordings: run go test ./internal/agent -run TestRecord -record %s", name, name)
			continue
		}
		for _, sc := range scenarios {
			t.Run(name+"/"+sc.name, func(t *testing.T) {
				if why := sc.cannot[name]; why != "" {
					t.Skip(why)
				}
				t.Logf("recorded with %s", strings.TrimSpace(string(version)))
				row := replay(t, New(name, ""), filepath.Join("testdata", name, sc.name))
				sc.check(t, row)
			})
		}
	}
}

// row is what the app ends with for a session: its record, every conversation its hooks named, and what it used.
type row struct {
	session.Record
	conversations map[string]bool
	usage         []usage.Part
}

// replay feeds the recorded hooks to p one by one, and reads the transcripts after each, as the app does.
func replay(t *testing.T, p Provider, dir string) row {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "hooks.jsonl"))
	if err != nil {
		t.Fatalf("no recording: %v", err)
	}
	abs, _ := filepath.Abs(dir)
	data = bytes.ReplaceAll(data, []byte(recordingDir), []byte(abs))
	r := row{conversations: map[string]bool{}}
	reader := p.Usage(time.Time{})
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	lines := bufio.NewScanner(bytes.NewReader(data))
	lines.Buffer(nil, 1<<20)
	for lines.Scan() {
		var h hook
		if err := json.Unmarshal(lines.Bytes(), &h); err != nil {
			t.Fatalf("hook %q: %v", lines.Text(), err)
		}
		ev, _, ok := p.ParseEvent(h.Event, bytes.NewReader(h.Payload))
		if !ok {
			continue
		}
		at = at.Add(time.Second)
		r.Record = session.Apply("demo/abcdefgh", r.Record, ev, at)
		if ev.Conversation != "" {
			r.conversations[ev.Conversation] = true
		}
		if r.Transcript == "" {
			continue
		}
		parts, _, err := reader.Read(r.Transcript, r.AgentModel)
		if err != nil {
			t.Fatal(err)
		}
		r.usage = append(r.usage, parts...)
	}
	return r
}

func (sc scenario) check(t *testing.T, r row) {
	t.Helper()
	if r.State != sc.state {
		t.Errorf("state = %q, want %q", r.State, sc.state)
	}
	if sc.reply != "" && !strings.Contains(strings.ToLower(r.Detail), sc.reply) {
		t.Errorf("reply = %q, want it to hold %q", r.Detail, sc.reply)
	}
	if sc.tool != (r.Tool != "") {
		t.Errorf("tool = %q, want one: %v", r.Tool, sc.tool)
	}
	if len(r.conversations) != 1 {
		t.Errorf("conversations = %v, want one", r.conversations)
	}
	var used usage.Tokens
	for _, p := range r.usage {
		used.Add(p.Tokens)
		if p.Model == "" {
			t.Errorf("usage of no model: %+v", p)
		}
	}
	if sc.usage != (used.Total() > 0 && used.Cost > 0) {
		t.Errorf("usage = %+v, want tokens and a cost: %v", used, sc.usage)
	}
}
