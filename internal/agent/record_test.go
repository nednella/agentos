package agent

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

var record = flag.String("record", "", "record the scenarios of these providers, comma-separated, by running the real agents")

// recordEnv names the file the hooks of a recorded agent append to. The hooks run this test binary.
const recordEnv = "AGENTOS_RECORD"

// The recordings keep these in place of the paths of the machine they were made on.
const (
	recordingDir = "$RECORDING"
	workDir      = "$WORK"
	homeDir      = "$HOME"
)

func TestMain(m *testing.M) {
	if file := os.Getenv(recordEnv); file != "" && len(os.Args) == 3 && os.Args[1] == "hook" {
		recordHook(file, os.Getenv("AGENTOS_AGENT"), os.Args[2])
		return
	}
	os.Exit(m.Run())
}

// hook is one line of a recording: a hook's event name and the payload the agent gave it.
type hook struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}

// recordHook appends the payload on stdin to file, and prints the reply the agent reads, as `agentos hook` does.
func recordHook(file, agent, event string) {
	payload, _ := io.ReadAll(os.Stdin)
	var compact bytes.Buffer
	if json.Compact(&compact, payload) != nil {
		compact.Reset()
		compact.WriteString("null")
	}
	line, _ := json.Marshal(hook{event, compact.Bytes()})
	if f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600); err == nil {
		_, _ = f.Write(append(line, '\n'))
		_ = f.Close()
	}
	_, reply, _ := New(agent, "").ParseEvent(event, bytes.NewReader(payload))
	fmt.Print(reply)
}

// recorder runs one provider's agent for real, with the provider's own hooks calling this test binary.
type recorder struct {
	version []string
	// command is the agent's command line, on the conversation to resume or a new one.
	command func(exe, resume string) []string
	// prepare writes what the agent needs in its folder, work, before it starts; nil for nothing.
	prepare func(exe, work string) error
	// trust are the keys that answer the agent's question whether it trusts a new folder. retrust starts the agent
	// again after it, for an agent that loads no hooks in a folder it did not trust when it started.
	trust   []string
	retrust bool
	// step is the type of the transcript lines the recording keeps: the ones with what a model call used and replied.
	// The others hold the owner's own instructions and account.
	step string
}

// recorders run cheap models with no settings, plugins or MCP servers of the owner's.
var recorders = map[string]recorder{
	"claude": {
		version: []string{"claude", "--version"},
		command: func(exe, resume string) []string {
			argv := Claude{Exe: exe}.Command(Launch{Model: "haiku", Resume: resume})
			return append(argv, "--setting-sources", "project", "--strict-mcp-config", "--permission-mode", "default")
		},
		trust: []string{"Down", "Enter"},
		step:  "assistant",
	},
	"agy": {
		version: []string{"agy", "--version"},
		command: func(_, resume string) []string {
			return Agy{}.Command(Launch{Model: "gemini-3.8-flash-low", Resume: resume})
		},
		prepare: func(exe, work string) error {
			return Agy{ConfigDir: filepath.Join(work, ".agents"), Installed: exe}.installHooks()
		},
		trust:   []string{"Enter"},
		retrust: true,
		step:    "PLANNER_RESPONSE",
	},
}

// TestRecord runs the real agents through every scenario and saves their hooks and transcripts in testdata. It spends
// requests, so it runs only when asked: go test ./internal/agent -run TestRecord -record claude,agy. Each agent runs in
// a throwaway folder on a tmux server of its own, with no agentos session around it.
func TestRecord(t *testing.T) {
	if *record == "" {
		t.Skip("records only with -record <provider>")
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range strings.Split(*record, ",") {
		r, ok := recorders[name]
		if !ok {
			t.Fatalf("no recorder for %q", name)
		}
		out, err := exec.Command(r.version[0], r.version[1:]...).Output()
		if err != nil {
			t.Fatalf("%s: %v", strings.Join(r.version, " "), err)
		}
		if err := os.MkdirAll(filepath.Join("testdata", name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("testdata", name, "version"), out, 0o644); err != nil {
			t.Fatal(err)
		}
		for _, sc := range scenarios {
			t.Run(name+"/"+sc.name, func(t *testing.T) {
				dir := filepath.Join("testdata", name, sc.name)
				if err := os.RemoveAll(dir); err != nil {
					t.Fatal(err)
				}
				if why := sc.cannot[name]; why != "" {
					t.Skip(why)
				}
				r.record(t, name, exe, sc, dir)
			})
		}
	}
}

func (r recorder) record(t *testing.T, name, exe string, sc scenario, dir string) {
	work, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(work, "note.txt"), []byte("kiwi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r.prepare != nil {
		if err := r.prepare(exe, work); err != nil {
			t.Fatal(err)
		}
	}
	hooks := filepath.Join(t.TempDir(), "hooks.jsonl")
	tm := newTmux(t, recordEnv+"="+hooks, "AGENTOS_AGENT="+name)
	start := func(resume string) {
		tm.start(work, r.command(exe, resume))
		tm.waitFor(func(pane string) bool {
			if !strings.Contains(pane, "trust this folder") {
				return strings.Contains(pane, "? for shortcuts")
			}
			tm.keys(r.trust...)
			if r.retrust {
				time.Sleep(2 * time.Second)
				tm.kill()
				tm.start(work, r.command(exe, resume))
			}
			return false
		})
	}
	start("")
	for _, s := range sc.steps {
		switch {
		case s.prompt != "":
			ends := len(turnEnds(readHooks(t, hooks)))
			tm.typeLine(s.prompt)
			waitUntil(t, "the turn to end", func() bool { return len(turnEnds(readHooks(t, hooks))) > ends })
		case s.exit:
			tm.typeLine("/exit")
			waitUntil(t, "the agent to exit", func() bool { return !tm.running() })
		case s.resume:
			start(lastConversation(New(name, ""), readHooks(t, hooks)))
		}
	}
	// The agent writes its transcript a moment after the hook, and a kill would add the hooks of an exit.
	time.Sleep(2 * time.Second)
	recorded := readHooks(t, hooks)
	transcripts := transcriptsOf(New(name, ""), recorded)
	tm.kill()
	save(t, dir, work, r.step, recorded, transcripts)
}

// save writes the hooks and the step lines of the transcripts they name into dir, with the paths of this machine
// replaced. Claude Code also names work by its path with each / and . made a -.
func save(t *testing.T, dir, work, step string, hooks []hook, transcripts []string) {
	home, _ := os.UserHomeDir()
	pairs := []string{}
	for i, path := range transcripts {
		pairs = append(pairs, path, recordingDir+"/transcript-"+fmt.Sprint(i+1)+".jsonl")
	}
	pairs = append(pairs, work, workDir, strings.NewReplacer("/", "-", ".", "-").Replace(work), workDir, home, homeDir)
	clean := strings.NewReplacer(pairs...)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	for _, h := range hooks {
		line, _ := json.Marshal(h)
		out.WriteString(clean.Replace(string(line)) + "\n")
	}
	if err := os.WriteFile(filepath.Join(dir, "hooks.jsonl"), out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	for i, path := range transcripts {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var steps strings.Builder
		for _, line := range strings.SplitAfter(string(data), "\n") {
			var l struct{ Type string }
			if json.Unmarshal([]byte(line), &l) == nil && l.Type == step {
				steps.WriteString(clean.Replace(line))
			}
		}
		if err := os.WriteFile(filepath.Join(dir, "transcript-"+fmt.Sprint(i+1)+".jsonl"), []byte(steps.String()), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func readHooks(t *testing.T, file string) []hook {
	data, err := os.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var hooks []hook
	lines := bufio.NewScanner(bytes.NewReader(data))
	lines.Buffer(nil, 1<<20)
	for lines.Scan() {
		var h hook
		if json.Unmarshal(lines.Bytes(), &h) == nil {
			hooks = append(hooks, h)
		}
	}
	return hooks
}

// turnEnds are the hooks that end a turn or wait on the owner.
func turnEnds(hooks []hook) []hook {
	var ends []hook
	for _, h := range hooks {
		if h.Event == "Stop" || h.Event == "Notification" {
			ends = append(ends, h)
		}
	}
	return ends
}

func lastConversation(p Provider, hooks []hook) string {
	var id string
	for _, h := range hooks {
		if ev, _, _ := p.ParseEvent(h.Event, bytes.NewReader(h.Payload)); ev.Conversation != "" {
			id = ev.Conversation
		}
	}
	return id
}

// transcriptsOf are the transcripts the hooks name, in the order they first name them.
func transcriptsOf(p Provider, hooks []hook) []string {
	var paths []string
	for _, h := range hooks {
		if ev, _, _ := p.ParseEvent(h.Event, bytes.NewReader(h.Payload)); ev.Transcript != "" && !slices.Contains(paths, ev.Transcript) {
			paths = append(paths, ev.Transcript)
		}
	}
	return paths
}

func waitUntil(t *testing.T, what string, done func() bool) {
	t.Helper()
	for deadline := time.Now().Add(2 * time.Minute); !done(); time.Sleep(300 * time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
	}
}

// tmux drives the agent in a tmux server of its own, which starts with none of the session's agentos or Claude Code
// environment, so nothing the agent runs reaches the owner's app.
type tmux struct {
	t      *testing.T
	socket string
	env    []string
}

func newTmux(t *testing.T, env ...string) *tmux {
	tm := &tmux{t: t, socket: fmt.Sprintf("agentos-record-%d", os.Getpid())}
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(name, "AGENTOS_") && !strings.HasPrefix(name, "CLAUDE") && name != "TMUX" {
			tm.env = append(tm.env, kv)
		}
	}
	tm.env = append(tm.env, env...)
	t.Cleanup(func() { _ = tm.run("kill-server") })
	return tm
}

func (tm *tmux) run(args ...string) error {
	cmd := exec.Command("tmux", append([]string{"-L", tm.socket}, args...)...)
	cmd.Env = tm.env
	return cmd.Run()
}

func (tm *tmux) start(dir string, argv []string) {
	if err := tm.run(append([]string{"new-session", "-d", "-s", "agent", "-x", "200", "-y", "50", "-c", dir, "--"}, argv...)...); err != nil {
		tm.t.Fatalf("starting %q: %v", argv, err)
	}
}

func (tm *tmux) running() bool { return tm.run("has-session", "-t", "agent") == nil }

func (tm *tmux) kill() { _ = tm.run("kill-session", "-t", "agent") }

func (tm *tmux) keys(keys ...string) {
	if err := tm.run(append([]string{"send-keys", "-t", "agent"}, keys...)...); err != nil {
		tm.t.Fatal(err)
	}
}

// typeLine types text and presses Enter a moment later, as an agent's prompt takes a paste before a submit.
func (tm *tmux) typeLine(text string) {
	tm.keys("-l", text)
	time.Sleep(500 * time.Millisecond)
	tm.keys("Enter")
}

// waitFor waits until ready says yes to what the pane shows.
func (tm *tmux) waitFor(ready func(pane string) bool) {
	waitUntil(tm.t, "the agent to be ready", func() bool {
		cmd := exec.Command("tmux", "-L", tm.socket, "capture-pane", "-p", "-t", "agent")
		cmd.Env = tm.env
		out, _ := cmd.Output()
		return ready(string(out))
	})
}
