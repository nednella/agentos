package session

import (
	"strings"
	"testing"
	"time"
)

func TestNext(t *testing.T) {
	tests := []struct {
		name string
		cur  State
		ev   Event
		want State
	}{
		{"start", Idle, Event{Name: "SessionStart"}, Idle},
		{"start after a reply", Idle, Event{Name: "SessionStart"}, Idle},
		{"prompt", Idle, Event{Name: "UserPromptSubmit"}, Working},
		{"pre tool", Idle, Event{Name: "PreToolUse"}, Working},
		{"post tool answers a prompt", Waiting, Event{Name: "PostToolUse"}, Working},
		{"permission prompt", Working, Event{Name: "Notification", NotificationType: "permission_prompt"}, Waiting},
		{"notification without type", Working, Event{Name: "Notification", Message: "Claude needs your permission to use Bash"}, Waiting},
		{"idle reminder stops a working session", Working, Event{Name: "Notification", NotificationType: "idle_prompt"}, Idle},
		{"idle reminder by message stops a working session", Working, Event{Name: "Notification", Message: "Claude is waiting for your input"}, Idle},
		{"idle reminder leaves waiting alone", Waiting, Event{Name: "Notification", NotificationType: "idle_prompt"}, Waiting},
		{"idle reminder leaves idle alone", Idle, Event{Name: "Notification", NotificationType: "idle_prompt"}, Idle},
		{"idle reminder leaves ended alone", Ended, Event{Name: "Notification", NotificationType: "idle_prompt"}, Ended},
		{"stop", Working, Event{Name: "Stop"}, Idle},
		{"stop while waiting", Waiting, Event{Name: "Stop"}, Idle},
		{"end", Working, Event{Name: "SessionEnd"}, Ended},
		{"unknown event", Working, Event{Name: "PreCompact"}, Working},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Next(tt.cur, tt.ev); got != tt.want {
				t.Errorf("Next(%s, %s) = %s, want %s", tt.cur, tt.ev.Name, got, tt.want)
			}
		})
	}
}

func TestApply(t *testing.T) {
	t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Minute)
	stop := Record{Session: "p/1", State: Idle, Event: "Stop", At: t0, Detail: "done"}
	tests := []struct {
		name string
		prev Record
		ev   Event
		want Record
	}{
		{"first event", Record{}, Event{Name: "PreToolUse", Detail: "Bash ls"}, Record{Session: "p/1", State: Working, Event: "PreToolUse", At: t1, Detail: "Bash ls"}},
		{"idle reminder ends a working turn", Record{Session: "p/1", State: Working, Event: "PreToolUse", At: t0, Tool: "Bash"}, Event{Name: "Notification", NotificationType: "idle_prompt"}, Record{Session: "p/1", State: Idle, Event: "Notification", At: t1, Notify: "idle_prompt", Tool: "Bash"}},
		{"idle reminder leaves a waiting record alone", Record{Session: "p/1", State: Waiting, Event: "Notification", At: t0}, Event{Name: "Notification", NotificationType: "idle_prompt"}, Record{Session: "p/1", State: Waiting, Event: "Notification", At: t0}},
		{"idle reminder keeps the record", stop, Event{Name: "Notification", NotificationType: "idle_prompt", Detail: "waiting"}, stop},
		{"permission prompt", stop, Event{Name: "Notification", NotificationType: "permission_prompt", Detail: "needs Bash"}, Record{Session: "p/1", State: Waiting, Event: "Notification", At: t1, Detail: "needs Bash", Notify: "permission_prompt"}},
		{"prompt keeps the last tool", Record{Session: "p/1", State: Working, Event: "PreToolUse", Tool: "Bash", Command: "yarn test"}, Event{Name: "Notification", NotificationType: "permission_prompt"}, Record{Session: "p/1", State: Waiting, Event: "Notification", At: t1, Notify: "permission_prompt", Tool: "Bash", Command: "yarn test"}},
		{"new prompt forgets the tool", Record{Session: "p/1", State: Working, Event: "PreToolUse", Tool: "Bash", Command: "yarn test"}, Event{Name: "UserPromptSubmit"}, Record{Session: "p/1", State: Working, Event: "UserPromptSubmit", At: t1}},
		{"reminder with no record", Record{}, Event{Name: "Notification", NotificationType: "idle_prompt", Detail: "x"}, Record{Session: "p/1", State: Idle, Event: "Notification", At: t1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Apply("p/1", tt.prev, tt.ev, t1); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseEvent(t *testing.T) {
	tests := []struct {
		name  string
		event string
		body  string
		want  Event
	}{
		{"permission notification", "Notification", `{"message":"Claude needs your permission to use Bash","notification_type":"permission_prompt"}`,
			Event{Name: "Notification", NotificationType: "permission_prompt", Message: "Claude needs your permission to use Bash", Detail: "Claude needs your permission to use Bash"}},
		{"tool with command", "PreToolUse", `{"tool_name":"Bash","tool_input":{"command":"echo hi","description":"Print hi"}}`,
			Event{Name: "PreToolUse", Detail: "Bash echo hi", Tool: "Bash", Command: "echo hi"}},
		{"tool without known input", "PostToolUse", `{"tool_name":"Task","tool_input":{"x":1}}`,
			Event{Name: "PostToolUse", Detail: "Task", Tool: "Task"}},
		{"prompt on many lines", "UserPromptSubmit", `{"prompt":"fix\n  the   bug"}`,
			Event{Name: "UserPromptSubmit", Detail: "fix the bug"}},
		{"not json", "Stop", `garbage`, Event{Name: "Stop"}},
		{"empty", "SessionStart", ``, Event{Name: "SessionStart"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseEvent(tt.event, strings.NewReader(tt.body)); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestSort(t *testing.T) {
	t0 := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	mk := func(n int, s State, ago time.Duration) Session {
		return Session{Name: Name{"p", n}, State: s, At: t0.Add(-ago)}
	}
	ss := []Session{
		mk(1, Idle, 0),
		mk(2, Working, time.Minute),
		mk(3, Idle, time.Hour),
		mk(4, Waiting, time.Hour),
		mk(5, Working, time.Second),
		mk(6, Idle, time.Minute),
		mk(7, Idle, 0),
		mk(8, Ended, 0),
	}
	Sort(ss)
	var got []int
	for _, s := range ss {
		got = append(got, s.Name.N)
	}
	want := []int{4, 1, 7, 6, 3, 5, 2, 8}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		in      string
		want    Name
		wantErr bool
	}{
		{"api/3", Name{"api", 3}, false},
		{"my-app/12", Name{"my-app", 12}, false},
		{"a/b/7", Name{"a/b", 7}, false},
		{"api", Name{}, true},
		{"/3", Name{}, true},
		{"api/", Name{}, true},
		{"api/x", Name{}, true},
		{"api/0", Name{}, true},
		{"api/shell", Name{Project: "api"}, false},
		{"a/b/shell", Name{Project: "a/b"}, false},
		{"api/shells", Name{}, true},
		{"/shell", Name{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := ParseName(tt.in)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("ParseName(%q) = %+v, %v", tt.in, got, err)
			}
			if err == nil && got.String() != tt.in {
				t.Errorf("round trip = %q, want %q", got.String(), tt.in)
			}
			if err == nil && got.IsShell() != strings.HasSuffix(tt.in, "/shell") {
				t.Errorf("IsShell = %v for %q", got.IsShell(), tt.in)
			}
		})
	}
}

func TestNextN(t *testing.T) {
	tests := []struct {
		used []int
		want int
	}{{nil, 1}, {[]int{1, 2}, 3}, {[]int{2, 3}, 1}, {[]int{1, 3}, 2}}
	for _, tt := range tests {
		if got := NextN(tt.used); got != tt.want {
			t.Errorf("NextN(%v) = %d, want %d", tt.used, got, tt.want)
		}
	}
}
