package sessions

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
			t.Setenv("npm_config_cache", t.TempDir())
			s := &Sessions{agent: tt.agent, browsers: fakeBrowsers{tt.browser}}
			tt.proj.Dir = "/work/p"
			argv := s.commandFor(session.Name{Project: "p", Token: "a1"}, tt.proj, project.Model{}, "", "", false)
			prompt, config, denied := slices.Index(argv, "--append-system-prompt"), slices.Index(argv, "--mcp-config"), slices.Index(argv, "--disallowedTools")
			if (config >= 0) != tt.want || (denied >= 0) != tt.want {
				t.Fatalf("argv = %q", argv)
			}
			_, claude := tt.agent.(agent.Claude)
			if (prompt >= 0) != claude {
				t.Fatalf("argv = %q", argv)
			}
			if claude && !strings.HasPrefix(argv[prompt+1], prompts.Session()+"\n\n"+prompts.SessionWork()) {
				t.Errorf("system prompt = %q", argv[prompt+1])
			}
			if claude && strings.Contains(argv[prompt+1], prompts.BrowserSession()) != tt.want {
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
	argv := claude.commandFor(name, project.Project{Name: "p"}, project.Model{}, "abc", "", false)
	if i := slices.Index(argv, "--resume"); i < 0 || argv[i+1] != "abc" {
		t.Errorf("argv = %q", argv)
	}
	plain := &Sessions{agent: agent.Plain{Argv: []string{"bash"}}, browsers: fakeBrowsers{}}
	if argv := plain.commandFor(name, project.Project{Name: "p"}, project.Model{}, "abc", "", false); !slices.Equal(argv, []string{"bash"}) {
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

// fakeNpxCache makes npm cache folders the way npx fills them: _npx/<hash>/node_modules/<package>, with a .bin link.
func fakeNpxCache(t *testing.T, hashes map[string]string, withBin bool) string {
	t.Helper()
	cache := t.TempDir()
	for hash, version := range hashes {
		modules := filepath.Join(cache, "_npx", hash, "node_modules")
		if err := os.MkdirAll(filepath.Join(modules, "chrome-devtools-mcp"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(modules, "chrome-devtools-mcp", "package.json"), []byte(`{"name":"chrome-devtools-mcp","version":"`+version+`"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		if withBin {
			if err := os.MkdirAll(filepath.Join(modules, ".bin"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(modules, ".bin", "chrome-devtools-mcp"), []byte("#!/usr/bin/env node\n"), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	return cache
}

func TestMCPServerRunsFromTheNpxCacheWhenThePinnedVersionIsThere(t *testing.T) {
	cache := fakeNpxCache(t, map[string]string{"aaaa": "1.9.0", "bbbb": chromeDevToolsMCPVersion}, true)
	bin := filepath.Join(cache, "_npx", "bbbb", "node_modules", ".bin", "chrome-devtools-mcp")
	want := `{"mcpServers":{"browser":{"command":` + strconv.Quote(bin) + `,"args":["--browser-url=http://127.0.0.1:23456",` +
		`"--no-usage-statistics","--no-performance-crux","--workspace=/work/p"],"env":{"CHROME_DEVTOOLS_MCP_NO_UPDATE_CHECKS":"1"}}}}`
	if got := browserMCPConfig(23456, "/work/p", cache); got != want {
		t.Errorf("config = %s\nwant %s", got, want)
	}
}

func TestMCPServerFallsBackToNpx(t *testing.T) {
	tests := []struct {
		name   string
		cache  string
		hashes map[string]string
		bin    bool
	}{
		{"no cache folder", "", nil, false},
		{"empty cache", "", map[string]string{}, true},
		{"another version only", "", map[string]string{"aaaa": "1.9.0"}, true},
		{"the version without its command file", "", map[string]string{"aaaa": chromeDevToolsMCPVersion}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := tt.cache
			if tt.hashes != nil {
				cache = fakeNpxCache(t, tt.hashes, tt.bin)
			}
			if got := browserMCPConfig(23456, "/work/p", cache); got != wantMCPConfig {
				t.Errorf("config = %s\nwant %s", got, wantMCPConfig)
			}
		})
	}
}

func TestNpmCacheDir(t *testing.T) {
	t.Setenv("npm_config_cache", "/somewhere/cache")
	if got := npmCacheDir(); got != "/somewhere/cache" {
		t.Errorf("npmCacheDir with the variable set = %q", got)
	}
	t.Setenv("npm_config_cache", "")
	home, _ := os.UserHomeDir()
	if got := npmCacheDir(); got != filepath.Join(home, ".npm") {
		t.Errorf("npmCacheDir without it = %q", got)
	}
}

func TestCommandGivesABriefedSessionOnlyTheBrief(t *testing.T) {
	name := session.Name{Project: "p", Token: "a1"}
	s := &Sessions{agent: agent.Claude{Exe: "/x"}, browsers: fakeBrowsers{available: true}}
	argv := s.commandFor(name, project.Project{Name: "p"}, project.Model{}, "", "set it up", false)
	if i := slices.Index(argv, "--append-system-prompt"); i < 0 || argv[i+1] != "set it up" || slices.Contains(argv, "--mcp-config") {
		t.Errorf("argv = %q", argv)
	}
}

func TestCommandMakesAChatReadOnlyWithoutABrowser(t *testing.T) {
	t.Setenv("npm_config_cache", t.TempDir())
	name := session.Name{Project: "p", Token: "a1"}
	s := &Sessions{agent: agent.Claude{Exe: "/x"}, browsers: fakeBrowsers{available: true}}
	argv := s.commandFor(name, project.Project{Name: "p", Dir: "/work/p"}, project.Model{}, "", "", true)
	if i := slices.Index(argv, "--permission-mode"); i < 0 || argv[i+1] != "plan" {
		t.Errorf("no plan mode: %q", argv)
	}
	const tools = "Bash(agentos new:*),Bash(agentos note:*),Bash(gh issue create:*)"
	if i := slices.Index(argv, "--allowedTools"); i < 0 || argv[i+1] != tools {
		t.Errorf("allowed tools: %q", argv)
	}
	if slices.Contains(argv, "--mcp-config") || slices.Contains(argv, "--disallowedTools") {
		t.Errorf("a chat has a browser: %q", argv)
	}
	want := prompts.Session() + "\n\n" + prompts.Chat()
	if i := slices.Index(argv, "--append-system-prompt"); i < 0 || argv[i+1] != want {
		t.Errorf("system prompt = %q", argv)
	}

	work := s.commandFor(name, project.Project{Name: "p", Dir: "/work/p"}, project.Model{}, "", "", false)
	if slices.Contains(work, "--permission-mode") || slices.Contains(work, "--allowedTools") {
		t.Errorf("a session runs read-only: %q", work)
	}
}

func TestCommandOfAPlainAgentIgnoresChat(t *testing.T) {
	s := &Sessions{agent: agent.Plain{Argv: []string{"bash"}}, browsers: fakeBrowsers{}}
	if argv := s.commandFor(session.Name{Project: "p", Token: "a1"}, project.Project{Name: "p"}, project.Model{}, "", "", true); !slices.Equal(argv, []string{"bash"}) {
		t.Errorf("argv = %q", argv)
	}
}
