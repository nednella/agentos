package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func userLine(at time.Time, content string) string {
	return fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"role":"user","content":%s}}`+"\n", at.UTC().Format(time.RFC3339Nano), content)
}

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "conv.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "")), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestReadTurn(t *testing.T) {
	t0 := time.Now().Add(-time.Minute).Truncate(time.Millisecond)
	t1 := t0.Add(time.Second)
	prompt := userLine(t0, `"write an essay"`)
	reply := `{"type":"assistant","timestamp":"` + t1.UTC().Format(time.RFC3339Nano) + `","message":{"content":[]}}` + "\n"
	tests := []struct {
		name  string
		lines []string
		since time.Time
		want  Turn
	}{
		{"interrupted while streaming", []string{prompt, reply, userLine(t1, `[{"type":"text","text":"[Request interrupted by user]"}]`), `{"type":"last-prompt"}` + "\n"}, t0, Turn{Interrupted: t1, Replied: true}},
		{"interrupted during a tool", []string{prompt, reply, userLine(t1, `[{"type":"tool_result","content":"rejected"}]`), userLine(t1, `[{"type":"text","text":"[Request interrupted by user for tool use]"}]`)}, t0, Turn{Interrupted: t1, Replied: true}},
		{"marker as a string", []string{prompt, userLine(t1, `"[Request interrupted by user]"`)}, t0, Turn{Interrupted: t1}},
		{"interrupt before the given time", []string{prompt, userLine(t1, `"[Request interrupted by user]"`)}, t1, Turn{}},
		{"turn still going", []string{prompt, reply}, t0, Turn{Replied: true}},
		{"no reply yet", []string{prompt}, t0.Add(-time.Second), Turn{}},
		{"prompt quoting the marker", []string{userLine(t1, `"why does it say [Request interrupted by user]?"`)}, t0, Turn{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ReadTurn(writeTranscript(t, tt.lines...), tt.since); !got.Interrupted.Equal(tt.want.Interrupted) || got.Replied != tt.want.Replied {
				t.Errorf("ReadTurn = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadTurnReadsOnlyTheEnd(t *testing.T) {
	t0 := time.Now().Add(-time.Minute)
	big := userLine(t0, fmt.Sprintf("%q", strings.Repeat("x", 2*turnTail)))
	path := writeTranscript(t, big, userLine(t0.Add(time.Second), `"[Request interrupted by user]"`))
	if ReadTurn(path, t0).Interrupted.IsZero() {
		t.Error("missed the interrupt after a line longer than the tail")
	}
}

func TestReadTurnWithoutATranscript(t *testing.T) {
	if got := ReadTurn(filepath.Join(t.TempDir(), "missing.jsonl"), time.Time{}); got != (Turn{}) {
		t.Errorf("ReadTurn = %+v with no transcript", got)
	}
}
