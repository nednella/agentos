package evidence

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/nednella/agentos/desktop/internal/media"
	ctl "github.com/nednella/agentos/internal/control"
)

const maxTextCard = 64 << 10

// Asker checks which session a command comes from.
type Asker interface {
	AskerSession(req ctl.Request) (string, error)
}

// Changes is told when a session's evidence changed.
type Changes struct {
	Emit  func(event string, payload any)
	Touch func() // the session list shows the evidence count
}

// Service is bound to the front end.
type Service struct {
	store   *Store
	changes Changes
}

func NewService(s *Store, c Changes) *Service { return &Service{store: s, changes: c} }

// Evidence is the session's evidence, oldest first.
func (s *Service) Evidence(id string) []Evidence { return s.store.List(id) }

// DeleteEvidence removes one item of a session's evidence.
func (s *Service) DeleteEvidence(id, evidenceID string) error {
	err := s.store.Delete(id, evidenceID)
	if err == nil {
		s.changes.Changed(s.store, id, false)
	}
	return err
}

// Changed tells the front end that a session's evidence changed; an agent's new evidence also raises the session.
func (c Changes) Changed(store *Store, id string, byAgent bool) {
	c.Emit("evidence", map[string]any{"id": id, "items": store.List(id)})
	if byAgent {
		c.Emit("attention", map[string]string{"id": id, "state": "evidence"})
	}
	c.Touch()
}

// Commands answers agentos show.
type Commands struct {
	store   *Store
	asker   Asker
	changes Changes
}

func NewCommands(s *Store, a Asker, c Changes) *Commands {
	return &Commands{store: s, asker: a, changes: c}
}

// Show files a picture, a text file or some words as the session's evidence.
func (c *Commands) Show(_ context.Context, req ctl.Request) (string, error) {
	id, err := c.asker.AskerSession(req)
	if err != nil {
		return "", err
	}
	caption := req.Opts["caption"]
	if text := req.Opts["text"]; text != "" {
		if _, err := c.store.AddText(id, text, caption, "agent"); err != nil {
			return "", err
		}
		c.changes.Changed(c.store, id, true)
		return "filed as evidence", nil
	}
	if len(req.Args) != 1 {
		return "", errors.New("usage: agentos show <file> [--caption <text>] or agentos show --text <words>")
	}
	path := req.Args[0]
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a file", path)
	}
	if info.Size() > media.MaxImageSize {
		return "", fmt.Errorf("%s is over 10 MB", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	if _, err = c.store.AddImage(id, data, caption, "agent"); err != nil {
		if len(data) > maxTextCard || !utf8.Valid(data) {
			return "", fmt.Errorf("%s is neither a picture nor a text file under 64 KB", path)
		}
		if caption == "" {
			caption = filepath.Base(path)
		}
		if _, err = c.store.AddText(id, string(data), caption, "agent"); err != nil {
			return "", err
		}
	}
	c.changes.Changed(c.store, id, true)
	return "filed as evidence", nil
}
