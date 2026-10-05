// Package prompts holds the texts agentos gives to agents and prints for them, as Markdown files.
// Each file starts with an HTML comment that names its {token} placeholders; the callers fill them in.
package prompts

import (
	_ "embed"
	"strings"
)

var (
	//go:embed harness-check.md
	harnessCheck string
	//go:embed digest.md
	digest string
	//go:embed browser-session.md
	browserSession string
	//go:embed browser-help.md
	browserHelp string
)

// HarnessCheck is typed into the "Harness check" session.
func HarnessCheck() string { return body(harnessCheck) }

// Digest is what the weekly digest run asks Claude to do.
func Digest() string { return body(digest) }

// BrowserSession is appended to a session's system prompt when it has a browser.
func BrowserSession() string { return body(browserSession) }

// BrowserHelp is what agentos browser help prints.
func BrowserHelp() string { return body(browserHelp) }

// body drops the leading comment and the final newline of a file.
func body(file string) string {
	if strings.HasPrefix(file, "<!--") {
		_, file, _ = strings.Cut(file, "-->\n\n")
	}
	return strings.TrimSuffix(file, "\n")
}
