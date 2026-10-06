package update

import "context"

// Service is bound to the front end.
type Service struct {
	u   *Updater
	ctx func() context.Context
}

func NewService(u *Updater, ctx func() context.Context) *Service { return &Service{u: u, ctx: ctx} }

// Update installs the newer release and relaunches the app.
func (s *Service) Update() error { return s.u.Apply(s.ctx()) }
