package usage

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

// AgyReader reads what an agy session used from its transcript: one line per step, where a PLANNER_RESPONSE line
// is one model call. The lines name no model, so each read is priced as the model given to it, when that is a Google
// model: an agy session runs no other.
type AgyReader struct {
	after time.Time
	tail
	seen map[int64]bool
}

// NewAgyReader counts only the steps written after the time given, like NewCursor.
func NewAgyReader(after time.Time) *AgyReader {
	return &AgyReader{after: after, tail: newTail(), seen: map[int64]bool{}}
}

// After is the time of the latest step read.
func (a *AgyReader) After() time.Time { return a.after }

// agyLine is the part of an agy transcript line that tells what a model call used.
type agyLine struct {
	StepIndex       int64     `json:"step_index"`
	Type            string    `json:"type"`
	CreatedAt       time.Time `json:"created_at"`
	InputTokens     int64     `json:"input_tokens"`
	CacheReadTokens int64     `json:"cache_read_tokens"`
	OutputTokens    int64     `json:"output_tokens"`
}

// Read counts the model calls written to the transcript at path since the last read, as one part of model. What the
// agent used last is left empty: the lines name no model, and the agy provider knows it from its hooks.
func (a *AgyReader) Read(path, model string) ([]Part, Current, error) {
	var sum Tokens
	latest := a.after
	err := a.lines(path, func(b []byte, known bool) {
		if !bytes.Contains(b, []byte(`"PLANNER_RESPONSE"`)) {
			return
		}
		var l agyLine
		if json.Unmarshal(b, &l) != nil || l.Type != "PLANNER_RESPONSE" {
			return
		}
		if !known && !l.CreatedAt.Truncate(time.Millisecond).After(a.after) {
			return
		}
		if l.CreatedAt.After(latest) {
			latest = l.CreatedAt
		}
		if a.seen[l.StepIndex] {
			return
		}
		a.seen[l.StepIndex] = true
		u := apiUsage{InputTokens: l.InputTokens, OutputTokens: l.OutputTokens, CacheReadInputTokens: l.CacheReadTokens}
		var c int64
		if strings.HasPrefix(model, "gemini-") {
			c = cost(model, u, l.CreatedAt)
		}
		sum.Add(Tokens{Input: l.InputTokens, Output: l.OutputTokens, CacheRead: l.CacheReadTokens, Cost: c})
	})
	if err != nil {
		return nil, Current{}, err
	}
	a.after = latest
	if sum.Total() == 0 {
		return nil, Current{}, nil
	}
	return []Part{{Model: model, Tokens: sum}}, Current{}, nil
}
