package session

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"
)

// interruptMarker starts the message Claude Code writes to the transcript when the user presses Esc.
// No hook fires for it.
const interruptMarker = "[Request interrupted by user"

// turnTail is how much of the end of a transcript is read: the marker is the last message of a
// turn, and only short bookkeeping lines follow it.
const turnTail = 64 << 10

// Turn is what a transcript tells of the agent's turn since a given time.
type Turn struct {
	Interrupted time.Time // when the user stopped it with Esc, or zero
	Replied     bool      // whether the agent wrote a message
}

// ReadTurn reads the end of the transcript at path for what happened in the turn since the given time.
func ReadTurn(path string, since time.Time) Turn {
	f, err := os.Open(path)
	if err != nil {
		return Turn{}
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.ModTime().After(since) {
		return Turn{}
	}
	start := max(0, info.Size()-turnTail)
	data, err := io.ReadAll(io.NewSectionReader(f, start, info.Size()-start))
	if err != nil {
		return Turn{}
	}
	if start > 0 {
		_, data, _ = bytes.Cut(data, []byte("\n")) // the first line is only a part of one
	}
	var turn Turn
	for line := range bytes.SplitSeq(data, []byte("\n")) {
		var entry struct {
			Type      string    `json:"type"`
			Timestamp time.Time `json:"timestamp"`
			Message   struct {
				Content json.RawMessage `json:"content"`
			} `json:"message"`
		}
		if json.Unmarshal(line, &entry) != nil || !entry.Timestamp.After(since) {
			continue
		}
		switch {
		case entry.Type == "assistant":
			turn.Replied = true
		case entry.Type == "user" && isInterrupt(entry.Message.Content) && entry.Timestamp.After(turn.Interrupted):
			turn.Interrupted = entry.Timestamp
		}
	}
	return turn
}

// isInterrupt says whether a user message's content, a string or a list of parts, is the interrupt marker.
func isInterrupt(content json.RawMessage) bool {
	var text string
	if json.Unmarshal(content, &text) == nil {
		return strings.HasPrefix(text, interruptMarker)
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(content, &parts)
	for _, p := range parts {
		if p.Type == "text" && strings.HasPrefix(p.Text, interruptMarker) {
			return true
		}
	}
	return false
}
