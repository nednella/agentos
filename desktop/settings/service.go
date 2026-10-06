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
	Cleanup(projectKey string) project.Cleanup
	SetCleanup(projectKey, event, mode string) error
}

// Current names the project the panel edits.
type Current interface {
	Current() project.Project
}

// Settings is what the settings panel shows.
type Settings struct {
	Theme   string          `json:"theme"`   // system, light or dark
	Cleanup project.Cleanup `json:"cleanup"` // the current project's: auto or manual, for merge and close
}

// Service is bound to the front end.
type Service struct {
	store   Store
	project Current
}

func NewService(store Store, project Current) *Service {
	return &Service{store: store, project: project}
}

// Settings are the current settings.
func (s *Service) Settings() Settings {
	theme := s.store.Theme()
	if theme == "" {
		theme = system
	}
	return Settings{Theme: theme, Cleanup: s.store.Cleanup(s.project.Current().Key())}
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

// SetCleanup saves how the current project cleans up after an event, "merge" or "close": "auto" or "manual".
func (s *Service) SetCleanup(event, mode string) (Settings, error) {
	if err := s.store.SetCleanup(s.project.Current().Key(), event, mode); err != nil {
		return Settings{}, err
	}
	return s.Settings(), nil
}
