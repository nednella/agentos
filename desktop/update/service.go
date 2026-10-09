package update

import (
	"context"

	"github.com/nednella/agentos"
)

// Service is bound to the front end.
type Service struct {
	u   *Updater
	ctx func() context.Context
}

func NewService(u *Updater, ctx func() context.Context) *Service { return &Service{u: u, ctx: ctx} }

// Update installs the newer release and relaunches the app.
func (s *Service) Update() error { return s.u.Apply(s.ctx()) }

// PatchNotes is the releases since the version the app last ran; it returns them once.
func (s *Service) PatchNotes() ([]Release, error) { return s.u.PatchNotes() }

// Changelog is every release the app shipped with, newest first.
func (s *Service) Changelog() []Release { return ParseChangelog(agentos.Changelog) }
