package stats

import (
	"context"
	"time"

	"github.com/nednella/agentos/internal/project"
)

// Project says which project's tally to read.
type Project interface{ Current() project.Project }

// Service is bound to the front end.
type Service struct {
	waits   *Waits
	project Project
	books   *Books
	ctx     func() context.Context
}

func NewService(waits *Waits, p Project, books *Books, ctx func() context.Context) *Service {
	return &Service{waits: waits, project: p, books: books, ctx: ctx}
}

// Stats is the current project's interruption tally over the last days days.
func (s *Service) Stats(days int) (Stats, error) {
	return s.waits.Stats(s.project.Current().Key(), days, time.Now())
}

// Ledger is what the agents did in every project over the last days days; 0 is all time.
// It asks GitHub, so the front end loads it on demand.
func (s *Service) Ledger(days int) (Ledger, error) {
	return s.books.Ledger(s.ctx(), days, time.Now())
}
