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

// Commands answers agentos digest add, which the digest run uses to hand in its findings.
type Commands struct{ m *Manager }

func NewCommands(m *Manager) *Commands { return &Commands{m: m} }

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
