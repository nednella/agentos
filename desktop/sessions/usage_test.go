package sessions_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func assistantLine(id string, at time.Time, input, output int64) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"message":{"id":%q,"model":"claude-opus-5-5","usage":{"input_tokens":%d,"output_tokens":%d}}}`+"\n",
		at.UTC().Format(time.RFC3339Nano), id, input, output)
}

func appendLines(t *testing.T, path string, lines ...string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, l := range lines {
		if _, err := f.WriteString(l); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUsageIsCountedOncePerMessage(t *testing.T) {
	h := newClaudeHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "conv.jsonl")
	stop := fmt.Sprintf(`{"transcript_path":%q}`, transcript)
	reply := time.Now().Add(100 * time.Millisecond) // a reply comes after the session started
	appendLines(t, transcript,
		assistantLine("msg_old", time.Now().Add(-time.Hour), 999, 999), // an earlier session of a resumed conversation
		assistantLine("msg_1", reply, 1000, 100), assistantLine("msg_1", reply, 1000, 100))
	h.Hook(t, s.ID, "Stop", stop)

	eventually(t, "the turn's tokens on the row", func() bool {
		row, ok := h.Session(s.ID)
		return ok && row.Tokens == 1100
	})
	appendLines(t, transcript, assistantLine("msg_2", time.Now().Add(time.Second), 10, 1))
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"more"}`)
	h.Hook(t, s.ID, "Stop", stop)
	eventually(t, "the second turn's tokens", func() bool {
		row, _ := h.Session(s.ID)
		return row.Tokens == 1111
	})
	row, _ := h.Session(s.ID)
	if row.Cost != 0.00606 {
		t.Errorf("cost = %v, want 1010 input at $4 and 101 output at $20 per million", row.Cost)
	}

	second := h.Restart(t)
	eventually(t, "the total read back from the event log", func() bool {
		row, ok := second.Session(s.ID)
		return ok && row.Tokens == 1111
	})
	second.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"again"}`)
	second.Hook(t, s.ID, "Stop", stop)
	time.Sleep(200 * time.Millisecond)
	led, err := second.Ledger(30, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(led.Work) != 1 || led.Work[0].Issue != 4 || led.Work[0].Tokens != 1111 || led.Usage.Totals.Sessions != 1 {
		t.Errorf("work = %+v, usage = %+v: a message was counted twice or lost", led.Work, led.Usage)
	}
}

func TestUsageIsCountedAtEachHookAndRecordedAtTheEndOfATurn(t *testing.T) {
	h := newClaudeHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4, "")
	if err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(t.TempDir(), "conv.jsonl")
	h.Hook(t, s.ID, "UserPromptSubmit", fmt.Sprintf(`{"prompt":"go","transcript_path":%q}`, transcript))
	appendLines(t, transcript, assistantLine("msg_1", time.Now().Add(100*time.Millisecond), 1000, 100))
	h.Hook(t, s.ID, "PostToolUse", `{"tool_name":"Bash"}`)
	eventually(t, "the tokens of a turn still running", func() bool {
		row, _ := h.Session(s.ID)
		return row.Tokens == 1100
	})
	if led, err := h.Ledger(30, ""); err != nil || led.Usage.Totals.Tokens != 0 {
		t.Fatalf("usage = %+v, %v: the event log got usage before the turn ended", led.Usage.Totals, err)
	}

	h.Hook(t, s.ID, "Stop", "{}")
	// Claude Code writes a turn's last reply a few milliseconds after its Stop hook.
	appendLines(t, transcript, assistantLine("msg_2", time.Now().Add(time.Second), 10, 1))
	eventually(t, "the reply written after the Stop", func() bool {
		row, _ := h.Session(s.ID)
		return row.Tokens == 1111
	})
	led, err := h.Ledger(30, "")
	if err != nil {
		t.Fatal(err)
	}
	if led.Usage.Totals.Tokens != 1111 || len(led.Work) != 1 || led.Work[0].Tokens != 1111 {
		t.Errorf("usage = %+v, work = %+v: the stats do not add up to the transcript", led.Usage.Totals, led.Work)
	}

	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"more"}`)
	appendLines(t, transcript, assistantLine("msg_3", time.Now().Add(2*time.Second), 100, 10))
	h.Hook(t, s.ID, "PostToolUse", `{"tool_name":"Bash"}`)
	second := h.Restart(t)
	eventually(t, "the recorded usage and the usage read again after a restart in a turn", func() bool {
		row, _ := second.Session(s.ID)
		return row.Tokens == 1221
	})
}
