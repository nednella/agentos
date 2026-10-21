package session

import (
	"slices"
	"sort"
	"time"
)

type State string

const (
	Idle    State = "idle"    // started, or a turn ended: the user's move
	Working State = "working" // the agent is running
	Waiting State = "waiting" // blocked on the user
	Ended   State = "ended"   // the agent's session is gone
)

// Record is what a hook leaves behind: in a state file and on the bus.
type Record struct {
	Session      string    `json:"session"`
	State        State     `json:"state"`
	Event        string    `json:"event"`
	At           time.Time `json:"at"`
	Detail       string    `json:"detail,omitempty"`
	Tool         string    `json:"tool,omitempty"`         // the latest tool call, kept through the prompt it leads to
	Command      string    `json:"command,omitempty"`      // that call's shell command
	Notify       string    `json:"notify,omitempty"`       // notification_type of a Notification
	Cwd          string    `json:"cwd,omitempty"`          // where the agent was working at the latest event
	Conversation string    `json:"conversation,omitempty"` // the agent's id for its conversation, to resume it
	Transcript   string    `json:"transcript,omitempty"`   // the file the agent writes its conversation to
	AgentModel   string    `json:"agentModel,omitempty"`   // the model the agent says it runs, when its hooks say; prices a transcript that names none
	Changes      []Change  `json:"changes,omitempty"`      // the latest state changes, oldest first, so the timeline outlives the app

	Identity
	EndedAt int64 `json:"endedAt,omitempty"` // unix ms, set by the desktop app when it sees the session end
}

// Identity is what the desktop app started a session as. The app writes it when the session starts, so a
// session that ends while the app is closed comes back as itself. Model and Effort follow the agent: the start
// choice until its first turn, then what its transcript or its hooks say it used last.
type Identity struct {
	Title   string `json:"title,omitempty"`
	Issue   int    `json:"issue,omitempty"`
	Chat    bool   `json:"chat,omitempty"`
	Action  string `json:"action,omitempty"`
	Model   string `json:"model,omitempty"`
	Effort  string `json:"effort,omitempty"`
	Agent   string `json:"agent,omitempty"` // the provider, or the plain command, the session was started with
	Path    string `json:"path,omitempty"`
	Created int64  `json:"created,omitempty"` // unix ms
}

// Change is one entry of a session's state timeline.
type Change struct {
	State State `json:"state"`
	At    int64 `json:"at"` // unix ms
}

// ChangesCap is how many changes a timeline keeps.
const ChangesCap = 200

// AddChange appends c to changes and keeps the latest ChangesCap of them. It never writes into the array of changes.
func AddChange(changes []Change, c Change) []Change {
	out := append(slices.Clip(changes), c)
	if len(out) > ChangesCap {
		out = slices.Clone(out[len(out)-ChangesCap:])
	}
	return out
}

// Session is one agent as the screen sees it.
type Session struct {
	Name   Name
	N      int // the number shown, 0 when none was given
	Title  string
	State  State
	At     time.Time
	Detail string
}

func (s State) rank() int {
	switch s {
	case Waiting:
		return 0
	case Idle:
		return 1
	case Working:
		return 2
	}
	return 3
}

// Sort orders sessions by who needs the user most: waiting, idle, working, ended, then most recent event first.
func Sort(ss []Session) {
	sort.SliceStable(ss, func(i, j int) bool {
		a, b := ss[i], ss[j]
		if a.State.rank() != b.State.rank() {
			return a.State.rank() < b.State.rank()
		}
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		if a.N != b.N {
			return a.N < b.N
		}
		return a.Name.String() < b.Name.String()
	})
}
