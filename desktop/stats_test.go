package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/session"
)

func TestWaitsAreRecorded(t *testing.T) {
	h := newHarness(t)
	s := h.issueSession(t, 3)
	h.hook(t, s.ID, "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"upscope test api foo/bar.test.ts"}}`)
	h.hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Bash","notification_type":"permission_prompt"}`)
	eventually(t, "open wait", func() bool {
		st, _ := h.app.Stats(7)
		return st.Total == 1 && st.Recent[0].WaitedMs == 0
	})
	time.Sleep(20 * time.Millisecond)
	h.hook(t, s.ID, "PreToolUse", `{"tool_name":"Edit","tool_input":{"file_path":"a.ts"}}`)
	h.hook(t, s.ID, "Notification", `{"message":"waiting for your input","notification_type":"idle_prompt"}`)
	h.hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Edit","notification_type":"permission_prompt"}`)
	h.hook(t, s.ID, "PostToolUse", `{"tool_name":"Edit"}`)
	h.hook(t, s.ID, "Notification", `{"message":"pick one","notification_type":"elicitation_dialog"}`)
	h.hook(t, s.ID, "UserPromptSubmit", `{"prompt":"b"}`)
	h.hook(t, s.ID, "Stop", `{}`)
	eventually(t, "finished wait", func() bool {
		st, _ := h.app.Stats(7)
		return st.Total == 4
	})
	if err := h.app.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	st, err := h.app.Stats(7)
	if err != nil {
		t.Fatal(err)
	}
	var causes []string
	for _, c := range st.ByCause {
		causes = append(causes, fmt.Sprintf("%s/%s/%d", c.Kind, c.Label, c.Count))
	}
	want := []string{"idle/Reply landed/1", "permission/Bash: upscope test api/1", "permission/Edit/1", "question/Question/1"}
	slices.Sort(causes)
	if !slices.Equal(causes, want) {
		t.Errorf("causes = %q, want %q", causes, want)
	}
	if st.Recent[len(st.Recent)-1].Issue != 3 || st.Recent[0].SessionTitle != "#3 work" || st.MedianWaitMs <= 0 || len(st.ByDay) != 7 || st.ByDay[6].Count != 4 {
		t.Errorf("stats = %+v", st)
	}
	for _, w := range st.Recent {
		if w.WaitedMs <= 0 {
			t.Errorf("wait left open after the session ended: %+v", w)
		}
	}
	if h.rec.count("stats") < 4 {
		t.Errorf("stats events = %d", h.rec.count("stats"))
	}
	if _, err := os.Stat(filepath.Join(h.state, "data", "main", "stats.jsonl")); err != nil {
		t.Errorf("stats file: %v", err)
	}
}

func TestStatsSummary(t *testing.T) {
	w := newWaits(t.TempDir())
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	at := func(daysAgo int) int64 { return now.AddDate(0, 0, -daysAgo).UnixMilli() }
	for i, wait := range []Wait{
		{Kind: "permission", Label: "Bash: yarn test", StartedAt: at(0), WaitedMs: 1000},
		{Kind: "permission", Label: "Bash: yarn test", StartedAt: at(0), WaitedMs: 3000},
		{Kind: "idle", Label: "Reply landed", StartedAt: at(2), WaitedMs: 2000},
		{Kind: "permission", Label: "Edit", StartedAt: at(2) + 1, WaitedMs: 10000},
		{Kind: "question", Label: "Question", StartedAt: at(30), WaitedMs: 500},
	} {
		wait.SessionTitle = fmt.Sprint("s", i)
		if err := w.append("p", wait); err != nil {
			t.Fatal(err)
		}
	}
	st, err := w.Stats("p", 7, now)
	if err != nil {
		t.Fatal(err)
	}
	if st.Total != 4 || st.TotalWaitMs != 16000 || st.MedianWaitMs != 2500 {
		t.Errorf("total %d, wait %d, median %d", st.Total, st.TotalWaitMs, st.MedianWaitMs)
	}
	if st.ByCause[0].Label != "Bash: yarn test" || st.ByCause[0].Count != 2 || st.ByCause[0].TotalWaitMs != 4000 || len(st.ByCause) != 3 {
		t.Errorf("byCause = %+v", st.ByCause)
	}
	var days []string
	for _, d := range st.ByDay {
		days = append(days, fmt.Sprintf("%s=%d", d.Day, d.Count))
	}
	if len(days) != 7 || days[0] != "2026-09-26=0" || days[4] != "2026-09-30=2" || days[6] != "2026-10-02=2" {
		t.Errorf("byDay = %v", days)
	}
	if st.Recent[0].StartedAt < st.Recent[len(st.Recent)-1].StartedAt || len(st.Recent) != 4 {
		t.Errorf("recent order = %+v", st.Recent)
	}
	for i := range 60 {
		_ = w.append("q", Wait{Kind: "idle", Label: "Reply landed", StartedAt: at(0) + int64(i), WaitedMs: 1})
	}
	if st, _ = w.Stats("q", 1, now); st.Total != 60 || len(st.Recent) != 50 || st.Recent[0].StartedAt != at(0)+59 {
		t.Errorf("recent cap: total %d, kept %d", st.Total, len(st.Recent))
	}
	if st, _ = w.Stats("none", 3, now); st.Total != 0 || len(st.ByDay) != 3 || st.ByCause == nil || st.Recent == nil {
		t.Errorf("empty stats = %+v", st)
	}
}

func TestCauseOf(t *testing.T) {
	tests := []struct {
		name        string
		rec         session.Record
		kind, label string
	}{
		{"bash command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash", Command: "yarn test api  --watch"}, "permission", "Bash: yarn test api"},
		{"short bash command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash", Command: "ls"}, "permission", "Bash: ls"},
		{"bash without command", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Bash"}, "permission", "Bash"},
		{"other tool", session.Record{State: session.Waiting, Notify: "permission_prompt", Tool: "Edit", Command: "ignored"}, "permission", "Edit"},
		{"unknown tool", session.Record{State: session.Waiting, Notify: "permission_prompt"}, "permission", "Permission"},
		{"untyped permission", session.Record{State: session.Waiting, Detail: "Claude needs your permission to use Edit", Tool: "Edit"}, "permission", "Edit"},
		{"elicitation", session.Record{State: session.Waiting, Notify: "elicitation_dialog", Tool: "Edit"}, "question", "Question"},
		{"reply landed", session.Record{State: session.Idle, Event: "Stop", Tool: "Edit"}, "idle", "Reply landed"},
	}
	for _, tt := range tests {
		if kind, label := causeOf(tt.rec); kind != tt.kind || label != tt.label {
			t.Errorf("%s: got %s/%q, want %s/%q", tt.name, kind, label, tt.kind, tt.label)
		}
	}
}
