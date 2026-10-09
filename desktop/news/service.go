package news

// Service is bound to the front end.
type Service struct{ m *Manager }

func NewService(m *Manager) *Service { return &Service{m: m} }

// News is the stored newsletter issues.
func (s *Service) News() News { return s.m.view() }

// RefreshNews starts a fetch in the background; progress arrives as news events.
func (s *Service) RefreshNews() error { return s.m.start() }
