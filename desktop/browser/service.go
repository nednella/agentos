package browser

import (
	"context"

	"github.com/nednella/agentos/desktop/evidence"
)

// Service is bound to the front end.
type Service struct {
	browsers *Browsers
	store    *evidence.Store
	changes  evidence.Changes
	ctx      func() context.Context
}

func NewService(b *Browsers, store *evidence.Store, changes evidence.Changes, ctx func() context.Context) *Service {
	return &Service{browsers: b, store: store, changes: changes, ctx: ctx}
}

func (s *Service) browserCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(s.ctx(), browserStartLimit+callLimit)
}

func (s *Service) BrowserOpen(id, url string) (BrowserState, error) {
	ctx, cancel := s.browserCtx()
	defer cancel()
	return s.browsers.Open(ctx, id, url)
}

func (s *Service) BrowserGoto(id, url string) error {
	ctx, cancel := s.browserCtx()
	defer cancel()
	return s.browsers.Goto(ctx, id, url)
}

func (s *Service) BrowserNav(id, action string) error {
	ctx, cancel := s.browserCtx()
	defer cancel()
	return s.browsers.Nav(ctx, id, action)
}

func (s *Service) BrowserInput(id string, in BrowserInput) error {
	ctx, cancel := context.WithTimeout(s.ctx(), callLimit)
	defer cancel()
	return s.browsers.Input(ctx, id, in)
}

func (s *Service) BrowserResize(id string, width, height int) error {
	ctx, cancel := context.WithTimeout(s.ctx(), callLimit)
	defer cancel()
	return s.browsers.Resize(ctx, id, width, height)
}

func (s *Service) BrowserView(id string, visible bool) error {
	ctx, cancel := context.WithTimeout(s.ctx(), callLimit)
	defer cancel()
	return s.browsers.View(ctx, id, visible)
}

func (s *Service) BrowserShow(id string) error {
	ctx, cancel := context.WithTimeout(s.ctx(), callLimit)
	defer cancel()
	return s.browsers.Show(ctx, id)
}

func (s *Service) BrowserState(id string) BrowserState { return s.browsers.State(id) }

// BrowserScreenshot files the user's own capture of the tab as evidence.
func (s *Service) BrowserScreenshot(id, caption string) (evidence.Evidence, error) {
	ctx, cancel := s.browserCtx()
	defer cancel()
	png, err := s.browsers.Screenshot(ctx, id, false)
	if err != nil {
		return evidence.Evidence{}, err
	}
	item, err := s.store.AddImage(id, png, caption, "user")
	if err == nil {
		s.changes.Changed(s.store, id, false)
	}
	return item, err
}

func (s *Service) BrowserClose(id string) {
	ctx, cancel := context.WithTimeout(s.ctx(), callLimit)
	defer cancel()
	s.browsers.Close(ctx, id)
}
