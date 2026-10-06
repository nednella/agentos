package notes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
)

var issueURL = regexp.MustCompile(`https://[^\s/]+/[^\s/]+/[^\s/]+/issues/\d+`)

// Projects knows the current project.
type Projects interface{ Current() project.Project }

// Sessions starts sessions.
type Sessions interface {
	Create(title, text string, send bool, issue int) (sessions.Session, error)
}

// Issues knows a folder's GitHub repo and can reload a project's issues.
type Issues interface {
	Repo(ctx context.Context, dir string) string
	Reload(ctx context.Context, proj project.Project)
}

// Service is bound to the front end.
type Service struct {
	notes    *Notes
	project  Projects
	sessions Sessions
	issues   Issues
	run      run.Runner
	emit     func(event string, payload any)
	ctx      func() context.Context

	filingMu sync.Mutex
	filing   map[string]bool // "<project key>/<note id>" of notes being filed as issues
}

func NewService(n *Notes, p Projects, s Sessions, i Issues, runner run.Runner, emit func(string, any), ctx func() context.Context) *Service {
	return &Service{notes: n, project: p, sessions: s, issues: i, run: runner, emit: emit, ctx: ctx, filing: map[string]bool{}}
}

func (s *Service) emitNotes() {
	key := s.project.Current().Key()
	notes, err := s.notes.List(key)
	if err != nil {
		log.Printf("agentos: %v", err)
		return
	}
	s.emit("notes", scoped.Of(key, notes))
}

// AddNote adds a note. Its text may be empty: a note made to hold pictures is created before they are attached.
func (s *Service) AddNote(text string) (Note, error) {
	n, err := s.notes.add(s.project.Current().Key(), text)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) UpdateNote(id, text string) (Note, error) {
	n, err := s.notes.Update(s.project.Current().Key(), id, text)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) SetNotePinned(id string, pinned bool) (Note, error) {
	n, err := s.notes.SetPinned(s.project.Current().Key(), id, pinned)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) SetNoteArchived(id string, archived bool) (Note, error) {
	n, err := s.notes.SetArchived(s.project.Current().Key(), id, archived)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) AddNoteImage(id, data, mime string) (Note, error) {
	n, err := s.notes.AddImage(s.project.Current().Key(), id, data, mime)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) RemoveNoteImage(id, url string) (Note, error) {
	n, err := s.notes.RemoveImage(s.project.Current().Key(), id, url)
	if err == nil {
		s.emitNotes()
	}
	return n, err
}

func (s *Service) DeleteNote(id string) error {
	err := s.notes.Delete(s.project.Current().Key(), id)
	if err == nil {
		s.emitNotes()
	}
	return err
}

// NoteToIssue files the note on GitHub, its first line as the title, the rest as the body and
// its pictures attached, then deletes the note. A second call for a note that is being filed
// fails, so one note never becomes two issues.
func (s *Service) NoteToIssue(id string) error {
	cur := s.project.Current()
	filing := cur.Key() + "/" + id
	s.filingMu.Lock()
	busy := s.filing[filing]
	s.filing[filing] = true
	s.filingMu.Unlock()
	if busy {
		return errors.New("this note is already being filed as an issue")
	}
	defer func() {
		s.filingMu.Lock()
		delete(s.filing, filing)
		s.filingMu.Unlock()
	}()

	n, err := s.notes.Get(cur.Key(), id)
	if err != nil {
		return err
	}
	if n.title() == "" {
		return errors.New("the note has no text to use as the issue title")
	}
	if s.issues.Repo(s.ctx(), cur.Dir) == "" {
		return errors.New("this project has no GitHub repo")
	}
	ctx, cancel := context.WithTimeout(s.ctx(), run.GHTimeout)
	defer cancel()
	args := []string{"issue", "create", "--title", n.title(), "--body", issueBody(n), "--assignee", "@me"}
	for i, image := range n.Images {
		args = append(args, "--attach", fmt.Sprintf("%s#Picture %d", s.notes.MediaFile(image), i+1))
	}
	out, err := s.run(ctx, cur.Dir, "gh", args...)
	url := issueURL.FindString(string(out))
	if err != nil && url == "" {
		return fmt.Errorf("filing the issue: %w", err)
	}
	if url == "" {
		return fmt.Errorf("gh did not print an issue address: %q", strings.TrimSpace(string(out)))
	}
	if err != nil {
		// gh filed the issue but some pictures did not upload; the note stays so they are not lost.
		s.issues.Reload(s.ctx(), cur)
		return fmt.Errorf("filed %s but some pictures did not upload: %w", url, err)
	}
	if err := s.notes.Delete(cur.Key(), id); err != nil {
		return fmt.Errorf("filed %s but could not delete the note: %w", url, err)
	}
	s.emitNotes()
	s.issues.Reload(s.ctx(), cur)
	return nil
}

// NoteToSession starts a session titled by the note, with the project's note command typed in.
func (s *Service) NoteToSession(id string) (sessions.Session, error) {
	cur := s.project.Current()
	n, err := s.notes.Get(cur.Key(), id)
	if err != nil {
		return sessions.Session{}, err
	}
	return s.sessions.Create(n.title(), cur.NoteCommand(n.Text), cur.SendsPrompt(), 0)
}

// issueBody is the issue template filled with the note's text; gh appends the pictures.
func issueBody(n Note) string { return "## Description\n\n" + n.Text }
