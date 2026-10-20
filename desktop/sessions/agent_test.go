package sessions_test

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/sessions"
	ctl "github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/project"
)

func sessionNumbered(t *testing.T, h *apptest.Harness, n int) sessions.Session {
	t.Helper()
	list := h.Sessions().List()
	i := slices.IndexFunc(list, func(s sessions.Session) bool { return s.N == n })
	if i < 0 {
		t.Fatalf("no session %d in %+v", n, list)
	}
	return list[i]
}

func TestNewWithAgentClaudeStartsAsWithoutTheFlag(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "claude"})
	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"flagged"}, Opts: map[string]string{"agent": "claude"}}); !resp.OK {
		t.Fatalf("new --agent claude = %+v", resp)
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"plain"}}); !resp.OK {
		t.Fatalf("new = %+v", resp)
	}
	flagged, bare := sessionNumbered(t, h, 1), sessionNumbered(t, h, 2)
	if flagged.Agent != "claude" || bare.Agent != "claude" {
		t.Errorf("agents = %q and %q, want claude", flagged.Agent, bare.Agent)
	}
	if a, b := claudeArgs(t, dir, flagged.ID), claudeArgs(t, dir, bare.ID); !slices.Equal(a, b) {
		t.Errorf("--agent claude started\n%q\nwithout the flag\n%q", a, b)
	}
}

func TestNewAndIssueRefuseAnUnknownAgent(t *testing.T) {
	h := newHarness(t)
	for _, cmd := range []string{"new", "issue"} {
		resp := h.Ask(t, ctl.Request{Cmd: cmd, Args: []string{"12"}, Opts: map[string]string{"agent": "codex"}})
		if resp.OK || !strings.Contains(resp.Error, `no agent "codex": the agents are claude, agy`) {
			t.Errorf("%s --agent codex = %+v", cmd, resp)
		}
	}
	if len(h.Sessions().List()) != 0 {
		t.Errorf("a session started: %+v", h.Sessions().List())
	}
}

func TestAgentSettingAppliesToTheNextSession(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := newHarness(t)
	if got := h.Settings(); got.Agent != "bash" || !slices.Equal(got.Agents, []string{"claude", "agy"}) {
		t.Fatalf("settings = %+v", got)
	}
	plain := plainSession(h, t)
	if plain.Agent != "bash" {
		t.Errorf("agent of a session on the default = %q, want bash", plain.Agent)
	}

	if got, err := h.SetAgent(" claude "); err != nil || got.Agent != "claude" {
		t.Fatalf("SetAgent = %+v, %v", got, err)
	}
	if cfg, err := project.Load(h.Conf); err != nil || cfg.Agent != "claude" {
		t.Errorf("config = %+v, %v", cfg, err)
	}
	claude, err := h.NewSession("work", "")
	if err != nil {
		t.Fatal(err)
	}
	if claude.Agent != "claude" {
		t.Errorf("agent of a session after the change = %q, want claude", claude.Agent)
	}
	if args := claudeArgs(t, dir, claude.ID); !slices.Contains(args, "--settings") {
		t.Errorf("the session did not run the claude provider: %q", args)
	}
	if _, err := h.SetAgent("  "); err == nil {
		t.Error("an empty agent was accepted")
	}
	if got := h.Settings().Agent; got != "claude" {
		t.Errorf("a rejected agent changed the setting to %q", got)
	}
}

func TestSessionKeepsItsAgentOverARestartAndAResume(t *testing.T) {
	dir := apptest.ClaudeOnPath(t)
	h := apptest.NewWith(t, apptest.Options{Agent: "bash"})
	if resp := h.Ask(t, ctl.Request{Cmd: "new", Args: []string{"#12", "work"}, Opts: map[string]string{"agent": "claude"}}); !resp.OK {
		t.Fatalf("new --agent claude = %+v", resp)
	}
	started := sessionNumbered(t, h, 1)
	claudeArgs(t, dir, started.ID)
	if started.Agent != "claude" {
		t.Fatalf("agent = %q", started.Agent)
	}
	if data, _ := os.ReadFile(h.Conf); !strings.Contains(string(data), "agent_command: bash") {
		t.Fatalf("the default changed:\n%s", data)
	}

	h.App.Stop()
	h = h.Restart(t)
	if got, ok := h.Session(started.ID); !ok || got.Agent != "claude" {
		t.Errorf("after a restart the session is %+v", got)
	}

	fresh := endedWhileClosed(t, h, started.ID)
	var gone sessions.Session
	eventually(t, "the ended row", func() bool { gone, _ = fresh.Session(started.ID); return gone.State == "ended" })
	if gone.Agent != "claude" || !gone.Resumable {
		t.Errorf("ended row = %+v", gone)
	}
	resumed, err := fresh.ResumeSession(started.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Agent != "claude" {
		t.Errorf("resumed row = %+v", resumed)
	}
	if args := claudeArgs(t, dir, resumed.ID); !slices.Contains(args, "--resume") {
		t.Errorf("the resumed session did not run claude: %q", args)
	}
}
