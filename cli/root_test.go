package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "agentos dev\n"; got != want {
		t.Errorf("version printed %q, want %q", got, want)
	}
}

func TestRootHelpListsTheCommands(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd()
	root.SetOut(&out)
	root.SetArgs([]string{"--help"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "version") || !strings.Contains(out.String(), "opens the app") {
		t.Errorf("help does not list the version command or say what no command does:\n%s", out.String())
	}
}
