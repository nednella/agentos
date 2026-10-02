package cmd

import (
	"slices"
	"testing"
)

func TestAppCandidates(t *testing.T) {
	tests := []struct {
		name string
		exe  string
		want []string
	}{
		{"inside a bundle", "/Users/n/Applications/agentos.app/Contents/MacOS/agentos", []string{
			"/Users/n/Applications/agentos.app", "/Users/n/Applications/agentos.app/Contents/MacOS/agentos.app", "/Users/n/Applications/agentos.app", "/Applications/agentos.app"}},
		{"plain binary", "/Users/n/.local/bin/agentos", []string{
			"/Users/n/.local/bin/agentos.app", "/Users/n/Applications/agentos.app", "/Applications/agentos.app"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appCandidates(tt.exe, "/Users/n"); !slices.Equal(got, tt.want) {
				t.Errorf("got %q\nwant %q", got, tt.want)
			}
		})
	}
}
