// Package notes keeps a project's notes and their pictures.
package notes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
)

// Note is a line of thought kept for a project, which can become an issue or a session.
type Note struct {
	ID        string   `json:"id"`
	Text      string   `json:"text"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
	Pinned    bool     `json:"pinned"`
	Archived  bool     `json:"archived"`
	Images    []string `json:"images"` // URLs under /media/
}

// title is the first line of the note.
func (n Note) title() string {
	first, _, _ := strings.Cut(strings.TrimSpace(n.Text), "\n")
	return strings.TrimSpace(first)
}

func (n Note) body() string {
	_, rest, _ := strings.Cut(strings.TrimSpace(n.Text), "\n")
	return strings.TrimSpace(rest)
}

// New keeps the notes under dir.
func New(dir string) *Notes { return &Notes{dir: dir} }

// Notes keeps each project's notes in one JSON file under dir.
type Notes struct {
	dir string

	mu sync.Mutex
}

func (s *Notes) path(key string) string { return filepath.Join(s.dir, key, "notes.json") }

// load needs mu.
func (s *Notes) load(key string) ([]Note, error) {
	data, err := os.ReadFile(s.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading notes: %w", err)
	}
	var notes []Note
	if err := json.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("decoding notes: %w", err)
	}
	for i := range notes {
		if notes[i].Images == nil {
			notes[i].Images = []string{}
		}
	}
	return notes, nil
}

// save needs mu.
func (s *Notes) save(key string, notes []Note) error {
	data, err := json.MarshalIndent(notes, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding notes: %w", err)
	}
	return atomicfile.Write(s.path(key), data, 0o600)
}

// List returns pinned notes first, then the rest newest first, and archived notes last.
func (s *Notes) List(key string) ([]Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load(key)
	if err != nil {
		return nil, err
	}
	return sorted(notes), nil
}

func sorted(notes []Note) []Note {
	out := slices.Clone(notes)
	if out == nil {
		out = []Note{}
	}
	slices.SortStableFunc(out, func(a, b Note) int {
		switch {
		case a.Archived != b.Archived:
			return boolRank(a.Archived) - boolRank(b.Archived)
		case a.Pinned != b.Pinned:
			return boolRank(b.Pinned) - boolRank(a.Pinned)
		}
		return int(b.CreatedAt - a.CreatedAt)
	})
	return out
}

// edit applies change to one note and saves.
func (s *Notes) edit(key, id string, change func(*Note) error) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load(key)
	if err != nil {
		return Note{}, err
	}
	i := slices.IndexFunc(notes, func(n Note) bool { return n.ID == id })
	if i < 0 {
		return Note{}, errors.New("note not found")
	}
	if err := change(&notes[i]); err != nil {
		return Note{}, err
	}
	notes[i].UpdatedAt = time.Now().UnixMilli()
	return notes[i], s.save(key, notes)
}

func (s *Notes) Get(key, id string) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load(key)
	if err != nil {
		return Note{}, err
	}
	for _, n := range notes {
		if n.ID == id {
			return n, nil
		}
	}
	return Note{}, errors.New("note not found")
}

// Add keeps a note that has text.
func (s *Notes) Add(key, text string) (Note, error) {
	if strings.TrimSpace(text) == "" {
		return Note{}, errors.New("the note is empty")
	}
	return s.add(key, text)
}

func (s *Notes) add(key, text string) (Note, error) {
	text = strings.TrimSpace(text)
	id := make([]byte, 6)
	if _, err := rand.Read(id); err != nil {
		return Note{}, fmt.Errorf("making a note id: %w", err)
	}
	now := time.Now().UnixMilli()
	n := Note{ID: hex.EncodeToString(id), Text: text, CreatedAt: now, UpdatedAt: now, Images: []string{}}
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load(key)
	if err != nil {
		return Note{}, err
	}
	return n, s.save(key, append(notes, n))
}

func (s *Notes) Update(key, id, text string) (Note, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Note{}, errors.New("the note is empty")
	}
	return s.edit(key, id, func(n *Note) error { n.Text = text; return nil })
}

func (s *Notes) SetPinned(key, id string, pinned bool) (Note, error) {
	return s.edit(key, id, func(n *Note) error { n.Pinned = pinned; return nil })
}

func (s *Notes) SetArchived(key, id string, archived bool) (Note, error) {
	return s.edit(key, id, func(n *Note) error { n.Archived = archived; return nil })
}

func (s *Notes) Delete(key, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	notes, err := s.load(key)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(notes, func(n Note) bool { return n.ID == id })
	if i < 0 {
		return errors.New("note not found")
	}
	images := notes[i].Images
	if err := s.save(key, slices.Delete(notes, i, i+1)); err != nil {
		return err
	}
	for _, url := range images {
		s.removeFile(url)
	}
	return nil
}

func boolRank(b bool) int {
	if b {
		return 1
	}
	return 0
}
