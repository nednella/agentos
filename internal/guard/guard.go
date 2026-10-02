// Package guard decides which shell commands an agent may not run.
package guard

import (
	"encoding/json"
	"os"
	"regexp"
)

// Env holds the patterns as a JSON array, set on each session by agentos.
const Env = "AGENTOS_GUARD"

// Reason is shown to the agent when a command is denied.
const Reason = "agentos: merging, marking ready, requesting reviewers and force pushes are the owner's to do."

// Defaults are the commands no agent may run: they publish or merge work the owner has not looked at.
// --force-with-lease stays allowed.
var Defaults = []string{
	`gh\s+pr\s+merge`,
	`gh\s+pr\s+ready`,
	`gh\s+pr\s+(edit|create)\b.*--(add-)?reviewer`,
	`gh\s+api\b.*requested_reviewers`,
	`git\s+push\b.*(--force(\s|$)|\s-f(\s|$)|\s\+\S)`,
}

// Denied reports whether command matches a pattern. A pattern that does not
// compile never matches: the guard must not stop work because of its own mistake.
func Denied(patterns []string, command string) bool {
	for _, p := range patterns {
		if re, err := regexp.Compile("(?s)" + p); err == nil && re.MatchString(command) {
			return true
		}
	}
	return false
}

// FromEnv reads the patterns the session was started with; without any, the defaults apply.
func FromEnv() []string {
	raw, ok := os.LookupEnv(Env)
	if !ok {
		return Defaults
	}
	var patterns []string
	if json.Unmarshal([]byte(raw), &patterns) != nil {
		return Defaults
	}
	return patterns
}

// Encode is the value for Env.
func Encode(patterns []string) string {
	b, _ := json.Marshal(patterns)
	return string(b)
}
