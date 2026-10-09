// Package activity is what the app records of what it does, one event at a time.
package activity

// The kinds of event.
const (
	Prompt       = "prompt"
	SessionStart = "session_start" // label "resumed" when it continues a conversation
	SessionEnd   = "session_end"   // ms is how long the session ran
	Worked       = "worked"        // ms is how long a session worked
	Wait         = "wait"          // ms is how long a session waited on the user; label is why
	PROpened     = "pr_opened"
	PRMerged     = "pr_merged" // ms is the time from first seen open
	PRClosed     = "pr_closed" // ms is the time from first seen open
	Review       = "review"    // a review command was typed into the session
	Checks       = "checks"    // a checks command was typed into the session
	Conflict     = "conflict"  // a conflict command was typed into the session
	IssueFiled   = "issue_filed"
	Cleanup      = "cleanup" // label done or blocked
	Evidence     = "evidence"
)

// Resumed is the label of a session that continues an earlier session's conversation.
const Resumed = "resumed"

// Event is one thing the app did or saw.
type Event struct {
	At      int64  `json:"at"` // unix ms
	Kind    string `json:"kind"`
	Session string `json:"session,omitempty"`
	Issue   int    `json:"issue,omitempty"`
	PR      int    `json:"pr,omitempty"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
	Ms      int64  `json:"ms,omitempty"`
	Label   string `json:"label,omitempty"`
}
