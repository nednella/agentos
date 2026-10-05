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

func TestDigestNamesTheDependencies(t *testing.T) {
	got := Digest([]string{"react", "gopkg.in/yaml.v3"})
	for _, want := range []string{"react\ngopkg.in/yaml.v3\n", "agentos digest add --title", "Do nothing else."} {
		if !strings.Contains(got, want) {
			t.Errorf("the digest prompt lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "{dependencies}") || strings.Contains(got, "<!--") {
		t.Errorf("the digest prompt keeps its framing:\n%s", got)
	}
	if strings.Contains(Digest(nil), "{dependencies}") {
		t.Error("an empty list left the placeholder")
	}
}

func TestPRPromptsNameTheNumber(t *testing.T) {
	for _, got := range []string{PRReview(41), PRChecks(41)} {
		if !strings.Contains(got, "PR #41") || strings.Contains(got, "{n}") || strings.Contains(got, "<!--") || strings.HasSuffix(got, "\n") {
			t.Errorf("prompt = %q", got)
		}
	}
}
