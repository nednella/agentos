package stats

import (
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
}

func NewService(waits *Waits, p Project, books *Books) *Service {
	return &Service{waits: waits, project: p, books: books}
}

// Stats is the current project's interruption tally over the last days days.
func (s *Service) Stats(days int) (Stats, error) {
	return s.waits.Stats(s.project.Current().Key(), days, time.Now())
}

// Ledger is what the agents did in every project over the last days days; 0 is all time.
func (s *Service) Ledger(days int) (Ledger, error) {
	return s.books.Ledger(days, time.Now())
}
