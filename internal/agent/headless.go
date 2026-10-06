package agent

import (
	"slices"
	"strings"
)

// Headless is an agent that can answer one prompt with no one at the terminal.
type Headless struct {
	Name string
	// Check exits 0 when the agent is installed and signed in. It must not spend a request.
	Check []string
	// Argv is the command that runs prompt with the web and `agentos digest add`, and nothing else.
	Argv func(prompt string) []string
	// Env keeps what the agent needs to start, find its login and reach its provider.
	Env func(environ []string) []string
}

// Headlesses are the agents the digest can run on, in the order it tries them.
var Headlesses = []Headless{ClaudeHeadless}

// ClaudeHeadless runs `claude -p`, which uses the login of the claude command.
var ClaudeHeadless = Headless{
	Name:  "claude",
	Check: []string{"claude", "auth", "status"},
	Argv: func(prompt string) []string {
		return []string{"claude", "-p", prompt, "--allowedTools", "WebSearch WebFetch Bash(agentos digest add:*)"}
	},
	Env: func(environ []string) []string {
		exact := []string{"PATH", "HOME", "USER", "LOGNAME", "TMPDIR", "LANG", "LC_ALL", "SSL_CERT_FILE", "NODE_EXTRA_CA_CERTS",
			"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "GOOGLE_APPLICATION_CREDENTIALS"}
		prefixes := []string{"ANTHROPIC_", "CLAUDE_", "AWS_", "CLOUD_ML_", "VERTEX_"}
		var out []string
		for _, kv := range environ {
			name, _, _ := strings.Cut(kv, "=")
			if slices.Contains(exact, name) || slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(name, p) }) {
				out = append(out, kv)
			}
		}
		return out
	},
}
