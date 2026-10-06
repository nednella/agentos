package issues

import "context"

import "github.com/nednella/agentos/desktop/sessions"

// Service is bound to the front end.
type Service struct {
	issues *Issues
	ctx    func() context.Context
}

// NewService binds the issues; ctx gives the app's context at call time.
func NewService(i *Issues, ctx func() context.Context) *Service { return &Service{issues: i, ctx: ctx} }

// Issues is the current project's open issues, cached unless refresh.
func (s *Service) Issues(refresh bool) ([]Issue, error) {
	list, err := s.issues.List(s.ctx(), s.issues.sessions.Current(), refresh)
	if err != nil {
		return nil, err
	}
	return s.issues.WithSessions(list), nil
}

// StartIssue opens a session for the issue and types the command of the action with that name, the section's
// default for ""; an issue that has a live session gets it back.
func (s *Service) StartIssue(number int, action string) (sessions.Session, error) {
	return s.issues.Start(s.ctx(), number, action)
}

// IssueDetail is the issue's body and comments as GitHub renders them.
func (s *Service) IssueDetail(number int) (IssueDetail, error) {
	return s.issues.Detail(s.ctx(), s.issues.sessions.Current().Dir, number)
}
