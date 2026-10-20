package session

import (
	"cmp"
	"strings"
	"time"
)

// Event is a hook call, in Claude Code's event names, which every agent's provider maps its own onto.
type Event struct {
	Name             string
	NotificationType string
	Message          string
	Detail           string
	Tool             string // the tool of a PreToolUse or PostToolUse
	Command          string // that tool's shell command, when it has one
	Cwd              string // where the agent was working
	Conversation     string // the agent's id for its conversation
	Transcript       string // the file the agent writes its conversation to
	Asks             bool   // a Stop whose reply ends in a question to the owner
}

// Next is the state reducer: the state after ev, given the state before it.
func Next(cur State, ev Event) State {
	switch ev.Name {
	case "SessionStart":
		return Idle
	case "Stop":
		// A session that asks in text, not with AskUserQuestion, still waits on the owner.
		if ev.Asks {
			return Waiting
		}
		return Idle
	case "UserPromptSubmit", "PreToolUse", "PostToolUse":
		return Working
	case "Notification":
		if isIdleReminder(ev) {
			// A turn that ended in an error fires no Stop, so the reminder is what stops it showing as working.
			if cur == Working {
				return Idle
			}
			return cur
		}
		return Waiting
	case "SessionEnd":
		return Ended
	}
	return cur
}

// isIdleReminder spots the "waiting for your input" nudge Claude sends a while
// after a turn ends.
func isIdleReminder(ev Event) bool {
	if ev.NotificationType != "" {
		return ev.NotificationType == "idle_prompt"
	}
	return strings.Contains(strings.ToLower(ev.Message), "waiting for your input")
}

// Apply builds the record a hook leaves for ev on top of the previous record.
// An idle reminder leaves the previous record alone unless the session was
// working: then it is the sign that the turn is over.
func Apply(name string, prev Record, ev Event, now time.Time) Record {
	if prev.State == "" {
		prev = Record{Session: name, State: Idle, Event: ev.Name, At: now, Identity: prev.Identity}
		if isIdleReminder(ev) {
			prev.Changes = []Change{{State: Idle, At: now.UnixMilli()}}
			return prev
		}
	} else if isIdleReminder(ev) && prev.State != Working {
		return prev
	}
	rec := Record{Session: name, State: Next(prev.State, ev), Event: ev.Name, At: now, Detail: ev.Detail, Notify: ev.NotificationType, Cwd: cmp.Or(ev.Cwd, prev.Cwd), Conversation: cmp.Or(ev.Conversation, prev.Conversation), Transcript: cmp.Or(ev.Transcript, prev.Transcript), Changes: prev.Changes, Identity: prev.Identity}
	// A record with no changes, the first or one saved before records kept them, starts its timeline here.
	if rec.State != prev.State || len(prev.Changes) == 0 {
		rec.Changes = AddChange(prev.Changes, Change{State: rec.State, At: now.UnixMilli()})
	}
	switch ev.Name {
	case "PreToolUse", "PostToolUse":
		rec.Tool, rec.Command = ev.Tool, ev.Command
	case "Notification", "Stop":
		// The prompt that follows a tool call is about that tool.
		rec.Tool, rec.Command = prev.Tool, prev.Command
	}
	return rec
}
