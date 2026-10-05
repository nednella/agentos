package sessions

import "context"

// Service is bound to the front end.
type Service struct {
	s   *Sessions
	ctx func() context.Context
}

// NewService binds the sessions; ctx gives the app's context at call time.
func NewService(s *Sessions, ctx func() context.Context) *Service { return &Service{s: s, ctx: ctx} }

// NewSession starts the agent in the current project; a non-empty prefill is typed in, not sent, once the agent is ready.
func (v *Service) NewSession(title, prefill string) (Session, error) {
	return v.s.Create(title, prefill, 0)
}

// KillSession stops the agent; its row stays as ended.
func (v *Service) KillSession(id string) error { return v.s.Kill(id) }

// DismissSession removes the row of an ended session.
func (v *Service) DismissSession(id string) error { return v.s.Dismiss(id) }

func (v *Service) RenameSession(id, title string) error { return v.s.Rename(id, title) }

// TypeInto types text into the session's prompt, not sent.
func (v *Service) TypeInto(id, text string) error { return v.s.TypeInto(id, text) }

// RefreshPRs polls the pull requests of the issue sessions now.
func (v *Service) RefreshPRs() { v.s.life.Poll(v.ctx()) }

// AckPR says the user has seen the session's pull request comments and failing checks.
func (v *Service) AckPR(id string) error { return v.s.life.Ack(id) }

// Cleanup cleans up after a session now; without force the safety checks apply.
func (v *Service) Cleanup(id string, force bool) error { return v.s.life.Cleanup(v.ctx(), id, force) }

// Cleanups is the current project's clean-up log, newest first.
func (v *Service) Cleanups() []Cleanup { return v.s.life.Cleanups(v.s.Current().Key()) }
