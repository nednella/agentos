package agent

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/usage"
	"github.com/nednella/agentos/internal/util"
)

// Agy runs the Antigravity CLI. agy takes no hooks or MCP servers on its command line: they come from the global
// files in ConfigDir, which Prepare writes with the installed agentos, Installed, as their command, so one file
// serves every build of the app. Writes says whether this build may write them: the installed app always, any other
// only when AGENTOS_AGY_CONFIG_DIR named ConfigDir, so a dev or test build never touches the owner's own files.
// Settings is agy's own settings file, where it keeps its default model. ListModels reads agy models, nil when agy
// cannot say.
//
// In a hook, HookPrompt and HookChat come from the session's environment: the file holding the prompt that
// SessionStart injects, and whether the PreToolUse hook keeps the session read-only.
type Agy struct {
	ConfigDir  string
	Installed  string
	Writes     bool
	Dirs       []string
	Settings   string
	ListModels func() []agyModel
	HookPrompt string
	HookChat   bool
}

const agyConfigDirEnv = "AGENTOS_AGY_CONFIG_DIR"

// NewAgy is the agy provider for this process: the app, which starts sessions, or the hook, which reads one.
func NewAgy(dirs []string) Agy {
	home, _ := os.UserHomeDir()
	a := Agy{
		ConfigDir: os.Getenv(agyConfigDirEnv), Installed: filepath.Join(home, ".local", "bin", "agentos"), Writes: true, Dirs: dirs,
		ListModels: agyModels, HookPrompt: os.Getenv("AGENTOS_PROMPT"), HookChat: os.Getenv("AGENTOS_CHAT") != "",
	}
	if a.ConfigDir == "" {
		a.ConfigDir = filepath.Join(home, ".gemini", "config")
		a.Writes = runsAs(a.Installed)
	}
	a.Settings = filepath.Join(filepath.Dir(a.ConfigDir), "antigravity-cli", "settings.json")
	return a
}

// runsAs says whether this process is the command at path, through its link.
func runsAs(path string) bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	real, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return false
	}
	want, err := filepath.EvalSymlinks(path)
	return err == nil && real == want
}

func (a Agy) Name() string        { return "agy" }
func (a Agy) Can(Capability) bool { return true }

// Command is agy's command line. The model id carries an effort agy lists (gemini-4-argon-medium); any other effort
// goes to --effort. A chat runs in plan mode, which only tells agy to plan: the PreToolUse hook is what keeps it from
// changing files.
func (a Agy) Command(l Launch) []string {
	argv := []string{"agy"}
	model, effort := joinModel(l.Model, l.Effort)
	if model != "" {
		argv = append(argv, "--model", model)
	}
	if effort != "" {
		argv = append(argv, "--effort", effort)
	}
	if l.Resume != "" {
		argv = append(argv, "--conversation", l.Resume)
	}
	if l.Chat {
		argv = append(argv, "--mode", "plan")
	}
	for _, dir := range a.Dirs {
		argv = append(argv, "--add-dir", dir)
	}
	if l.Start != "" {
		argv = append(argv, "-i", l.Start)
	}
	return argv
}

// Env names the session's prompt file for the SessionStart hook, and marks a chat for the PreToolUse hook.
func (a Agy) Env(l Launch) []string {
	var env []string
	if file := promptFile(l); file != "" {
		env = append(env, "AGENTOS_PROMPT="+file)
	}
	if l.Chat {
		env = append(env, "AGENTOS_CHAT=1")
	}
	return env
}

// promptFile is where the session's prompt waits for the SessionStart hook, "" when there is none to inject: the
// message the hook injects stays in the conversation, so a resumed one has it already.
func promptFile(l Launch) string {
	if l.Prompt == "" || l.Resume != "" || l.TmpDir == "" {
		return ""
	}
	return filepath.Join(l.TmpDir, "prompt.md")
}

// Fill keeps the model and the effort apart, as a label or an action may give the full id, and fills what l leaves
// empty from agy's default model. agy keeps that as the name agy models prints beside the id; a name agy does not
// list, as when it is not signed in, leaves the choice to agy.
func (a Agy) Fill(_ string, l Launch) Launch {
	model, effort := splitModel(l.Model)
	l.Model, l.Effort = model, cmp.Or(l.Effort, effort)
	model, effort = splitModel(a.defaultModel())
	l.Model, l.Effort = cmp.Or(l.Model, model), cmp.Or(l.Effort, effort)
	return l
}

// defaultModel is the id of the model agy's settings name, "" when they name none of the Google models agy lists.
func (a Agy) defaultModel() string {
	data, err := os.ReadFile(a.Settings)
	if err != nil || a.ListModels == nil {
		return ""
	}
	var settings struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(data, &settings) != nil || settings.Model == "" {
		return ""
	}
	for _, m := range a.ListModels() {
		if m.Name == settings.Model && googleModel(m.ID) {
			return m.ID
		}
	}
	return ""
}

// agyEfforts are the efforts a model id of agy ends in.
var agyEfforts = []string{"low", "medium", "high"}

// agyEffortFlag are the efforts agy's --effort takes.
var agyEffortFlag = []string{"low", "medium", "high", "xhigh", "max"}

// splitModel parts a model id of agy into the model and the effort it ends in, "" when it ends in none.
func splitModel(id string) (model, effort string) {
	for _, e := range agyEfforts {
		if model, ok := strings.CutSuffix(id, "-"+e); ok && model != "" {
			return model, e
		}
	}
	return id, ""
}

// joinModel is splitModel turned around: the id for --model, and the effort that is left for --effort.
func joinModel(model, effort string) (id, rest string) {
	if model != "" && slices.Contains(agyEfforts, effort) {
		return model + "-" + effort, ""
	}
	return model, effort
}

// Models are the Google models agy models lists, with the efforts their ids end in and those only --effort takes;
// nil when agy cannot say.
func (a Agy) Models() []Model {
	if a.ListModels == nil {
		return nil
	}
	listed := a.ListModels()
	if len(listed) == 0 {
		return nil
	}
	var bases []string
	ends := map[string][]string{}
	for _, m := range listed {
		if !googleModel(m.ID) {
			continue
		}
		base, effort := splitModel(m.ID)
		if !slices.Contains(bases, base) {
			bases = append(bases, base)
		}
		ends[base] = append(ends[base], effort)
	}
	models := make([]Model, len(bases))
	for i, base := range bases {
		models[i] = Model{ID: base}
		for _, e := range agyEffortFlag {
			if !slices.Contains(agyEfforts, e) || slices.Contains(ends[base], e) {
				models[i].Efforts = append(models[i].Efforts, e)
			}
		}
	}
	return models
}

// ValidModel takes the model as Fill does: a full id sets the effort too, unless an effort is given. Any Google model
// passes when agy cannot say, as when it is not signed in.
func (a Agy) ValidModel(model, effort string) error {
	if model == "" && effort == "" {
		return nil
	}
	model, own := splitModel(model)
	effort = cmp.Or(effort, own)
	if model != "" && !googleModel(model) {
		return fmt.Errorf("agy runs only Google models, not %q", model)
	}
	models := a.Models()
	if models == nil {
		models = []Model{{ID: model, Efforts: agyEffortFlag}}
	}
	return checkModel("agy", models, model, effort)
}

// agyModel is one line of agy models: the id agy takes and the name its settings keep.
type agyModel struct{ ID, Name string }

// googleModel says whether the model id is one of Google's, the only models an agy session runs. agy also lists
// Claude models, which run through the claude provider on the owner's subscription, and GPT-OSS.
func googleModel(id string) bool { return strings.HasPrefix(id, "gemini-") }

// agyModels asks agy once per process which models it has.
var agyModels = sync.OnceValue(listAgyModels)

// listAgyModels reads agy models, nil when agy is not there or cannot answer.
func listAgyModels() []agyModel {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "agy", "models").Output()
	if err != nil {
		return nil
	}
	var models []agyModel
	for _, line := range strings.Split(string(out), "\n") {
		if id, name, ok := strings.Cut(line, "\t"); ok && id != "" {
			models = append(models, agyModel{id, strings.TrimSpace(name)})
		}
	}
	return models
}

// Prepare writes the session's prompt file, and the global hooks and MCP entry when this build may.
func (a Agy) Prepare(l Launch) error {
	if file := promptFile(l); file != "" {
		if err := os.WriteFile(file, []byte(l.Prompt), 0o600); err != nil {
			return fmt.Errorf("saving the session's prompt: %w", err)
		}
	}
	if !a.Writes {
		return nil
	}
	if err := a.installHooks(); err != nil {
		return fmt.Errorf("writing agy's hooks: %w", err)
	}
	if err := a.installMCP(); err != nil {
		return fmt.Errorf("writing agy's MCP servers: %w", err)
	}
	return nil
}

// agyEvents are the hooks agentos takes from agy. The two tool hooks are grouped under a matcher, as agy wants.
var agyEvents = []string{"SessionStart", "PreInvocation", "PreToolUse", "PostToolUse", "Stop"}

// installHooks keeps the named hook agentos in hooks.json up to date and leaves the others alone.
func (a Agy) installHooks() error {
	hook := map[string]any{}
	for _, ev := range agyEvents {
		handler := hookCommand{Type: "command", Command: util.ShellQuote(a.Installed) + " hook " + ev}
		if ev == "PreToolUse" || ev == "PostToolUse" {
			hook[ev] = []hookGroup{{Matcher: "*", Hooks: []hookCommand{handler}}}
		} else {
			hook[ev] = []hookCommand{handler}
		}
	}
	return a.setEntry(filepath.Join(a.ConfigDir, "hooks.json"), nil, "agentos", hook)
}

// installMCP keeps the server browser in mcp_config.json up to date: agentos stands in for the page tools and finds the
// session's browser itself, and agy withholds the tools the session must not have.
func (a Agy) installMCP() error {
	server := map[string]any{"command": a.Installed, "args": []string{"mcp-browser"}, "disabledTools": BrowserToolsDenied}
	return a.setEntry(filepath.Join(a.ConfigDir, "mcp_config.json"), []string{"mcpServers"}, "browser", server)
}

// BrowserToolsDenied are the page tools a session must not have: new_page and close_page would open and close pages
// outside the session's own, in a window of another session.
var BrowserToolsDenied = []string{"new_page", "close_page"}

// setEntry sets key to value in the JSON object at path, under the objects named by keys, and writes the file only
// when that changes it. A missing or empty file is an empty object.
func (a Agy) setEntry(path string, keys []string, key string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var top map[string]json.RawMessage
	if len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &top); err != nil {
			return fmt.Errorf("%s is not a JSON object: %w", path, err)
		}
	}
	want, _ := json.Marshal(value)
	changed, err := setIn(&top, append(keys, key), want)
	if err != nil || !changed {
		return err
	}
	out, _ := json.MarshalIndent(top, "", "  ")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return atomicfile.Write(path, append(out, '\n'), 0o600)
}

// setIn sets the value at the path of keys in obj, making the objects on the way, and reports whether it differed.
func setIn(obj *map[string]json.RawMessage, keys []string, value json.RawMessage) (bool, error) {
	if *obj == nil {
		*obj = map[string]json.RawMessage{}
	}
	if len(keys) == 1 {
		var have bytes.Buffer
		if json.Compact(&have, (*obj)[keys[0]]) == nil && bytes.Equal(have.Bytes(), value) {
			return false, nil
		}
		(*obj)[keys[0]] = value
		return true, nil
	}
	var inner map[string]json.RawMessage
	if raw := (*obj)[keys[0]]; len(raw) > 0 {
		if err := json.Unmarshal(raw, &inner); err != nil {
			return false, fmt.Errorf("%s is not a JSON object: %w", keys[0], err)
		}
	}
	changed, err := setIn(&inner, keys[1:], value)
	if err != nil || !changed {
		return false, err
	}
	(*obj)[keys[0]], _ = json.Marshal(inner)
	return true, nil
}

func (a Agy) Usage(after time.Time) usage.Reader { return agyUsage{usage.NewAgyReader(after)} }

// agyUsage says with each read what the agent used last: the id its hooks report, parted into model and effort.
type agyUsage struct{ *usage.AgyReader }

func (u agyUsage) Read(path, id string) ([]usage.Part, usage.Current, error) {
	parts, _, err := u.AgyReader.Read(path, id)
	model, effort := splitModel(id)
	return parts, usage.Current{Model: model, Effort: effort}, err
}

type agyPayload struct {
	ConversationID string   `json:"conversationId"`
	WorkspacePaths []string `json:"workspacePaths"`
	TranscriptPath string   `json:"transcriptPath"`
	ModelName      string   `json:"modelName"`
	InvocationNum  int      `json:"invocationNum"`
	ToolCall       agyTool  `json:"toolCall"`
}

type agyTool struct {
	Name string         `json:"name"`
	Args map[string]any `json:"args"`
}

// AgyNoReply is what an agy hook prints when agentos has nothing to say. agy reads a JSON object from every hook,
// but on PreToolUse it takes an empty object for a deny (agy 1.4.3), and only no output leaves the decision to it.
func AgyNoReply(event string) string {
	if event == "PreToolUse" {
		return ""
	}
	return "{}"
}

// ParseEvent maps agy's hook onto the canonical event. PreInvocation fires before every model call of a turn,
// numbered from 0: the first is the prompt, and the tool calls keep the row working after it. Stop carries no reply,
// so the reply comes from the transcript, which has it by then.
func (a Agy) ParseEvent(name string, r io.Reader) (session.Event, string, bool) {
	var p agyPayload
	_ = json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&p)
	ev := session.Event{Name: name, Conversation: p.ConversationID, Transcript: p.TranscriptPath, Model: p.ModelName}
	if len(p.WorkspacePaths) > 0 {
		ev.Cwd = p.WorkspacePaths[0]
	}
	reply := AgyNoReply(name)
	switch name {
	case "SessionStart":
		reply = a.startReply()
	case "PreInvocation":
		if p.InvocationNum != 0 {
			return ev, reply, false
		}
		ev.Name = "UserPromptSubmit"
	case "PreToolUse", "PostToolUse":
		ev.Tool = p.ToolCall.Name
		command, _ := p.ToolCall.Args["CommandLine"].(string)
		ev.Command = oneLine(command, 200)
		ev.Detail = oneLine(agyToolDetail(p.ToolCall), 120)
		if name == "PreToolUse" && a.HookChat {
			reply = chatDecision(p.ToolCall.Name, command)
		}
	case "Stop":
		text := lastReply(p.TranscriptPath)
		ev.Detail, ev.Asks = oneLine(text, 120), asks(text)
	default:
		return ev, reply, false
	}
	return ev, reply, true
}

// startReply injects the session's prompt as a system message, which agy keeps in the conversation.
func (a Agy) startReply() string {
	if a.HookPrompt == "" {
		return "{}"
	}
	prompt, err := os.ReadFile(a.HookPrompt)
	if err != nil || len(prompt) == 0 {
		return "{}"
	}
	type systemMessage struct {
		SystemMessage string `json:"systemMessage"`
	}
	type step struct {
		SystemMessage systemMessage `json:"systemMessage"`
	}
	reply, _ := json.Marshal(map[string][]step{"injectSteps": {{systemMessage{string(prompt)}}}})
	return string(reply)
}

// agyWriteTools are the tools a chat must not run, besides any named write_*, replace_*, delete_* or move_*.
var agyWriteTools = []string{"write_to_file", "replace_file_content", "multi_replace_file_content"}

var agyWritePrefixes = []string{"write_", "replace_", "delete_", "move_"}

// chatDecision is the PreToolUse reply that keeps a chat read-only: no writes, allow for the commands that hand work
// on, a prompt for any other command, and agy's own permissions for the rest. agy 1.4.3 asks before an allowed command
// all the same; the allow is for an agy that honours it.
func chatDecision(tool, command string) string {
	switch {
	case slices.Contains(agyWriteTools, tool) || hasPrefix(tool, agyWritePrefixes):
		return decision("deny", "a chat reads the project and does not change it")
	case tool == "run_command":
		if isChatCommand(command) {
			return decision("allow", "")
		}
		return decision("ask", "a chat runs only the agentos and gh issue commands that hand work on")
	}
	return ""
}

func decision(what, reason string) string {
	reply, _ := json.Marshal(map[string]string{"decision": what, "reason": reason})
	return string(reply)
}

// isChatCommand says whether the command line is one of chatTools, which claude allows a chat as Bash(<words>:*).
func isChatCommand(command string) bool {
	command = strings.TrimSpace(command)
	for _, tool := range chatTools {
		words := strings.TrimSuffix(strings.TrimPrefix(tool, "Bash("), ":*)")
		if command == words || strings.HasPrefix(command, words+" ") {
			return true
		}
	}
	return false
}

func hasPrefix(s string, prefixes []string) bool {
	return slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(s, p) })
}

// agyToolDetail is the tool and the argument that says most about the call.
func agyToolDetail(tool agyTool) string {
	keys := make([]string, 0, len(tool.Args))
	for k := range tool.Args {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range append([]string{"CommandLine", "AbsolutePath", "TargetFile", "Query", "Url"}, keys...) {
		if v, ok := tool.Args[key].(string); ok && v != "" {
			return tool.Name + " " + v
		}
	}
	return tool.Name
}

// lastReply is the text of the latest model reply in the transcript at path, "" when there is none to read.
func lastReply(path string) string {
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := bytes.Split(data, []byte("\n"))
	for i := len(lines) - 1; i >= 0; i-- {
		if !bytes.Contains(lines[i], []byte(`"PLANNER_RESPONSE"`)) {
			continue
		}
		var step struct {
			Type    string `json:"type"`
			Content string `json:"content"`
		}
		if json.Unmarshal(lines[i], &step) == nil && step.Type == "PLANNER_RESPONSE" && step.Content != "" {
			return step.Content
		}
	}
	return ""
}
