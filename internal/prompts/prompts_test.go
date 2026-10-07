package prompts

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// The hashes pin the texts: changing one is deliberate, and updates its hash here.
func TestPromptsKeepTheirText(t *testing.T) {
	tests := []struct {
		name string
		text string
		hash string
	}{
		{"browser session", BrowserSession(), "9e17563dc37844a7c4b42ad45bc2190b089cf872b8a409e65300b8c15f553d28"},
		{"browser help", BrowserHelp(), "2c488dfb52eea50ed13383aa71bedb6a48dda552646ba9f5a7de423ff38a16f3"},
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
