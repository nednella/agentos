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
		{"browser", Browser(), "89a9746065d583edb60e70a281c1560f0aa15b66df48737e59269af8def02fbb"},
		{"session", Session(), "288ea5f16bbda82e35a1bbe2f9fae91166accf07de5b1455930468e26e5582b4"},
		{"task", Task(), "5181694ae5b63faba593eade34cb2f8e912ea7b0444ade07b08183da5c2373f2"},
		{"chat", Chat(), "3a4ef510420450b4c67d88375811ae17a111621f89a5dcb2b405e33895b9c4e9"},
		{"browser help", BrowserHelp(), "d84f5a28ba26ad87c9290bb7cdd0e6d0f0bc593666b9f90cbcdea54f9cd7aefa"},
		{"setup brief", SetupBrief(), "eb691cd78708a321a783e4282be33e48ec7adbdfd71aa385bb755697ba745d37"},
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

func TestHandoverEndsWithTheContext(t *testing.T) {
	if got := Handover("line one\n{context}"); !strings.HasSuffix(got, ":\n\nline one\n{context}") || strings.Contains(got, "<!--") {
		t.Errorf("Handover = %q", got)
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
