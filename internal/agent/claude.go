package agent

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/nednella/agentos/internal/util"
)

var claudeEvents = []string{
	"SessionStart", "UserPromptSubmit", "PreToolUse", "PostToolUse",
	"Notification", "Stop", "SessionEnd",
}

// Claude runs Claude Code with hooks that call back into the agentos binary at Exe.
// The hooks come from --settings, so they apply to our sessions only.
// ConfigDir holds the user's own settings.json.
type Claude struct{ Exe, ConfigDir string }

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
	if l.Resume != "" {
		argv = append(argv, "--resume", l.Resume)
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
