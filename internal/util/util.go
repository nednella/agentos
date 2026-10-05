// Package util holds the few helpers several packages share.
package util

import (
	"strings"
	"time"
)

// ShellQuote quotes s as one word for a POSIX shell.
func ShellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// FirstLine is the first line of s, trimmed.
func FirstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// Millis is t in unix milliseconds, 0 for the zero time.
func Millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
}
