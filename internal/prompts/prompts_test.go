package prompts

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// The hashes are those of the texts as they stood in Go source before they moved here.
func TestPromptsKeepTheirText(t *testing.T) {
	tests := []struct {
		name string
		text string
		hash string
	}{
		{"harness check", HarnessCheck(), "8337a3069da3450c56372a6047babb19e6395e7a6f5a830c1aae565fbb96fd0b"},
		{"digest", Digest(), "d4b5dd6e11d8dd7aa0c2ce9a9c9944bcf96e3499bbc9c1b8b9ccb60b0530b6e5"},
		{"browser session", BrowserSession(), "479d542f0a6920f861bf8db587040d05b7999c1421491e52640b3a23f3e20312"},
		{"browser help", BrowserHelp(), "5ce9b1d5055ded6f64387636a5fa93a9c88fb048cc99b0eb74729ccad64d41e4"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fmt.Sprintf("%x", sha256.Sum256([]byte(tt.text))); got != tt.hash {
				t.Errorf("text changed: hash %s, want %s", got, tt.hash)
			}
			if strings.HasPrefix(tt.text, "<!--") || strings.HasSuffix(tt.text, "\n") {
				t.Errorf("text keeps its file framing: %q", tt.text)
			}
		})
	}
}

func TestBodyWithoutComment(t *testing.T) {
	tests := []struct{ in, want string }{
		{"<!-- tokens -->\n\ntext\n", "text"},
		{"text\n", "text"},
		{"text", "text"},
		{"<!-- a -->\n\nline one\n\nline two\n", "line one\n\nline two"},
	}
	for _, tt := range tests {
		if got := body(tt.in); got != tt.want {
			t.Errorf("body(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
