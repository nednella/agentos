// Package settings holds the app's settings, which the config file stores.
package settings

import (
	"fmt"
	"slices"

	"github.com/nednella/agentos/internal/project"
)

const system = "system"

// Store is where the settings live.
type Store interface {
	Theme() string
	SetTheme(theme string) error
}

// Settings is what the settings panel shows.
type Settings struct {
	Theme string `json:"theme"` // system, light or dark
}

// Service is bound to the front end.
type Service struct{ store Store }

func NewService(store Store) *Service { return &Service{store: store} }

// Settings are the current settings.
func (s *Service) Settings() Settings {
	if theme := s.store.Theme(); theme != "" {
		return Settings{Theme: theme}
	}
	return Settings{Theme: system}
}

// SetTheme saves the theme: system, light or dark.
func (s *Service) SetTheme(theme string) (Settings, error) {
	if theme != system && !slices.Contains(project.Themes, theme) {
		return Settings{}, fmt.Errorf("theme must be system, light or dark, not %q", theme)
	}
	if theme == system {
		theme = ""
	}
	if err := s.store.SetTheme(theme); err != nil {
		return Settings{}, err
	}
	return s.Settings(), nil
}
