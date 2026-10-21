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
	parts, _, err := c.Read(path, "")
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

	again, _, _ := c.Read(path, "")
	if len(again) != 0 {
		t.Errorf("a second read with nothing new = %+v", again)
	}
	appendTo(t, path, assistant("msg_3", "claude-opus-5-5", t0.Add(3*time.Second), 0, 7, 0, 0))
	next, _, _ := c.Read(path, "")
	if len(next) != 1 || next[0].Output != 7 {
		t.Errorf("new line read = %+v, want 7 output tokens", next)
	}
}

func TestReadSaysWhatTheAgentUsedLast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	appendTo(t, path, assistant("msg_old", "claude-opus-5-5", t0.Add(-time.Hour), 1, 1, 0, 0))
	c := NewCursor(t0)
	if _, cur, _ := c.Read(path, ""); cur != (Current{}) {
		t.Errorf("current from a line before the start = %+v, want none", cur)
	}
	withEffort := func(id, model, effort string, at time.Time) string {
		return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"effort":%q,"message":{"id":%q,"model":%q,"usage":{"input_tokens":1,"output_tokens":1}}}`+"\n",
			at.Format(time.RFC3339Nano), effort, id, model)
	}
	appendTo(t, path, withEffort("msg_1", "claude-opus-5-5", "high", t0.Add(time.Second)),
		withEffort("msg_2", "claude-fable-5-1", "medium", t0.Add(2*time.Second)), withEffort("msg_2", "claude-fable-5-1", "medium", t0.Add(2*time.Second)))
	appendTo(t, filepath.Join(t.TempDir(), "conv", "subagents", "agent-x.jsonl"),
		withEffort("msg_3", "claude-haiku-5-5", "low", t0.Add(3*time.Second)))
	_, cur, err := c.Read(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if cur != (Current{Model: "claude-fable-5-1", Effort: "medium"}) {
		t.Errorf("current = %+v, want the newest line of the session's own transcript", cur)
	}
	if _, again, _ := c.Read(path, ""); again != cur {
		t.Errorf("current after a read with nothing new = %+v, want %+v", again, cur)
	}
}

func TestReadSkipsWhatCameBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	appendTo(t, path,
		assistant("msg_old", "claude-opus-5-5", t0.Add(-time.Hour), 500, 500, 0, 0),
		assistant("msg_new", "claude-opus-5-5", t0.Add(time.Minute), 1, 2, 0, 0))
	parts, _, _ := NewCursor(t0).Read(path, "")
	if len(parts) != 1 || parts[0].Input != 1 || parts[0].Output != 2 {
		t.Errorf("parts = %+v, want only the message after the start", parts)
	}
}

func TestReadLeavesAPartLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	full := assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 1, 2, 0, 0)
	appendTo(t, path, full[:40])
	c := NewCursor(t0)
	if parts, _, _ := c.Read(path, ""); len(parts) != 0 {
		t.Fatalf("a part line was counted: %+v", parts)
	}
	appendTo(t, path, full[40:])
	if parts, _, _ := c.Read(path, ""); len(parts) != 1 || parts[0].Output != 2 {
		t.Errorf("the finished line = %+v", parts)
	}
}

func TestReadAddsSubagents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "abc.jsonl")
	appendTo(t, path, assistant("msg_1", "claude-opus-5-5", t0.Add(time.Second), 1, 1, 0, 0))
	appendTo(t, filepath.Join(dir, "abc", "subagents", "agent-x.jsonl"),
		assistant("msg_2", "claude-haiku-5-5", t0.Add(2*time.Second), 3, 3, 0, 0))
	parts, _, err := NewCursor(t0).Read(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 2 || parts[0].Subagent || !parts[1].Subagent || parts[1].Model != "claude-haiku-5-5" || parts[1].Input != 3 {
		t.Errorf("parts = %+v, want the session's then the subagent's", parts)
	}
}

func TestReadMissingFile(t *testing.T) {
	parts, _, err := NewCursor(t0).Read(filepath.Join(t.TempDir(), "none.jsonl"), "")
	if err != nil || len(parts) != 0 {
		t.Errorf("Read = %+v, %v; want nothing and no error", parts, err)
	}
}

func TestCost(t *testing.T) {
	var u apiUsage
	u.InputTokens, u.OutputTokens, u.CacheReadInputTokens = 1_000_000, 1_000_000, 1_000_000
	u.CacheCreationInputTokens, u.CacheCreation.Ephemeral1h = 1_000_000, 1_000_000
	// Claude Opus 5.5: $4 in, $20 out, $0.20 cache read, $8 for an hour-long cache write.
	if got := cost("claude-opus-5-5", u, t0); got != 32_200_000 {
		t.Errorf("opus 5.5 cost = %d, want 32200000", got)
	}
	u.Speed = "fast"
	if got := cost("claude-opus-5-5", u, t0); got != 64_400_000 {
		t.Errorf("fast cost = %d, want twice the standard", got)
	}
	u.Speed = ""
	u.CacheCreation.Ephemeral1h = 0 // no split: a 5-minute write at 1.25 times input
	if got := cost("claude-opus-5", u, t0); got != 5_000_000+25_000_000+500_000+6_250_000 {
		t.Errorf("opus 5 cost = %d", got)
	}
	if got := cost("claude-haiku-5-5", apiUsage{InputTokens: 200_000}, t0); got != 100_000 {
		t.Errorf("long haiku prompt = %d, want 200000 tokens at $0.50", got)
	}
	if got := cost("<synthetic>", u, t0); got != 0 {
		t.Errorf("unknown model cost = %d, want 0", got)
	}
}

func TestReadAfterARestartSkipsTheLatestCounted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	at := t0.Add(1234567 * time.Microsecond)
	appendTo(t, path, assistant("msg_1", "claude-opus-5-5", at, 1, 1, 0, 0))
	// The event log keeps the time of the latest message counted in milliseconds.
	if parts, _, _ := NewCursor(time.UnixMilli(at.UnixMilli())).Read(path, ""); len(parts) != 0 {
		t.Errorf("the message counted before the restart was counted again: %+v", parts)
	}
}

func TestCostOfAgyModels(t *testing.T) {
	u := apiUsage{InputTokens: 1_000_000, OutputTokens: 1_000_000, CacheReadInputTokens: 1_000_000}
	// Gemini 4 Argon: $2 in, $10 out, $0.10 cached until the end of November 2026, then twice that.
	if got := cost("gemini-4-argon-medium", u, time.Date(2026, 11, 30, 23, 59, 59, 0, time.UTC)); got != 12_100_000 {
		t.Errorf("argon cost = %d, want 12100000", got)
	}
	if got := cost("gemini-4-argon-high", u, argonRise); got != 24_200_000 {
		t.Errorf("argon cost from December = %d, want 24200000", got)
	}
	if got := cost("gemini-3.8-flash-low", u, t0); got != 4_500_000 {
		t.Errorf("flash cost = %d, want 4500000", got)
	}
	// A claude id from agy ends in its effort and still prices as that claude model.
	if got := cost("claude-opus-5-5-low", apiUsage{InputTokens: 1_000_000}, t0); got != 4_000_000 {
		t.Errorf("claude through agy cost = %d, want 4000000", got)
	}
}
