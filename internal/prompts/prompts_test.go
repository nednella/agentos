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
		{"new session", NewSession(), "a738a01d16332bb1c28d6c5a83b38a95525c5685ad226a59216e95e86d9acfb3"},
		{"browser help", BrowserHelp(), "e03aad399bcfbeaf312fdae39f01dcd3543a5ad7de4de72a02490ebdedaa2631"},
		{"setup session", SetupSession(), "341ba267d4dd5a2f3f5fa1c0036a9426c432ad11b7136de7d9c8b05969c2db79"},
		{"setup help", SetupHelp(), "8930430e572628f949576b68d7a9056dd27f09c32a42f45b6de870b8b9d653ab"},
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
