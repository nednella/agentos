package notes

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
)

var issueURL = regexp.MustCompile(`https://[^\s/]+/[^\s/]+/[^\s/]+/issues/(\d+)`)

// Projects knows the current project.
type Projects interface{ Current() project.Project }

// Sessions starts sessions.
type Sessions interface {
	Create(title, prefill string, issue int) (sessions.Session, error)
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
}

func NewService(n *Notes, p Projects, s Sessions, i Issues, runner run.Runner, emit func(string, any), ctx func() context.Context) *Service {
	return &Service{notes: n, project: p, sessions: s, issues: i, run: runner, emit: emit, ctx: ctx}
}

func (s *Service) emitNotes() {
	notes, err := s.notes.List(s.project.Current().Key())
	if err != nil {
		log.Printf("agentos: %v", err)
		return
	}
	s.emit("notes", notes)
}

func (s *Service) AddNote(text string) (Note, error) {
	n, err := s.notes.Add(s.project.Current().Key(), text)
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

// NoteToIssue files the note on GitHub: its first line is the title, the rest the body.
func (s *Service) NoteToIssue(id string) (Note, error) {
	cur := s.project.Current()
	n, err := s.notes.Get(cur.Key(), id)
	if err != nil {
		return Note{}, err
	}
	if n.Issue > 0 {
		return Note{}, fmt.Errorf("this note is already issue #%d", n.Issue)
	}
	if s.issues.Repo(s.ctx(), cur.Dir) == "" {
		return Note{}, errors.New("this project has no GitHub repo")
	}
	ctx, cancel := context.WithTimeout(s.ctx(), run.GHTimeout)
	defer cancel()
	out, err := s.run(ctx, cur.Dir, "gh", "issue", "create", "--title", n.title(), "--body", issueBody(n), "--assignee", "@me")
	if err != nil {
		return Note{}, fmt.Errorf("filing the issue: %w", err)
	}
	m := issueURL.FindStringSubmatch(string(out))
	if m == nil {
		return Note{}, fmt.Errorf("gh did not print an issue address: %q", strings.TrimSpace(string(out)))
	}
	number, _ := strconv.Atoi(m[1])
	n, err = s.notes.SetIssue(cur.Key(), id, number, m[0])
	if err != nil {
		return Note{}, err
	}
	s.emitNotes()
	s.issues.Reload(s.ctx(), cur)
	return n, nil
}

// NoteToSession starts a session titled by the note, with the project's note command typed in.
func (s *Service) NoteToSession(id string) (sessions.Session, error) {
	cur := s.project.Current()
	n, err := s.notes.Get(cur.Key(), id)
	if err != nil {
		return sessions.Session{}, err
	}
	return s.sessions.Create(n.title(), cur.NoteCommand(n.Text), 0)
}

// issueBody is the issue template filled with the note's text. gh cannot upload
// pictures, so a note with pictures says they stay in agentos.
func issueBody(n Note) string {
	body := "## Description\n\n" + n.Text
	switch len(n.Images) {
	case 0:
	case 1:
		body += "\n\n(1 screenshot is attached to the note in agentos.)"
	default:
		body += fmt.Sprintf("\n\n(%d screenshots are attached to the note in agentos.)", len(n.Images))
	}
	return body
}
