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
func (f fakeBrowsers) Port(string) int               { return 23456 }

const wantMCPConfig = `{"mcpServers":{"browser":{"command":"npx","args":["-y","--prefer-offline","chrome-devtools-mcp@1.10.1",` +
	`"--browser-url=http://127.0.0.1:23456","--no-usage-statistics","--no-performance-crux","--workspace=/work/p"],` +
	`"env":{"CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS":"1"}}}}`

func TestClaudeGetsTheBrowserToolsAndPrompt(t *testing.T) {
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
			tt.proj.Dir = "/work/p"
			argv := s.commandFor(session.Name{Project: "p", Token: "a1"}, tt.proj, project.Model{}, "")
			prompt, config, denied := slices.Index(argv, "--append-system-prompt"), slices.Index(argv, "--mcp-config"), slices.Index(argv, "--disallowedTools")
			if (prompt >= 0) != tt.want || (config >= 0) != tt.want || (denied >= 0) != tt.want {
				t.Fatalf("argv = %q", argv)
			}
			if tt.want && argv[prompt+1] != prompts.BrowserSession() {
				t.Errorf("system prompt = %q", argv[prompt+1])
			}
			if tt.want && argv[config+1] != wantMCPConfig {
				t.Errorf("mcp config = %s\nwant %s", argv[config+1], wantMCPConfig)
			}
			if tt.want && argv[denied+1] != "mcp__browser__new_page mcp__browser__close_page" {
				t.Errorf("disallowed tools = %q", argv[denied+1])
			}
		})
	}
}

func TestCommandResumesAConversation(t *testing.T) {
	name := session.Name{Project: "p", Token: "a1"}
	claude := &Sessions{agent: agent.Claude{Exe: "/x"}, browsers: fakeBrowsers{}}
	argv := claude.commandFor(name, project.Project{Name: "p"}, project.Model{}, "abc")
	if i := slices.Index(argv, "--resume"); i < 0 || argv[i+1] != "abc" {
		t.Errorf("argv = %q", argv)
	}
	plain := &Sessions{agent: agent.Plain{Argv: []string{"bash"}}, browsers: fakeBrowsers{}}
	if argv := plain.commandFor(name, project.Project{Name: "p"}, project.Model{}, "abc"); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}

func TestConversationOfARunningAndAnEndedSession(t *testing.T) {
	s := &Sessions{
		records: map[string]session.Record{"p/1": {Conversation: "live"}},
		ended:   map[string]*endedSession{"p/2": {rec: session.Record{Conversation: "gone"}}},
	}
	if got := s.conversationOf("p/1"); got != "live" {
		t.Errorf("running: %q", got)
	}
	if got := s.conversationOf("p/2"); got != "gone" {
		t.Errorf("ended: %q", got)
	}
	if got := s.conversationOf("p/3"); got != "" {
		t.Errorf("unknown: %q", got)
	}
}
