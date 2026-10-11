package stats

import "time"

// Service is bound to the front end.
type Service struct {
	waits    *Waits
	projects Registry
	books    *Books
}

func NewService(waits *Waits, p Registry, books *Books) *Service {
	return &Service{waits: waits, projects: p, books: books}
}

// Stats is the interruption tally over the last days days of the project whose key is given, or of
// every project for "".
func (s *Service) Stats(days int, key string) (Stats, error) {
	keys := []string{key}
	if key == "" {
		keys = nil
		for _, p := range s.projects.List() {
			keys = append(keys, p.Key())
		}
	}
	return s.waits.Stats(keys, days, time.Now())
}

// Ledger is what the agents did over the last days days, 0 for all time, in the project whose key is
// given, or in every project for "".
func (s *Service) Ledger(days int, key string) (Ledger, error) {
	return s.books.Ledger(days, key, time.Now())
}
