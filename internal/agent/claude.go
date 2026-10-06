package agent

import (
	"encoding/json"

	"github.com/nednella/agentos/internal/util"
)

var claudeEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Notification", "Stop", "SessionEnd",
}

// Claude runs Claude Code with hooks that call back into the agentos binary at Exe.
// The hooks come from --settings, so they apply to our sessions only.
type Claude struct{ Exe string }

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type hookGroup struct {
	Hooks []hookCommand `json:"hooks"`
}

func (c Claude) Command(_ string, l Launch) []string {
	hooks := make(map[string][]hookGroup, len(claudeEvents))
	for _, ev := range claudeEvents {
		hooks[ev] = []hookGroup{{Hooks: []hookCommand{{Type: "command", Command: util.ShellQuote(c.Exe) + " hook " + ev}}}}
	}
	settings, _ := json.Marshal(struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}{hooks})
	argv := []string{"claude", "--settings", string(settings)}
	if l.Model != "" {
		argv = append(argv, "--model", l.Model)
	}
	if l.Effort != "" {
		argv = append(argv, "--effort", l.Effort)
	}
	return argv
}
