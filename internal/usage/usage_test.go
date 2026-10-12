package usage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

// assistant is a transcript line of an API response, written once per content block.
func assistant(id, model string, at time.Time, in, out, read, write int64) string {
	return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"message":{"id":%q,"model":%q,"usage":{"input_tokens":%d,"output_tokens":%d,"cache_read_input_tokens":%d,"cache_creation_input_tokens":%d,"cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":%d},"output_tokens_details":{"thinking_tokens":5}}}}`+"\n",
		at.Format(time.RFC3339Nano), id, model, in, out, read, write, write)
}

func appendTo(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
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

func TestReadCountsEachMessageOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	m1 := assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 10, 100, 1000, 2000)
	appendTo(t, path, `{"type":"user","timestamp":"2026-10-11T12:00:00Z","message":{"role":"user"}}`+"\n", m1, m1,
		assistant("msg_2", "claude-opus-5-5", t0.Add(2*time.Second), 1, 1, 1, 0))

	c := NewCursor(t0)
	parts, err := c.Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("parts = %+v, want one", parts)
	}
	got := parts[0]
	want := Tokens{Input: 11, Output: 101, CacheRead: 1001, CacheWrite: 2000, Thinking: 10}
	got.Cost = 0
	if got.Tokens != want || got.Model != "claude-opus-5-5" || got.Subagent {
		t.Errorf("part = %+v, want %+v of claude-opus-5-5", got, want)
	}
	if !c.After().Equal(t0.Add(2 * time.Second)) {
		t.Errorf("after = %v", c.After())
	}

	again, _ := c.Read(path)
	if len(again) != 0 {
		t.Errorf("a second read with nothing new = %+v", again)
	}
	appendTo(t, path, assistant("msg_3", "claude-opus-5-5", t0.Add(3*time.Second), 0, 7, 0, 0))
	next, _ := c.Read(path)
	if len(next) != 1 || next[0].Output != 7 {
		t.Errorf("new line read = %+v, want 7 output tokens", next)
	}
}

func TestReadSkipsWhatCameBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	appendTo(t, path,
		assistant("msg_old", "claude-opus-5-5", t0.Add(-time.Hour), 500, 500, 0, 0),
		assistant("msg_new", "claude-opus-5-5", t0.Add(time.Minute), 1, 2, 0, 0))
	parts, _ := NewCursor(t0).Read(path)
	if len(parts) != 1 || parts[0].Input != 1 || parts[0].Output != 2 {
		t.Errorf("parts = %+v, want only the message after the start", parts)
	}
}

func TestReadLeavesAPartLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	full := assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 1, 2, 0, 0)
	appendTo(t, path, full[:40])
	c := NewCursor(t0)
	if parts, _ := c.Read(path); len(parts) != 0 {
		t.Fatalf("a part line was counted: %+v", parts)
	}
	appendTo(t, path, full[40:])
	if parts, _ := c.Read(path); len(parts) != 1 || parts[0].Output != 2 {
		t.Errorf("the finished line = %+v", parts)
	}
}

func TestUnread(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	c := NewCursor(t0)
	if c.Unread(path) {
		t.Error("a transcript not written yet has lines to read")
	}
	appendTo(t, path, assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 1, 2, 0, 0))
	if !c.Unread(path) {
		t.Error("a new line was not seen")
	}
	_, _ = c.Read(path)
	if c.Unread(path) {
		t.Error("a transcript read to its end has lines to read")
	}
}

func TestReadAddsSubagents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.jsonl")
	appendTo(t, path, assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 1, 1, 0, 0))
	appendTo(t, filepath.Join(dir, "abc", "subagents", "agent-x.jsonl"),
		assistant("msg_2", "claude-haiku-5-5", t0.Add(2*time.Second), 3, 3, 0, 0))
	parts, err := NewCursor(t0).Read(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 || parts[0].Subagent || !parts[1].Subagent || parts[1].Model != "claude-haiku-5-5" || parts[1].Input != 3 {
		t.Errorf("parts = %+v, want the session's then the subagent's", parts)
	}
}

func TestReadMissingFile(t *testing.T) {
	parts, err := NewCursor(t0).Read(filepath.Join(t.TempDir(), "none.jsonl"))
	if err != nil || len(parts) != 0 {
		t.Errorf("Read = %+v, %v; want nothing and no error", parts, err)
	}
}

func TestCost(t *testing.T) {
	var u apiUsage
	u.InputTokens, u.OutputTokens, u.CacheReadInputTokens = 1_000_000, 1_000_000, 1_000_000
	u.CacheCreationInputTokens, u.CacheCreation.Ephemeral1h = 1_000_000, 1_000_000
	// Claude Opus 5.5: $4 in, $20 out, $0.20 cache read, $8 for an hour-long cache write.
	if got := cost("claude-opus-5-5", u); got != 32_200_000 {
		t.Errorf("opus 5.5 cost = %d, want 32200000", got)
	}
	u.Speed = "fast"
	if got := cost("claude-opus-5-5", u); got != 64_400_000 {
		t.Errorf("fast cost = %d, want twice the standard", got)
	}
	u.Speed = ""
	u.CacheCreation.Ephemeral1h = 0 // no split: a 5-minute write at 1.25 times input
	if got := cost("claude-opus-5", u); got != 5_000_000+25_000_000+500_000+6_250_000 {
		t.Errorf("opus 5 cost = %d", got)
	}
	if got := cost("claude-haiku-5-5", apiUsage{InputTokens: 200_000}); got != 100_000 {
		t.Errorf("long haiku prompt = %d, want 200000 tokens at $0.50", got)
	}
	if got := cost("<synthetic>", u); got != 0 {
		t.Errorf("unknown model cost = %d, want 0", got)
	}
}

func TestReadAfterARestartSkipsTheLatestCounted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	at := t0.Add(1234567 * time.Microsecond)
	appendTo(t, path, assistant("msg_1", "claude-opus-5-5", at, 1, 1, 0, 0))
	// The event log keeps the time of the latest message counted in milliseconds.
	if parts, _ := NewCursor(time.UnixMilli(at.UnixMilli())).Read(path); len(parts) != 0 {
		t.Errorf("the message counted before the restart was counted again: %+v", parts)
	}
}
