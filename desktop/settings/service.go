// Package settings holds the app's settings, which the config file stores.
package settings

import (
	"fmt"
	"slices"

	"github.com/nednella/agentos/internal/project"
)

const (
	system       = "system"
	defaultScale = 1.0
)

// Store is where the settings live.
type Store interface {
	Theme() string
	SetTheme(theme string) error
	TextScale() float64
	SetTextScale(scale float64) error
	KeepsAwake() bool
	SetKeepAwake(on bool) error
	Cleanup(projectKey string) project.Cleanup
	SetCleanup(projectKey, event, mode string) error
	PromptSend(projectKey string) string
	SetPromptSend(projectKey, mode string) error
	BrowserEnabled(projectKey string) bool
	SetBrowserEnabled(projectKey string, on bool) error
	DigestSchedule(projectKey string) string
	SetDigestSchedule(projectKey, schedule string) error
	SetDataDir(dir string) error
}

// Current names the project the panel edits.
type Current interface {
	Current() project.Project
}

// Settings is what the settings panel shows.
type Settings struct {
	Theme     string          `json:"theme"`          // system, light or dark
	TextScale float64         `json:"textScale"`      // a multiple of the default text size
	KeepAwake bool            `json:"keepAwake"`      // hold off idle sleep while a session works, for projects that do not say otherwise
	Cleanup   project.Cleanup `json:"cleanup"`        // the current project's: auto or manual, for merge and close
	Prompt    string          `json:"promptSend"`     // the current project's: auto or manual
	Browser   bool            `json:"browserEnabled"` // the current project's sessions get the browser and evidence commands
	Digest    string          `json:"digestSchedule"` // the current project's: weekly or off
	DataDir   string          `json:"dataDir"`        // the folder the running app keeps notes, evidence, stats and digests in
	DataFixed bool            `json:"dataDirFixed"`   // AGENTOS_DATA_DIR sets the folder, so it cannot change here
}

// Service is bound to the front end.
type Service struct {
	store     Store
	project   Current
	dataDir   string
	dataFixed bool
	pickDir   func() (string, error) // "" when the user cancels
	relaunch  func() error
}

func NewService(store Store, project Current, dataDir string, dataFixed bool, pickDir func() (string, error), relaunch func() error) *Service {
	return &Service{store: store, project: project, dataDir: dataDir, dataFixed: dataFixed, pickDir: pickDir, relaunch: relaunch}
}

// Settings are the current settings.
func (s *Service) Settings() Settings {
	theme := s.store.Theme()
	if theme == "" {
		theme = system
	}
	scale := s.store.TextScale()
	if scale == 0 {
		scale = defaultScale
	}
	key := s.project.Current().Key()
	return Settings{
		Theme:     theme,
		TextScale: scale,
		KeepAwake: s.store.KeepsAwake(),
		Cleanup:   s.store.Cleanup(key),
		Prompt:    s.store.PromptSend(key),
		Browser:   s.store.BrowserEnabled(key),
		Digest:    s.store.DigestSchedule(key),
		DataDir:   s.dataDir,
		DataFixed: s.dataFixed,
	}
}

// SetTheme saves the theme: system, light or dark.
func (s *Service) SetTheme(theme string) (Settings, error) {
	if theme != system && !slices.Contains(project.Themes, theme) {
		return Settings{}, fmt.Errorf("theme must be system, light or dark, not %q", theme)
	}
	if theme == system {
		theme = ""
	}
	return s.saved(s.store.SetTheme(theme))
}

// SetCleanup saves how the current project cleans up after an event, "merge" or "close": "auto" or "manual".
func (s *Service) SetCleanup(event, mode string) (Settings, error) {
	return s.saved(s.store.SetCleanup(s.project.Current().Key(), event, mode))
}

// SetTextScale saves the text size as a multiple of the default, from 0.85 to 1.5.
func (s *Service) SetTextScale(scale float64) (Settings, error) {
	if err := project.CheckTextScale(scale); err != nil {
		return Settings{}, err
	}
	if scale == defaultScale {
		scale = 0
	}
	return s.saved(s.store.SetTextScale(scale))
}

// SetKeepAwake saves whether a working session holds off idle sleep. A project's own setting still wins.
func (s *Service) SetKeepAwake(on bool) (Settings, error) {
	return s.saved(s.store.SetKeepAwake(on))
}

// SetPromptSend saves whether the current project sends the text a session starts with at once: "auto" or "manual".
func (s *Service) SetPromptSend(mode string) (Settings, error) {
	return s.saved(s.store.SetPromptSend(s.project.Current().Key(), mode))
}

// SetBrowserEnabled saves whether the current project's sessions get the browser and evidence commands.
func (s *Service) SetBrowserEnabled(on bool) (Settings, error) {
	return s.saved(s.store.SetBrowserEnabled(s.project.Current().Key(), on))
}

// SetDigestSchedule saves how the current project runs its digest: "weekly" or "off".
func (s *Service) SetDigestSchedule(schedule string) (Settings, error) {
	return s.saved(s.store.SetDigestSchedule(s.project.Current().Key(), schedule))
}

func (s *Service) saved(err error) (Settings, error) {
	if err != nil {
		return Settings{}, err
	}
	return s.Settings(), nil
}
