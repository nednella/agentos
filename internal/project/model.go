package project

import (
	"cmp"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// Model is the model and effort level a session's agent runs at. An empty field means unset.
type Model struct {
	Model  string `yaml:"model,omitempty"`
	Effort string `yaml:"effort,omitempty"`
}

const (
	DefaultModel  = "sonnet"
	DefaultEffort = "medium"
)

var efforts = []string{"low", "medium", "high", "xhigh", "max"}

// modelName keeps a value from being read as a flag when it reaches the agent's command line.
var modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\[\]-]*$`)

// defaultLaneModels is what a lane runs at when neither the config nor a label says otherwise.
// Planning gets the stronger model; every other lane runs at the project's default.
var defaultLaneModels = map[string]Model{"plan": {Model: "opus"}}

func (m Model) over(base Model) Model {
	return Model{Model: cmp.Or(m.Model, base.Model), Effort: cmp.Or(m.Effort, base.Effort)}
}

// Validate reports a model or effort the agent would not accept.
func (m Model) Validate() error {
	if m.Model != "" && !modelName.MatchString(m.Model) {
		return fmt.Errorf("model %q is not a model name", m.Model)
	}
	if m.Effort != "" && !slices.Contains(efforts, m.Effort) {
		return fmt.Errorf("effort must be one of %s, not %q", strings.Join(efforts, ", "), m.Effort)
	}
	return nil
}

// Default is the model and effort of a config that sets neither.
func (c Config) Default() Model {
	return Model{Model: c.Model, Effort: c.Effort}.over(Model{Model: DefaultModel, Effort: DefaultEffort})
}

// Pick chooses the model and effort for a session started for an issue in the lane, or for no issue
// when the lane is "". From weakest to strongest: base (the config's default), the project's own,
// the built-in choice of the lane, the project's choice for the lane, then the issue's `model:<x>`
// and `effort:<y>` labels. A value that fails Validate is skipped.
func (p Project) Pick(base Model, lane string, labels []string) Model {
	m := Model{Model: p.Model, Effort: p.Effort}.over(base)
	m = defaultLaneModels[lane].over(m)
	m = p.Models[lane].over(m)
	for _, l := range slices.Backward(labels) {
		var override Model
		switch {
		case strings.HasPrefix(l, "model:"):
			override.Model = strings.TrimPrefix(l, "model:")
		case strings.HasPrefix(l, "effort:"):
			override.Effort = strings.TrimPrefix(l, "effort:")
		}
		if override.Validate() == nil {
			m = override.over(m)
		}
	}
	return m
}

func (p Project) validateModels() error {
	if err := (Model{Model: p.Model, Effort: p.Effort}).Validate(); err != nil {
		return err
	}
	for lane, m := range p.Models {
		if err := m.Validate(); err != nil {
			return fmt.Errorf("models.%s: %w", lane, err)
		}
	}
	return nil
}
