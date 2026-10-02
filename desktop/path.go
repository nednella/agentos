package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const shellPathBudget = 3 * time.Second

// fixPath adopts the login shell's PATH when the tools we run are missing: an app
// started from Finder gets a minimal one.
func fixPath() error {
	missing := false
	for _, tool := range []string{"tmux", "claude", "gh", "agentos"} {
		if _, err := exec.LookPath(tool); err != nil {
			missing = true
		}
	}
	if !missing {
		return nil
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), shellPathBudget)
	defer cancel()
	out, err := exec.CommandContext(ctx, shell, "-lic", "echo -n $PATH").Output()
	if err != nil {
		return fmt.Errorf("reading PATH from %s: %w", shell, err)
	}
	// Interactive shells may print a banner first; the PATH is the last line.
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if path := strings.TrimSpace(lines[len(lines)-1]); path != "" {
		return os.Setenv("PATH", path)
	}
	return nil
}

// fixLocale gives an app started from Finder a UTF-8 locale, so the agents' shells handle non-ASCII text.
func fixLocale() {
	if os.Getenv("LC_ALL") == "" && os.Getenv("LANG") == "" {
		_ = os.Setenv("LANG", "en_US.UTF-8")
	}
}
