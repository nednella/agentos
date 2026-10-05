package session

import (
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
	Session string    `json:"session"`
	State   State     `json:"state"`
	Event   string    `json:"event"`
	At      time.Time `json:"at"`
	Detail  string    `json:"detail,omitempty"`

	// What the desktop app records when a session ends, so the row can outlive it.
	Title   string `json:"title,omitempty"`
	Issue   int    `json:"issue,omitempty"`
	Path    string `json:"path,omitempty"`
	Created int64  `json:"created,omitempty"` // unix ms
	EndedAt int64  `json:"endedAt,omitempty"` // unix ms
}

// Session is one agent as the screen sees it.
type Session struct {
	Name   Name
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
		return a.Name.N < b.Name.N
	})
}
