package awake

// Service is bound to the front end.
type Service struct{ a *Awake }

func NewService(a *Awake) *Service { return &Service{a: a} }

// Awake says whether the app is holding off idle sleep.
func (s *Service) Awake() bool { return s.a.Held() }
