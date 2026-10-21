package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestAgentProgramsHoldEveryProviderAndTheConfiguredAgent(t *testing.T) {
	conf := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(conf, []byte("agent_command: my-agent --fast\nprojects: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_CONFIG", conf)
	if got := agentPrograms(); !slices.Equal(got, []string{"claude", "agy", "my-agent"}) {
		t.Errorf("programs = %q", got)
	}
}

func TestFixLocale(t *testing.T) {
	t.Setenv("LANG", "")
	t.Setenv("LC_ALL", "")
	fixLocale()
	if got := os.Getenv("LANG"); got != "en_US.UTF-8" {
		t.Errorf("LANG = %q", got)
	}
	t.Setenv("LANG", "de_DE.UTF-8")
	fixLocale()
	if got := os.Getenv("LANG"); got != "de_DE.UTF-8" {
		t.Errorf("an existing locale was replaced by %q", got)
	}
}
