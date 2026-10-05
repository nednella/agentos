package util

import (
	"testing"
	"time"
)

func TestShellQuote(t *testing.T) {
	for in, want := range map[string]string{"plain": `'plain'`, "it's": `'it'\''s'`, "": `''`, "a b;c": `'a b;c'`} {
		if got := ShellQuote(in); got != want {
			t.Errorf("ShellQuote(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestFirstLine(t *testing.T) {
	for in, want := range map[string]string{"one\ntwo": "one", "  padded \n": "padded", "": "", "\n\nlate\nx": "late"} {
		if got := FirstLine(in); got != want {
			t.Errorf("FirstLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMillis(t *testing.T) {
	at := time.UnixMilli(1_700_000_000_123)
	if Millis(time.Time{}) != 0 || Millis(at) != 1_700_000_000_123 {
		t.Error("Millis gave a wrong answer")
	}
}
