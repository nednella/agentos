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
		{"browser session", BrowserSession(), "29ab13ce964154eb901819ae402b09898e4e439bb3ae1923d813d045c2a429fc"},
		{"session", Session(), "9a042ae2fbf137bb5cf1d696344d10d1b90162ef07d14767d1f1499b6db3a343"},
		{"browser help", BrowserHelp(), "e03aad399bcfbeaf312fdae39f01dcd3543a5ad7de4de72a02490ebdedaa2631"},
		{"setup help", SetupHelp(), "6dec5944af9e80b5c3dcdb18dafd66b4acd42e76ffefb76ae2d1e726426e05f3"},
		{"setup start", SetupStart(), "c8871af619bf6d2a82d2a23c2ac06f1a88b037ad130311c4377de22cf8135f1d"},
		{"setup review", SetupReview(), "6dc96647c3e24553864850e3a069dc7af5ad73d3cfaa0510c25952fd9f762ad7"},
		{"setup checks", SetupChecks(), "e9a02eb963d84ac6f5638782a59dff307d11ced02b795140dee356c475c347a7"},
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
