package terminal

// Service is bound to the front end.
type Service struct{ terms *Terms }

func NewService(t *Terms) *Service { return &Service{terms: t} }

// TermOpen attaches a terminal stream to the session; opening an open one redraws everything.
func (s *Service) TermOpen(id string, cols, rows int) error { return s.terms.Open(id, cols, rows) }

// TermWrite sends raw input bytes (xterm.js onData) to the session.
func (s *Service) TermWrite(id, data string) error { return s.terms.Write(id, data) }

func (s *Service) TermResize(id string, cols, rows int) error { return s.terms.Resize(id, cols, rows) }

// TermClose detaches the stream; the agent keeps running.
func (s *Service) TermClose(id string) { s.terms.Close(id) }
