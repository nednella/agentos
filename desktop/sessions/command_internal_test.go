package sessions

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/session"
)

type fakeBrowsers struct{ ready bool }

func (f fakeBrowsers) AgentReady() bool                          { return f.ready }
func (f fakeBrowsers) AgentEnv(string, string) ([]string, error) { return nil, nil }
func (f fakeBrowsers) Has(string) bool                           { return false }
func (f fakeBrowsers) Close(context.Context, string)             {}
func (f fakeBrowsers) Forget(context.Context, string)            {}
func (f fakeBrowsers) PIDs() []int                               { return nil }

// commandFor is the argv the session's agent p starts with.
func (s *Sessions) commandFor(p agent.Provider, proj project.Project, model project.Model, conversation, brief, handover, start string, chat bool) []string {
	return p.Command(launchFor(model, conversation, brief, handover, start, chat, s.hasBrowser(p, proj, brief)))
}

// allowedTools is the --allowedTools value in argv, "" when there is none.
func allowedTools(argv []string) string {
	if i := slices.Index(argv, "--allowedTools"); i >= 0 {
		return argv[i+1]
	}
	return ""
}

func TestClaudeGetsTheBrowserToolsAndPrompt(t *testing.T) {
	off := false
	tests := []struct {
		name    string
		agent   agent.Provider
		proj    project.Project
		browser bool
		want    bool
	}{
		{"claude with a browser", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, true, true},
		{"project turns it off", agent.Claude{Exe: "/x"}, project.Project{Name: "p", Browser: &off}, true, false},
		{"no browser or agent-browser", agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, false, false},
		{"plain agent", agent.Plain{Argv: []string{"claude"}}, project.Project{Name: "p"}, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Sessions{browsers: fakeBrowsers{tt.browser}}
			argv := s.commandFor(tt.agent, tt.proj, project.Model{}, "", "", "", "", false)
			if (allowedTools(argv) == "Bash(agent-browser:*)") != tt.want {
				t.Fatalf("argv = %q", argv)
			}
			prompt := slices.Index(argv, "--append-system-prompt")
			_, claude := tt.agent.(agent.Claude)
			if (prompt >= 0) != claude {
				t.Fatalf("argv = %q", argv)
			}
			if claude && !strings.HasPrefix(argv[prompt+1], prompts.Session()+"\n\n"+prompts.Task()) {
				t.Errorf("system prompt = %q", argv[prompt+1])
			}
			if claude && strings.Contains(argv[prompt+1], prompts.Browser()) != tt.want {
				t.Errorf("system prompt = %q", argv[prompt+1])
			}
		})
	}
}

func TestCommandResumesAConversation(t *testing.T) {
	claude := &Sessions{browsers: fakeBrowsers{}}
	argv := claude.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, project.Model{}, "abc", "", "", "", false)
	if i := slices.Index(argv, "--resume"); i < 0 || argv[i+1] != "abc" {
		t.Errorf("argv = %q", argv)
	}
	plain := &Sessions{browsers: fakeBrowsers{}}
	if argv := plain.commandFor(agent.Plain{Argv: []string{"bash"}}, project.Project{Name: "p"}, project.Model{}, "abc", "", "", "", false); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}

func TestConversationOfARunningAndAnEndedSession(t *testing.T) {
	written := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(written, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Sessions{
		records: map[string]session.Record{"p/1": {Conversation: "live", Transcript: written}, "p/4": {Conversation: "unwritten", Transcript: written + ".none"}},
		ended:   map[string]*endedSession{"p/2": {rec: session.Record{Conversation: "gone", Transcript: written}}},
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
	if got := s.conversationOf("p/4"); got != "" {
		t.Errorf("no transcript: %q", got)
	}
}

func TestCommandGivesABriefedSessionOnlyTheBrief(t *testing.T) {
	s := &Sessions{browsers: fakeBrowsers{ready: true}}
	argv := s.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, project.Model{}, "", "set it up", "", "", false)
	if i := slices.Index(argv, "--append-system-prompt"); i < 0 || argv[i+1] != "set it up" || allowedTools(argv) != "" {
		t.Errorf("argv = %q", argv)
	}
}

func TestCommandEndsWithTheStartPrompt(t *testing.T) {
	s := &Sessions{browsers: fakeBrowsers{}}
	for _, brief := range []string{"", "set it up"} {
		argv := s.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, project.Model{}, "", brief, "", "-v is a flag", false)
		if !slices.Equal(argv[len(argv)-2:], []string{"--", "-v is a flag"}) {
			t.Errorf("argv with brief %q = %q", brief, argv)
		}
	}
	plain := &Sessions{browsers: fakeBrowsers{}}
	if argv := plain.commandFor(agent.Plain{Argv: []string{"bash"}}, project.Project{Name: "p"}, project.Model{}, "", "", "", "/work 7", false); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}

func TestCommandEndsATasksPromptWithTheHandover(t *testing.T) {
	s := &Sessions{browsers: fakeBrowsers{}}
	argv := s.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p"}, project.Model{}, "", "", "the patch is in /tmp/x", "", false)
	want := prompts.Session() + "\n\n" + prompts.Task() + "\n\n" + prompts.Handover("the patch is in /tmp/x")
	if i := slices.Index(argv, "--append-system-prompt"); i < 0 || argv[i+1] != want {
		t.Errorf("argv = %q", argv)
	}
}

func TestCommandMakesAChatReadOnlyWithTheBrowser(t *testing.T) {
	s := &Sessions{browsers: fakeBrowsers{ready: true}}
	argv := s.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p", Dir: "/work/p"}, project.Model{}, "", "", "", "", true)
	if i := slices.Index(argv, "--permission-mode"); i < 0 || argv[i+1] != "plan" {
		t.Errorf("no plan mode: %q", argv)
	}
	const tools = "Bash(agentos issue:*),Bash(agentos new:*),Bash(agentos note:*),Bash(agentos refresh:*),Bash(agentos browser:*),Bash(gh issue create:*),Bash(gh issue comment:*),Bash(gh issue edit:*),Bash(agent-browser:*)"
	if got := allowedTools(argv); got != tools {
		t.Errorf("allowed tools = %q", got)
	}
	want := prompts.Session() + "\n\n" + prompts.Chat() + "\n\n" + prompts.Browser()
	if i := slices.Index(argv, "--append-system-prompt"); i < 0 || argv[i+1] != want {
		t.Errorf("system prompt = %q", argv)
	}

	work := s.commandFor(agent.Claude{Exe: "/x"}, project.Project{Name: "p", Dir: "/work/p"}, project.Model{}, "", "", "", "", false)
	if slices.Contains(work, "--permission-mode") || allowedTools(work) != "Bash(agent-browser:*)" {
		t.Errorf("a task runs read-only or without the browser: %q", work)
	}
}

func TestCommandOfAPlainAgentIgnoresChat(t *testing.T) {
	s := &Sessions{browsers: fakeBrowsers{}}
	if argv := s.commandFor(agent.Plain{Argv: []string{"bash"}}, project.Project{Name: "p"}, project.Model{}, "", "", "", "", true); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}
