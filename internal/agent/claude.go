package agent

import "encoding/json"

// BrowserPrompt tells a session about the browser and the evidence commands.
const BrowserPrompt = "A browser is available to you through the agentos command: run `agentos browser help` to see how. " +
	"Use it to check your own UI work in a real page. File screenshots of what you built with " +
	"`agentos browser screenshot --caption \"...\"`, and use `agentos show` for other evidence the owner should see."

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

func (c Claude) Command(string) []string {
	hooks := make(map[string][]hookGroup, len(claudeEvents))
	for _, ev := range claudeEvents {
		hooks[ev] = []hookGroup{{Hooks: []hookCommand{{Type: "command", Command: shellQuote(c.Exe) + " hook " + ev}}}}
	}
	settings, _ := json.Marshal(struct {
		Hooks map[string][]hookGroup `json:"hooks"`
	}{hooks})
	return []string{"claude", "--settings", string(settings)}
}
