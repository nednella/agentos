package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

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

// allowTool is a PreToolUse hook command that lets the tool run. Plan mode asks before any page tool that changes
// or reads a page, and no allow rule changes that; a hook's answer does.
const allowTool = `printf '%s' '{"hookSpecificOutput":{"hookEventName":"PreToolUse","permissionDecision":"allow"}}'`

func (c Claude) Command(_ string, l Launch) []string {
	hooks := make(map[string][]hookGroup, len(claudeEvents))
	for _, ev := range claudeEvents {
		hooks[ev] = []hookGroup{{Hooks: []hookCommand{{Type: "command", Command: util.ShellQuote(c.Exe) + " hook " + ev}}}}
	}
	if l.Chat && l.Browser {
		hooks["PreToolUse"] = append(hooks["PreToolUse"], hookGroup{Matcher: "mcp__browser__.*", Hooks: []hookCommand{{Type: "command", Command: allowTool}}})
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
	return argv
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
