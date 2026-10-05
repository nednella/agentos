package digest

import (
	"context"
	"errors"

	"github.com/nednella/agentos/desktop/notes"
	ctl "github.com/nednella/agentos/internal/control"
)

// Service is bound to the front end.
type Service struct{ m *Manager }

func NewService(m *Manager) *Service { return &Service{m: m} }

// Digest is the current project's digest.
func (s *Service) Digest() Digest { return s.m.view() }

// RunDigest starts a run in the background; progress arrives as digest events.
func (s *Service) RunDigest() error { return s.m.runCurrent() }

// DigestToNote saves the item as a note and marks it saved.
func (s *Service) DigestToNote(itemID string) (notes.Note, error) { return s.m.toNote(itemID) }

// DismissDigestItem removes an item.
func (s *Service) DismissDigestItem(itemID string) error { return s.m.dismiss(itemID) }

// Scene shows things on the screen of the project a command comes from.
type Scene interface {
	Enter(req ctl.Request)
	UI(req ctl.Request, name string, args ...string) string
}

// Commands answers agentos digest, and agentos digest add, which the digest run uses to hand in its findings.
type Commands struct {
	m     *Manager
	scene Scene
}

func NewCommands(m *Manager, scene Scene) *Commands { return &Commands{m: m, scene: scene} }

// Digest shows the digest, or with --run starts a run for the project.
func (c *Commands) Digest(_ context.Context, req ctl.Request) (string, error) {
	if req.Opts["run"] == "" {
		return c.scene.UI(req, "digest"), nil
	}
	c.scene.Enter(req)
	if err := c.m.runCurrent(); err != nil {
		return "", err
	}
	return "digest started", nil
}

// Add files an item found by the run in progress.
func (c *Commands) Add(_ context.Context, req ctl.Request) (string, error) {
	key := req.Project
	if key == "" {
		return "", errors.New("this command is for the digest run")
	}
	added, err := c.m.digests.Add(key, DigestItem{Title: req.Opts["title"], Why: req.Opts["why"], URL: req.Opts["url"], Source: req.Opts["source"]})
	if err != nil {
		return "", err
	}
	c.m.emitDigest(key)
	if !added {
		return "already in the digest", nil
	}
	return "added", nil
}
