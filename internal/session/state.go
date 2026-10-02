package session

import (
	"sort"
	"time"
)

type State string

const (
	Idle     State = "idle"
	Working  State = "working"
	Waiting  State = "waiting"
	Finished State = "finished"
	Gone     State = "gone"
)

// Record is what a hook leaves behind: in a state file and on the bus.
type Record struct {
	Session string    `json:"session"`
	State   State     `json:"state"`
	Event   string    `json:"event"`
	At      time.Time `json:"at"`
	Detail  string    `json:"detail,omitempty"`
	Tool    string    `json:"tool,omitempty"`    // the latest tool call, kept through the prompt it leads to
	Command string    `json:"command,omitempty"` // that call's shell command
	Notify  string    `json:"notify,omitempty"`  // notification_type of a Notification
}

// Session is one agent as the screen sees it.
type Session struct {
	Name   Name
	Title  string
	State  State
	At     time.Time
	Detail string
}

// NeedsUser reports whether the ball is in the user's court.
func (s State) NeedsUser() bool { return s == Waiting || s == Finished }

func (s State) rank() int {
	switch s {
	case Waiting:
		return 0
	case Finished:
		return 1
	case Working:
		return 2
	}
	return 3
}

// Sort orders sessions by who needs the user most, then most recent event first.
func Sort(ss []Session) {
	sort.SliceStable(ss, func(i, j int) bool {
		a, b := ss[i], ss[j]
		if a.State.rank() != b.State.rank() {
			return a.State.rank() < b.State.rank()
		}
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		return a.Name.N < b.Name.N
	})
}
