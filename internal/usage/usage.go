// Package usage reads the tokens a Claude Code session used from its transcripts, and prices them.
package usage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// Tokens is what API calls used, and what they cost at list prices.
type Tokens struct {
	Input      int64 `json:"input,omitempty"`
	Output     int64 `json:"output,omitempty"` // thinking included
	CacheRead  int64 `json:"cacheRead,omitempty"`
	CacheWrite int64 `json:"cacheWrite,omitempty"`
	Thinking   int64 `json:"thinking,omitempty"`
	Cost       int64 `json:"cost,omitempty"` // millionths of a US dollar
}

// Total is every token the calls sent or got back.
func (t Tokens) Total() int64 { return t.Input + t.Output + t.CacheRead + t.CacheWrite }

// Dollars is the cost in US dollars.
func (t Tokens) Dollars() float64 { return float64(t.Cost) / 1e6 }

func (t *Tokens) Add(o Tokens) {
	t.Input += o.Input
	t.Output += o.Output
	t.CacheRead += o.CacheRead
	t.CacheWrite += o.CacheWrite
	t.Thinking += o.Thinking
	t.Cost += o.Cost
}

// Part is what one model used, in the session itself or in its subagents.
type Part struct {
	Model    string
	Subagent bool
	Tokens
}

// Reader reads what a session's agent used from its transcript, a little at a time, as its provider knows how.
type Reader interface {
	// Read counts what was written to the transcript at path since the last read, one part for each model.
	Read(path string) ([]Part, error)
	// After is the time of the latest message read.
	After() time.Time
}

// Cursor reads a session's transcripts a little at a time. Claude Code writes each API response
// once per content block, so a message is counted once by its id.
type Cursor struct {
	after   time.Time // a file read for the first time is counted from this time on
	offsets map[string]int64
	seen    map[string]bool
}

// NewCursor counts only the messages written after the time given: the ones before were counted
// by an earlier run of the app or belong to an earlier session of a resumed conversation.
func NewCursor(after time.Time) *Cursor {
	return &Cursor{after: after, offsets: map[string]int64{}, seen: map[string]bool{}}
}

// After is the time of the latest message read.
func (c *Cursor) After() time.Time { return c.after }

// Read counts what was written since the last read to the transcript at path and to the transcripts
// of its subagents, which Claude Code keeps in <session id>/subagents beside it. It returns one part
// for each model, the session's own before its subagents'.
func (c *Cursor) Read(path string) ([]Part, error) {
	sums := map[Part]*Tokens{}
	var order []Part
	latest := c.after
	add := func(file string, subagent bool) error {
		return c.readFile(file, &latest, func(model string, t Tokens) {
			key := Part{Model: model, Subagent: subagent}
			if sums[key] == nil {
				sums[key] = &Tokens{}
				order = append(order, key)
			}
			sums[key].Add(t)
		})
	}
	if err := add(path, false); err != nil {
		return nil, err
	}
	agents, _ := filepath.Glob(filepath.Join(strings.TrimSuffix(path, ".jsonl"), "subagents", "agent-*.jsonl"))
	for _, f := range agents {
		if err := add(f, true); err != nil {
			return nil, err
		}
	}
	c.after = latest
	slices.SortStableFunc(order, func(a, b Part) int {
		switch {
		case a.Subagent == b.Subagent:
			return 0
		case a.Subagent:
			return 1
		}
		return -1
	})
	out := make([]Part, len(order))
	for i, key := range order {
		out[i] = Part{Model: key.Model, Subagent: key.Subagent, Tokens: *sums[key]}
	}
	return out, nil
}

// line is the part of a transcript line that tells what an API response used.
type line struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   struct {
		ID    string    `json:"id"`
		Model string    `json:"model"`
		Usage *apiUsage `json:"usage"`
	} `json:"message"`
}

type apiUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheCreation            struct {
		Ephemeral5m int64 `json:"ephemeral_5m_input_tokens"`
		Ephemeral1h int64 `json:"ephemeral_1h_input_tokens"`
	} `json:"cache_creation"`
	OutputTokensDetails struct {
		ThinkingTokens int64 `json:"thinking_tokens"`
	} `json:"output_tokens_details"`
	Speed string `json:"speed"`
}

// readFile reads the whole lines added to file since the last read. A line still being written is
// left for the next read. A file that is not there yet has nothing to count. latest moves to the
// time of the latest line read.
func (c *Cursor) readFile(file string, latest *time.Time, each func(model string, t Tokens)) error {
	f, err := os.Open(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer f.Close()
	start, known := c.offsets[file]
	if info, err := f.Stat(); err == nil && info.Size() < start {
		start = 0 // the file was replaced
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReaderSize(f, 1<<16)
	for {
		b, err := r.ReadBytes('\n')
		if err != nil {
			break
		}
		start += int64(len(b))
		c.count(b, known, latest, each)
	}
	c.offsets[file] = start
	return nil
}

// count counts the line's message, once. In a file read before, every new line is new; in a file
// read for the first time, only the lines after c.after are.
func (c *Cursor) count(b []byte, known bool, latest *time.Time, each func(model string, t Tokens)) {
	if !bytes.Contains(b, []byte(`"usage"`)) {
		return
	}
	var l line
	if json.Unmarshal(b, &l) != nil || l.Type != "assistant" || l.Message.Usage == nil || l.Message.ID == "" {
		return
	}
	// The event log keeps the time of the latest message in milliseconds.
	if !known && !l.Timestamp.Truncate(time.Millisecond).After(c.after) {
		return
	}
	if l.Timestamp.After(*latest) {
		*latest = l.Timestamp
	}
	if c.seen[l.Message.ID] {
		return
	}
	c.seen[l.Message.ID] = true
	u := *l.Message.Usage
	t := Tokens{
		Input: u.InputTokens, Output: u.OutputTokens, CacheRead: u.CacheReadInputTokens, CacheWrite: u.CacheCreationInputTokens,
		Thinking: u.OutputTokensDetails.ThinkingTokens, Cost: cost(l.Message.Model, u),
	}
	if t.Total() == 0 {
		return
	}
	each(l.Message.Model, t)
}
