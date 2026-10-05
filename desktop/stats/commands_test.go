package stats_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/nednella/agentos/desktop/stats"
	ctl "github.com/nednella/agentos/internal/control"
)

func TestStatsCommand(t *testing.T) {
	h := newHarness(t)
	s, err := h.NewSession("asker", "")
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"yarn test api"}}`)
	h.Hook(t, s.ID, "Notification", `{"message":"needs permission","notification_type":"permission_prompt"}`)
	eventually(t, "a wait", func() bool { st, _ := h.Stats(7); return st.Total == 1 })

	text := h.Ask(t, ctl.Request{Cmd: "stats", Session: s.ID, Opts: map[string]string{"days": "30"}})
	if !text.OK || !strings.Contains(text.Out, "over the last 30 days: 1") || !strings.Contains(text.Out, "Bash: yarn test api") {
		t.Errorf("text = %+v", text)
	}
	raw := h.Ask(t, ctl.Request{Cmd: "stats", Session: s.ID, Opts: map[string]string{"json": "1"}})
	var st stats.Stats
	if err := json.Unmarshal([]byte(raw.Out), &st); err != nil || st.Days != 7 || st.Total != 1 {
		t.Errorf("json = %+v, %v", st, err)
	}
	if resp := h.Ask(t, ctl.Request{Cmd: "stats", Session: s.ID, Opts: map[string]string{"days": "0"}}); resp.OK {
		t.Error("--days 0 was accepted")
	}
}
