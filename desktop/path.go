package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/project"
)

const shellPathBudget = 3 * time.Second

// fixPath adopts the login shell's PATH when the tools we run are missing: an app
// started from Finder gets a minimal one.
func fixPath() error {
	missing := false
	for _, tool := range append([]string{"tmux", "gh", "agentos"}, agentPrograms()...) {
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

// agentPrograms are the commands of every provider and of the configured agent.
func agentPrograms() []string {
	names := slices.Clone(agent.Names)
	if home, err := os.UserHomeDir(); err == nil {
		if cfg, err := project.Load(app.ConfigPath(home)); err == nil {
			names = append(names, cfg.Agent)
		}
	}
	programs := make([]string, 0, len(names))
	for _, name := range names {
		programs = append(programs, agent.Program(agent.New(name, "")))
	}
	return programs
}

// fixLocale gives an app started from Finder a UTF-8 locale, so the agents' shells handle non-ASCII text.
func fixLocale() {
	if os.Getenv("LC_ALL") == "" && os.Getenv("LANG") == "" {
		_ = os.Setenv("LANG", "en_US.UTF-8")
	}
}
