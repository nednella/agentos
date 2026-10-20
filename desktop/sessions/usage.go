package sessions

import (
	"cmp"
	"fmt"
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
	mu      sync.Mutex
	cursor  usage.Reader // nil until the event log told what was counted before
	pending []usage.Part // read and in the session's total, but not in the event log yet
}

// countUsage adds what the session's agent used since the last count to the session's total. With
// record, it also records all that is not in the event log yet, one event for each model. The first
// count of a run of the app starts after the latest usage the event log holds for the session, so a
// restart reads again what was not recorded, and counts nothing twice.
func (s *Sessions) countUsage(id, transcript string, record bool) {
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
		r.cursor = s.agent.Usage(after)
	}
	parts, err := r.cursor.Read(transcript)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: reading the usage of %s: %v\n", id, err)
	}
	for _, p := range parts {
		r.pending = addPart(r.pending, p)
	}
	if len(parts) == 0 && before == (usage.Tokens{}) && (!record || len(r.pending) == 0) {
		return
	}
	issue, _ := strconv.Atoi(in.Issue)
	s.mu.Lock()
	defer s.mu.Unlock()
	total := s.spent[id]
	total.Add(before)
	for _, p := range parts {
		total.Add(p.Tokens)
	}
	if record {
		for _, p := range r.pending {
			e := activity.Event{At: r.cursor.After().UnixMilli(), Kind: kind, Issue: issue, Model: p.Model, Effort: in.Effort, Title: cmp.Or(in.Title, defaultTitle), Tokens: p.Tokens}
			if p.Subagent {
				e.Label = activity.Subagent
			}
			s.record(id, e)
		}
		r.pending = nil
	}
	if _, kept := s.readers[id]; kept { // a dismissed session has no row to show it
		s.spent[id] = total
		s.sync()
	}
}

// addPart adds p to the part of parts with its model and kind, or appends it.
func addPart(parts []usage.Part, p usage.Part) []usage.Part {
	for i := range parts {
		if parts[i].Model == p.Model && parts[i].Subagent == p.Subagent {
			parts[i].Add(p.Tokens)
			return parts
		}
	}
	return append(parts, p)
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
