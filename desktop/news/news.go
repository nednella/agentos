// Package news keeps the latest issues of the TLDR Dev newsletter, fetched in the background.
package news

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
)

const (
	issuesKept     = 7
	issuesPerFetch = 3
	staleAfter     = 3 * time.Hour
)

// Item is one story of an issue.
type Item struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	ReadTime string `json:"readTime"`
	Summary  string `json:"summary"`
	URL      string `json:"url"`
}

// Issue is one daily issue of the newsletter.
type Issue struct {
	Date      string `json:"date"` // YYYY-MM-DD
	Title     string `json:"title"`
	URL       string `json:"url"`
	FetchedAt int64  `json:"fetchedAt"` // unix ms; 0 for issues stored before it was kept
	Items     []Item `json:"items"`
}

// News is what the front end sees.
type News struct {
	Running     bool    `json:"running"`
	LastFetchAt int64   `json:"lastFetchAt"`
	Error       string  `json:"error"`
	Issues      []Issue `json:"issues"`
}

type newsFile struct {
	LastFetchAt int64   `json:"lastFetchAt"`
	Issues      []Issue `json:"issues"`
}

// Store keeps the news in news.json of the data dir.
type Store struct {
	path string

	mu      sync.Mutex
	running bool
	failure string
}

// NewStore keeps the news under dataDir.
func NewStore(dataDir string) *Store { return &Store{path: filepath.Join(dataDir, "news.json")} }

// read needs mu.
func (s *Store) read() newsFile {
	var f newsFile
	if data, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	return f
}

// write needs mu.
func (s *Store) write(f newsFile) error {
	data, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("encoding the news: %w", err)
	}
	return atomicfile.Write(s.path, data, 0o600)
}

// View is the news.
func (s *Store) View() News {
	s.mu.Lock()
	defer s.mu.Unlock()
	f := s.read()
	v := News{Running: s.running, LastFetchAt: f.LastFetchAt, Error: s.failure, Issues: f.Issues}
	if v.Issues == nil {
		v.Issues = []Issue{}
	}
	return v
}

// Due says whether an automatic fetch should start now.
func (s *Store) Due(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.running && now.Sub(time.UnixMilli(s.read().LastFetchAt)) > staleAfter
}

// Begin marks a fetch as started.
func (s *Store) Begin() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return errors.New("the news are already being fetched")
	}
	s.running, s.failure = true, ""
	return nil
}

// Finish ends a fetch. fresh are the issues it got, which are kept even when failure is set;
// a failed fetch keeps the old news and records why.
func (s *Store) Finish(now time.Time, fresh []Issue, failure string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running, s.failure = false, failure
	f := s.read()
	if len(fresh) > 0 {
		f.Issues = append(fresh, f.Issues...)
		slices.SortStableFunc(f.Issues, func(a, b Issue) int { return strings.Compare(b.Date, a.Date) })
		f.Issues = f.Issues[:min(len(f.Issues), issuesKept)]
	}
	if failure == "" {
		f.LastFetchAt = now.UnixMilli()
	}
	if err := s.write(f); err != nil {
		s.failure = err.Error()
	}
}

// Has says whether the issue of the day is stored.
func (s *Store) Has(date string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.ContainsFunc(s.read().Issues, func(i Issue) bool { return i.Date == date })
}
