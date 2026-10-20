package agent

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/usage"
	"github.com/nednella/agentos/internal/util"
)

// chatTools are the commands a chat may run in plan mode: the ones that hand work to the owner's queue, plan
// an issue in it, or drive the session's own browser window.
var chatTools = []string{
	"Bash(agentos issue:*)", "Bash(agentos new:*)", "Bash(agentos note:*)", "Bash(agentos refresh:*)", "Bash(agentos browser:*)",
	"Bash(gh issue create:*)", "Bash(gh issue comment:*)", "Bash(gh issue edit:*)",
}

var claudeEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Notification", "Stop", "SessionEnd",
}

// Claude runs Claude Code with hooks that call back into the agentos binary at Exe.
// The hooks come from --settings, so they apply to our sessions only.
// ConfigDir holds the user's own settings.json. Dirs are the app's own folders, which the
// agent may use without asking, so a fresh claude install does not prompt for them.
type Claude struct {
	Exe, ConfigDir string
	Dirs           []string
}

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookGroup struct {
	Matcher string        `json:"matcher,omitempty"`
	Hooks   []hookCommand `json:"hooks"`
}

// allowTool is a PermissionRequest hook command that answers the prompt with yes. Plan mode asks before a page
// tool that changes the page, and neither an allow rule nor a PreToolUse hook's allow stops that.
const allowTool = `printf '%s' '{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"allow"}}}'`

func (c Claude) Name() string        { return "claude" }
func (c Claude) Can(Capability) bool { return true }

// Command is claude's command line. The "--" before the start prompt keeps one that starts with "-" from being
// read as a flag, and from being taken as one more value of a list flag such as --allowedTools.
func (c Claude) Command(l Launch) []string {
	hooks := make(map[string][]hookGroup, len(claudeEvents))
	for _, ev := range claudeEvents {
		hooks[ev] = []hookGroup{{Hooks: []hookCommand{{Type: "command", Command: util.ShellQuote(c.Exe) + " hook " + ev}}}}
	}
	browser := l.MCP.Command != ""
	if l.Chat && browser {
		hooks["PermissionRequest"] = []hookGroup{{Matcher: "mcp__browser__.*", Hooks: []hookCommand{{Type: "command", Command: allowTool}}}}
	}
	type permissions struct {
		AdditionalDirectories []string `json:"additionalDirectories,omitempty"`
	}
	settings, _ := json.Marshal(struct {
		Hooks       map[string][]hookGroup `json:"hooks"`
		Permissions permissions            `json:"permissions"`
	}{hooks, permissions{c.Dirs}})
	argv := []string{"claude", "--settings", string(settings)}
	if l.Model != "" {
		argv = append(argv, "--model", l.Model)
	}
	if l.Effort != "" {
		argv = append(argv, "--effort", l.Effort)
	}
	if l.Resume != "" {
		argv = append(argv, "--resume", l.Resume)
	}
	if l.Chat {
		argv = append(argv, "--permission-mode", "plan", "--allowedTools", strings.Join(chatTools, ","))
	}
	if browser {
		argv = append(argv, "--mcp-config", c.mcpConfig(l.MCP), "--disallowedTools", deniedTools(l.MCP))
	}
	if l.Prompt != "" {
		argv = append(argv, "--append-system-prompt", l.Prompt)
	}
	if l.Start != "" {
		argv = append(argv, "--", l.Start)
	}
	return argv
}

// mcpConfig is the --mcp-config JSON that adds m as the MCP server "browser". agentos, at c.Exe, stands in for the
// server and starts it at the first tool call, so a session that never uses the browser never runs it.
func (c Claude) mcpConfig(m MCPServer) string {
	type server struct {
		Command string            `json:"command"`
		Args    []string          `json:"args"`
		Env     map[string]string `json:"env"`
	}
	config, _ := json.Marshal(map[string]map[string]server{"mcpServers": {"browser": {
		Command: c.Exe, Args: append([]string{"mcp-browser", "--", m.Command}, m.Args...), Env: m.Env,
	}}})
	return string(config)
}

// deniedTools is the --disallowedTools value for the denied tools of m, in claude's name for an MCP tool.
func deniedTools(m MCPServer) string {
	names := make([]string, len(m.Denied))
	for i, tool := range m.Denied {
		names[i] = "mcp__browser__" + tool
	}
	return strings.Join(names, " ")
}

// Env sets claude's temp folder to the session's own.
func (c Claude) Env(l Launch) []string { return []string{"CLAUDE_CODE_TMPDIR=" + l.TmpDir} }

func (c Claude) ValidModel(model, effort string) error {
	return project.Model{Model: model, Effort: effort}.Validate()
}

func (c Claude) Usage(after time.Time) usage.Reader { return usage.NewCursor(after) }

type payload struct {
	NotificationType     string         `json:"notification_type"`
	Message              string         `json:"message"`
	ToolName             string         `json:"tool_name"`
	ToolInput            map[string]any `json:"tool_input"`
	Prompt               string         `json:"prompt"`
	LastAssistantMessage string         `json:"last_assistant_message"`
	Cwd                  string         `json:"cwd"`
	SessionID            string         `json:"session_id"`
	TranscriptPath       string         `json:"transcript_path"`
}

// ParseEvent reads the hook JSON from r. A body that is not JSON still gives a
// usable event, because the event name alone moves the state.
func (c Claude) ParseEvent(name string, r io.Reader) (session.Event, bool) {
	var p payload
	_ = json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&p)
	ev := session.Event{Name: name, NotificationType: p.NotificationType, Message: p.Message, Cwd: p.Cwd, Conversation: p.SessionID, Transcript: p.TranscriptPath}
	switch name {
	case "Notification":
		ev.Detail = p.Message
	case "PreToolUse", "PostToolUse":
		ev.Detail = toolDetail(p)
	case "UserPromptSubmit":
		ev.Detail = p.Prompt
	case "Stop":
		ev.Detail = p.LastAssistantMessage
		ev.Asks = strings.HasSuffix(strings.TrimRight(p.LastAssistantMessage, " \t\n*_"), "?")
	}
	ev.Detail = oneLine(ev.Detail, 120)
	if name == "PreToolUse" || name == "PostToolUse" {
		ev.Tool = p.ToolName
		if cmd, ok := p.ToolInput["command"].(string); ok {
			ev.Command = oneLine(cmd, 200)
		}
	}
	return ev, true
}

func toolDetail(p payload) string {
	for _, key := range []string{"command", "file_path", "pattern", "url", "description"} {
		if v, ok := p.ToolInput[key].(string); ok && v != "" {
			return p.ToolName + " " + v
		}
	}
	return p.ToolName
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

type claudeSettings struct {
	Model         string `json:"model"`
	EffortLevel   string `json:"effortLevel"`
	ModelSettings map[string]struct {
		EffortLevel string `json:"effortLevel"`
	} `json:"modelSettings"`
}

// Fill reads claude's settings files, the project's local and shared ones before the user's, so the first
// that sets a value wins. A model's own effort in modelSettings wins over effortLevel. A file that is
// missing or not JSON is skipped.
func (c Claude) Fill(dir string, l Launch) Launch {
	paths := []string{filepath.Join(dir, ".claude", "settings.local.json"), filepath.Join(dir, ".claude", "settings.json")}
	if c.ConfigDir != "" {
		paths = append(paths, filepath.Join(c.ConfigDir, "settings.json"))
	}
	var files []claudeSettings
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var s claudeSettings
		if json.Unmarshal(data, &s) == nil {
			files = append(files, s)
		}
	}
	for _, s := range files {
		if l.Model == "" {
			l.Model = s.Model
		}
	}
	for _, s := range files {
		if l.Effort == "" {
			l.Effort = s.ModelSettings[l.Model].EffortLevel
		}
	}
	for _, s := range files {
		if l.Effort == "" {
			l.Effort = s.EffortLevel
		}
	}
	return l
}
