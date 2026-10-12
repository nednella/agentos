package sessions

import (
	"cmp"
	"fmt"
	"maps"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
	"github.com/nednella/agentos/internal/usage"
)

// reader reads one session's transcripts, one read at a time.
type reader struct {
	mu     sync.Mutex
	cursor *usage.Cursor // nil until the event log told what was counted before
}

// countUsage records what the session's agent used since the last count, one event for each model,
// and adds it to the session's total. The first count of a run of the app starts after the latest
// usage the event log holds for the session, so a restart counts nothing twice.
func (s *Sessions) countUsage(id, transcript string) {
	s.mu.Lock()
	in, ok := s.infoOf(id)
	r := s.readers[id]
	if r == nil {
		r = &reader{}
		s.readers[id] = r
	}
	s.mu.Unlock()
	if !ok {
		return
	}
	kind := activity.Usage
	if in.Chat {
		kind = activity.ChatUsage
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var before usage.Tokens
	if r.cursor == nil {
		after := in.Created
		s.activity.Since(session.ProjectKey(id), in.Created.In(time.Local).Format(time.DateOnly), func(e activity.Event) {
			if e.Kind != kind || e.Session != id {
				return
			}
			before.Add(e.Tokens)
			if at := time.UnixMilli(e.At); at.After(after) {
				after = at
			}
		})
		r.cursor = usage.NewCursor(after)
	}
	parts, err := r.cursor.Read(transcript)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: reading the usage of %s: %v\n", id, err)
	}
	if len(parts) == 0 && before == (usage.Tokens{}) {
		return
	}
	issue, _ := strconv.Atoi(in.Issue)
	s.mu.Lock()
	defer s.mu.Unlock()
	total := s.spent[id]
	total.Add(before)
	for _, p := range parts {
		e := activity.Event{At: r.cursor.After().UnixMilli(), Kind: kind, Issue: issue, Model: p.Model, Effort: in.Effort, Title: cmp.Or(in.Title, defaultTitle), Tokens: p.Tokens}
		if p.Subagent {
			e.Label = activity.Subagent
		}
		s.record(id, e)
		total.Add(p.Tokens)
	}
	if _, kept := s.readers[id]; kept { // a dismissed session has no row to show it
		s.spent[id] = total
		s.sync()
	}
}

// countUnread counts the usage of each live session whose transcript holds lines not read yet: the
// cost moves during a long turn, and takes in the last reply of a turn, which Claude Code writes
// just after the turn's Stop.
func (s *Sessions) countUnread() {
	s.mu.Lock()
	transcripts := map[string]string{}
	for _, in := range s.info {
		id := in.Name.String()
		if t := s.records[id].Transcript; t != "" {
			transcripts[id] = t
		}
	}
	readers := maps.Clone(s.readers)
	s.mu.Unlock()
	for id, t := range transcripts {
		if r := readers[id]; r == nil || r.unread(t) {
			s.countUsage(id, t)
		}
	}
}

func (r *reader) unread(transcript string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cursor == nil || r.cursor.Unread(transcript)
}

// infoOf is what is known of the session, live or ended. It needs mu.
func (s *Sessions) infoOf(id string) (term.Info, bool) {
	for _, in := range s.info {
		if in.Name.String() == id {
			return in, true
		}
	}
	if e, ok := s.ended[id]; ok {
		return e.info, true
	}
	in, ok := s.known[id]
	return in, ok
}
