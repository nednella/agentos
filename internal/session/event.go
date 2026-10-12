package session

import (
	"cmp"
	"encoding/json"
	"io"
	"strings"
	"time"
)

// Event is a Claude Code hook call.
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
}

type payload struct {
	NotificationType     string         `json:"notification_type"`
	Message              string         `json:"message"`
	ToolName             string         `json:"tool_name"`
	ToolInput            map[string]any `json:"tool_input"`
	Prompt               string         `json:"prompt"`
	LastAssistantMessage string         `json:"last_assistant_message"`
	Cwd                  string         `json:"cwd"`
	SessionID            string         `json:"session_id"`
	TranscriptPath       string         `json:"transcript_path"`
}

// ParseEvent reads the hook JSON from r. A body that is not JSON still gives a
// usable event, because the event name alone moves the state.
func ParseEvent(name string, r io.Reader) Event {
	var p payload
	_ = json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&p)
	ev := Event{Name: name, NotificationType: p.NotificationType, Message: p.Message, Cwd: p.Cwd, Conversation: p.SessionID, Transcript: p.TranscriptPath}
	switch name {
	case "Notification":
		ev.Detail = p.Message
	case "PreToolUse", "PostToolUse":
		ev.Detail = toolDetail(p)
	case "UserPromptSubmit":
		ev.Detail = p.Prompt
	case "Stop":
		ev.Detail = p.LastAssistantMessage
	}
	ev.Detail = oneLine(ev.Detail, 120)
	if name == "PreToolUse" || name == "PostToolUse" {
		ev.Tool = p.ToolName
		if cmd, ok := p.ToolInput["command"].(string); ok {
			ev.Command = oneLine(cmd, 200)
		}
	}
	return ev
}

func toolDetail(p payload) string {
	for _, key := range []string{"command", "file_path", "pattern", "url", "description"} {
		if v, ok := p.ToolInput[key].(string); ok && v != "" {
			return p.ToolName + " " + v
		}
	}
	return p.ToolName
}

func oneLine(s string, max int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > max {
		return string(r[:max-1]) + "…"
	}
	return s
}

// Next is the state reducer: the state after ev, given the state before it.
func Next(cur State, ev Event) State {
	switch ev.Name {
	case "SessionStart", "Stop":
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
		prev = Record{Session: name, State: Idle, Event: ev.Name, At: now}
		if isIdleReminder(ev) {
			prev.Changes = []Change{{State: Idle, At: now.UnixMilli()}}
			return prev
		}
	} else if isIdleReminder(ev) && prev.State != Working {
		return prev
	}
	rec := Record{Session: name, State: Next(prev.State, ev), Event: ev.Name, At: now, Detail: ev.Detail, Notify: ev.NotificationType, Cwd: cmp.Or(ev.Cwd, prev.Cwd), Conversation: cmp.Or(ev.Conversation, prev.Conversation), Transcript: cmp.Or(ev.Transcript, prev.Transcript), Changes: prev.Changes}
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
