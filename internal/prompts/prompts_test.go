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
		{"browser session", BrowserSession(), "89a9746065d583edb60e70a281c1560f0aa15b66df48737e59269af8def02fbb"},
		{"session", Session(), "9a042ae2fbf137bb5cf1d696344d10d1b90162ef07d14767d1f1499b6db3a343"},
		{"browser help", BrowserHelp(), "8553189d3bd018d1fda10d831b3185525108810e3f333fc0b0d57fe02a8645b7"},
		{"setup brief", SetupBrief(), "c42e4a9dc47b6ec3859228e9cf668c889f86240e5b50ad81bc68c02a52fdd1a8"},
		{"setup start", SetupStart(), "cf5d8bdf29eda3fed3e8a7dd0fc7c5eb2129b3f810f26cf0df0da908a13e4606"},
		{"setup review", SetupReview(), "6dc96647c3e24553864850e3a069dc7af5ad73d3cfaa0510c25952fd9f762ad7"},
		{"setup checks", SetupChecks(), "e9a02eb963d84ac6f5638782a59dff307d11ced02b795140dee356c475c347a7"},
		{"setup conflict", SetupConflict(), "5622f11bb34997100d3673942a0827aa104afc57290f6555416d3d7fad3b04f9"},
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
