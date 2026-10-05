package sessions

import (
	"strings"
	"testing"
)

func TestTheHarnessPromptAsksForTheRightThings(t *testing.T) {
	for _, want := range []string{"agentos stats --days 30", "claude --help", "https://docs.claude.com/en/docs/claude-code", "at most seven", "agentos note", "Do not edit"} {
		if !strings.Contains(harnessPrompt, want) {
			t.Errorf("the prompt lacks %q", want)
		}
	}
}
