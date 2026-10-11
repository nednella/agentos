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
	//go:embed browser.md
	browser string
	//go:embed browser-help.md
	browserHelp string
	//go:embed setup-brief.md
	setupBrief string
	//go:embed setup-start.md
	setupStart string
	//go:embed setup-review.md
	setupReview string
	//go:embed setup-checks.md
	setupChecks string
	//go:embed setup-conflict.md
	setupConflict string
	//go:embed start-issue.md
	startIssue string
	//go:embed session.md
	session string
	//go:embed task.md
	task string
	//go:embed chat.md
	chat string
	//go:embed handover.md
	handover string
)

// Digest is what the weekly digest run asks Claude to do, given the names the project depends on.
func Digest(dependencies []string) string {
	return strings.Replace(body(digest), "{dependencies}", strings.Join(dependencies, "\n"), 1)
}

// Browser is appended to a task's system prompt when it has a browser.
func Browser() string { return body(browser) }

// Session starts the system prompt of every Claude session, task or chat: where it runs and the commands every session uses.
func Session() string { return body(session) }

// Task follows Session in the system prompt of a task, which works on the project: its commands and branches.
func Task() string { return body(task) }

// Chat follows Session in the system prompt of a chat: why it runs in plan mode and how it hands work off.
func Chat() string { return body(chat) }

// Handover ends the system prompt of a task that another session asked for, with the context that session gave.
func Handover(context string) string { return strings.Replace(body(handover), "{context}", context, 1) }

// BrowserHelp is what agentos browser help prints.
func BrowserHelp() string { return body(browserHelp) }

// SetupBrief is added to the system prompt of the session that sets up a project's repository.
func SetupBrief() string { return body(setupBrief) }

// SetupStart is what the session that sets up a project's repository starts with.
func SetupStart() string { return body(setupStart) }

// SetupReview is the pr_review_command that setting up a project writes; {n} is the PR number.
func SetupReview() string { return body(setupReview) }

// SetupChecks is the pr_checks_command that setting up a project writes; {n} is the PR number.
func SetupChecks() string { return body(setupChecks) }

// SetupConflict is the pr_conflict_command that setting up a project writes; {n} is the PR number.
func SetupConflict() string { return body(setupConflict) }

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
