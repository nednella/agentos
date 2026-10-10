package stats_test

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestWaitsAreRecorded(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("#3 work", "", false, 3)
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"make test api foo/bar.test.ts"}}`)
	h.Hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Bash","notification_type":"permission_prompt"}`)
	eventually(t, "open wait", func() bool {
		st, _ := h.Stats(7)
		return st.Total == 1 && st.Recent[0].WaitedMs == 0
	})
	time.Sleep(20 * time.Millisecond)
	h.Hook(t, s.ID, "PreToolUse", `{"tool_name":"Edit","tool_input":{"file_path":"a.ts"}}`)
	h.Hook(t, s.ID, "Notification", `{"message":"waiting for your input","notification_type":"idle_prompt"}`)
	h.Hook(t, s.ID, "Notification", `{"message":"Claude needs your permission to use Edit","notification_type":"permission_prompt"}`)
	h.Hook(t, s.ID, "PostToolUse", `{"tool_name":"Edit"}`)
	h.Hook(t, s.ID, "Notification", `{"message":"pick one","notification_type":"elicitation_dialog"}`)
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"b"}`)
	h.Hook(t, s.ID, "Stop", `{}`)
	eventually(t, "finished wait", func() bool {
		st, _ := h.Stats(7)
		return st.Total == 4
	})
	if err := h.KillSession(s.ID); err != nil {
		t.Fatal(err)
	}
	st, err := h.Stats(7)
	if err != nil {
		t.Fatal(err)
	}
	var causes []string
	for _, c := range st.ByCause {
		causes = append(causes, fmt.Sprintf("%s/%s/%d", c.Kind, c.Label, c.Count))
	}
	want := []string{"idle/Reply landed/1", "permission/Bash: make test api/1", "permission/Edit/1", "question/Question/1"}
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
	if h.Rec.Count("stats") < 4 {
		t.Errorf("stats events = %d", h.Rec.Count("stats"))
	}
	if entries, err := os.ReadDir(filepath.Join(h.State, "data", "main", "events")); err != nil || len(entries) == 0 {
		t.Errorf("events folder: %v, %v", entries, err)
	}
}
