// Package prompts holds the texts agentos gives to agents and prints for them, as Markdown files.
// Each file starts with an HTML comment that names its {token} placeholders; the callers fill them in.
package prompts

import (
	_ "embed"
	"strings"
)

var (
	//go:embed digest.md
	digest string
	//go:embed browser-session.md
	browserSession string
	//go:embed browser-help.md
	browserHelp string
	//go:embed setup-session.md
	setupSession string
	//go:embed setup-help.md
	setupHelp string
	//go:embed start-issue.md
	startIssue string
	//go:embed new-session.md
	newSession string
)

// Digest is what the weekly digest run asks Claude to do, given the names the project depends on.
func Digest(dependencies []string) string {
	return strings.Replace(body(digest), "{dependencies}", strings.Join(dependencies, "\n"), 1)
}

// BrowserSession is appended to a session's system prompt when it has a browser.
func BrowserSession() string { return body(browserSession) }

// NewSession is appended to every Claude session's system prompt: how to take a topic into a session of its own.
func NewSession() string { return body(newSession) }

// BrowserHelp is what agentos browser help prints.
func BrowserHelp() string { return body(browserHelp) }

// SetupSession is appended to every Claude session's system prompt: where to read how to set up a project.
func SetupSession() string { return body(setupSession) }

// SetupHelp is what agentos setup help prints.
func SetupHelp() string { return body(setupHelp) }

// StartIssue is what a session for an issue is started with when the project defines no action for it.
// {n} is the issue number and {title} its title.
func StartIssue() string { return body(startIssue) }

// body drops the leading comment and the final newline of a file.
func body(file string) string {
	if strings.HasPrefix(file, "<!--") {
		_, file, _ = strings.Cut(file, "-->\n\n")
	}
	return strings.TrimSuffix(file, "\n")
}
