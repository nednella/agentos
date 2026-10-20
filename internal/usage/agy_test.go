package usage

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// planner is an agy transcript line of one model call. agy writes its times to the second.
func planner(step int, at time.Time, in, read, out int64) string {
	return fmt.Sprintf(`{"step_index":%d,"source":"MODEL","type":"PLANNER_RESPONSE","status":"DONE","created_at":%q,"input_tokens":%d,"cache_read_tokens":%d,"output_tokens":%d,"content":"hi"}`+"\n",
		step, at.Format(time.RFC3339), in, read, out)
}

func TestAgyReadCountsEachStepOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	s1 := planner(1, t0.Add(time.Second), 10, 1000, 100)
	appendTo(t, path, `{"step_index":0,"source":"USER_EXPLICIT","type":"USER_INPUT","created_at":"2026-10-11T12:00:01Z","content":"go"}`+"\n",
		s1, s1, `{"step_index":2,"source":"MODEL","type":"GENERIC","created_at":"2026-10-11T12:00:02Z","content":"output"}`+"\n",
		planner(3, t0.Add(2*time.Second), 1, 1, 1))

	r := NewAgyReader(t0)
	parts, err := r.Read(path, "gemini-4-argon-medium")
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 1 {
		t.Fatalf("parts = %+v, want one", parts)
	}
	got := parts[0]
	// 11 in at $2, 101 out at $10, 1001 cached at $0.10: 22 + 1010 + 100.1 millionths.
	want := Tokens{Input: 11, Output: 101, CacheRead: 1001, Cost: 1132}
	if got.Tokens != want || got.Model != "gemini-4-argon-medium" || got.Subagent {
		t.Errorf("part = %+v, want %+v of gemini-4-argon-medium", got, want)
	}
	if !r.After().Equal(t0.Add(2 * time.Second)) {
		t.Errorf("after = %v", r.After())
	}

	if again, _ := r.Read(path, "gemini-4-argon-medium"); len(again) != 0 {
		t.Errorf("a second read with nothing new = %+v", again)
	}
	appendTo(t, path, planner(5, t0.Add(3*time.Second), 0, 0, 7))
	next, _ := r.Read(path, "claude-opus-5-5-low")
	if len(next) != 1 || next[0].Output != 7 || next[0].Model != "claude-opus-5-5-low" || next[0].Cost != 140 {
		t.Errorf("new line read = %+v, want 7 output tokens of claude-opus-5-5-low at $20", next)
	}
}

func TestAgyReadSkipsWhatCameBefore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	appendTo(t, path, planner(1, t0.Add(-time.Hour), 500, 0, 500), planner(3, t0.Add(time.Minute), 1, 0, 2))
	parts, _ := NewAgyReader(t0).Read(path, "gemini-4-argon-medium")
	if len(parts) != 1 || parts[0].Input != 1 || parts[0].Output != 2 {
		t.Errorf("parts = %+v, want only the step after the start", parts)
	}
}

func TestAgyReadLeavesAPartLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "transcript_full.jsonl")
	full := planner(1, t0.Add(time.Second), 1, 0, 2)
	appendTo(t, path, full[:40])
	r := NewAgyReader(t0)
	if parts, _ := r.Read(path, "gemini-4-argon-medium"); len(parts) != 0 {
		t.Fatalf("a part line was counted: %+v", parts)
	}
	appendTo(t, path, full[40:])
	if parts, _ := r.Read(path, "gemini-4-argon-medium"); len(parts) != 1 || parts[0].Output != 2 {
		t.Errorf("the finished line = %+v", parts)
	}
}

func TestAgyReadMissingFile(t *testing.T) {
	parts, err := NewAgyReader(t0).Read(filepath.Join(t.TempDir(), "none.jsonl"), "gemini-4-argon-medium")
	if err != nil || len(parts) != 0 {
		t.Errorf("Read = %+v, %v; want nothing and no error", parts, err)
	}
}
