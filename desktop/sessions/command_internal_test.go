package sessions

import (
	"context"
	"slices"
	"testing"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/session"
)

type fakeBrowsers struct{ available bool }

func (f fakeBrowsers) Available() bool               { return f.available }
func (f fakeBrowsers) Has(string) bool               { return false }
func (f fakeBrowsers) Close(context.Context, string) {}

func TestClaudeGetsTheBrowserPrompt(t *testing.T) {
	off := false
	tests := []struct {
		name    string
		agent   agent.Agent
		proj    project.Project
		browser bool
		want    bool
	}{
		{"claude with a browser", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, true, true},
		{"project turns it off", agent.Claude{Exe: "/x"}, project.Project{Name: "p", Browser: &off}, true, false},
		{"no browser installed", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, false, false},
		{"plain agent", agent.Plain{Argv: []string{"claude"}}, project.Project{Name: "p"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sessions{agent: tt.agent, browsers: fakeBrowsers{tt.browser}}
			argv := s.commandFor(session.Name{Project: "p", N: 1}, tt.proj)
			i := slices.Index(argv, "--append-system-prompt")
			if (i >= 0) != tt.want || (tt.want && argv[i+1] != prompts.BrowserSession()) {
				t.Errorf("argv = %q", argv)
			}
		})
	}
}
