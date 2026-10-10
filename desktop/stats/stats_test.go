package stats

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/session"
)

func TestStatsSummary(t *testing.T) {
	dir := t.TempDir()
	w := New(NewLog(dir))
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
		writeWait(t, dir, "p", wait)
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
		writeWait(t, dir, "q", Wait{Kind: "idle", Label: "Reply landed", StartedAt: at(0) + int64(i), WaitedMs: 1})
	}
	if st, _ = w.Stats("q", 1, now); st.Total != 60 || len(st.Recent) != 50 || st.Recent[0].StartedAt != at(0)+59 {
		t.Errorf("recent cap: total %d, kept %d", st.Total, len(st.Recent))
	}
	if st, _ = w.Stats("none", 3, now); st.Total != 0 || len(st.ByDay) != 3 || st.ByCause == nil || st.Recent == nil {
		t.Errorf("empty stats = %+v", st)
	}
}

// writeWait puts the wait in the event file of the day it ended.
func writeWait(t *testing.T, dir, key string, wait Wait) {
	t.Helper()
	e := activity.Event{At: wait.StartedAt + wait.WaitedMs, Kind: activity.Wait, Ms: wait.WaitedMs, Label: wait.Kind, Cause: wait.Label, Title: wait.SessionTitle}
	line, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := appendLine(filepath.Join(dir, key, eventsFolder, localDay(time.UnixMilli(e.At))+".jsonl"), line); err != nil {
		t.Fatal(err)
	}
}

func TestStatsLabelsAnEventWithoutCauseByItsKind(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)
	writeWait(t, dir, "p", Wait{Kind: "idle", StartedAt: now.UnixMilli(), WaitedMs: 5})
	st, err := New(NewLog(dir)).Stats("p", 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(st.ByCause) != 1 || st.ByCause[0].Kind != "idle" || st.ByCause[0].Label != "idle" {
		t.Errorf("byCause = %+v", st.ByCause)
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
		if kind, label := CauseOf(tt.rec); kind != tt.kind || label != tt.label {
			t.Errorf("%s: got %s/%q, want %s/%q", tt.name, kind, label, tt.kind, tt.label)
		}
	}
}

func TestWaitIsRecordedAsAnEvent(t *testing.T) {
	dir := t.TempDir()
	log := NewLog(dir)
	w := New(log)
	start := time.Now().Add(-time.Second)
	w.Begin("p/aaaaaaaa", Wait{SessionTitle: "#3 work", Issue: 3, Model: "opus", Kind: "permission", Label: "Edit", StartedAt: start.UnixMilli()})
	w.End("p/aaaaaaaa", start.Add(1500*time.Millisecond))
	w.End("p/aaaaaaaa", time.Now())
	data, err := os.ReadFile(filepath.Join(dir, "p", "events", time.Now().Format(time.DateOnly)+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`{"at":%d,"kind":"wait","session":"p/aaaaaaaa","issue":3,"model":"opus","ms":1500,"label":"permission","title":"#3 work","cause":"Edit"}`+"\n", start.Add(1500*time.Millisecond).UnixMilli())
	if string(data) != want {
		t.Errorf("events = %s, want %s", data, want)
	}
}
