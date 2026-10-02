// Package cmd is the agentos command line.
package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/term"
)

const defaultTmuxSocket = "agentos"

// Execute runs the agentos command line.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "agentos",
		Short:         "One screen for all your coding agents: this command opens the app",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return openApp(cmd.Context(), cmd.OutOrStdout())
		},
	}
	root.AddCommand(newHookCmd(), newGuardCmd(), newBrowserCmd(), newShowCmd(), newNoteCmd(), newStatsCmd(), newDigestCmd(), newKillCmd(), newVersionCmd())
	return root
}

// appCandidates lists where agentos.app may be, best first: the bundle the
// binary runs from, a bundle next to the binary, then the two Applications folders.
func appCandidates(exe, home string) []string {
	var out []string
	if i := strings.Index(exe, ".app/Contents/"); i >= 0 {
		out = append(out, exe[:i+len(".app")])
	}
	return append(out,
		filepath.Join(filepath.Dir(exe), "agentos.app"),
		filepath.Join(home, "Applications", "agentos.app"),
		"/Applications/agentos.app",
	)
}

func openApp(ctx context.Context, out io.Writer) error {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding the agentos binary: %w", err)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	home, _ := os.UserHomeDir()
	candidates := appCandidates(exe, home)
	for _, app := range candidates {
		if info, err := os.Stat(app); err == nil && info.IsDir() {
			if err := exec.CommandContext(ctx, "open", "-a", app).Run(); err != nil {
				return fmt.Errorf("opening %s: %w", app, err)
			}
			return nil
		}
	}
	fmt.Fprintln(out, "agentos.app not found. Looked in:")
	for _, c := range candidates {
		fmt.Fprintln(out, "  "+c)
	}
	return errors.New("build it with make desktop-app, or install it with make desktop-install")
}

// env is what the commands that touch sessions need.
type env struct {
	project  project.Project
	stateDir string
	tmux     *term.Tmux
}

func loadEnv() (env, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return env{}, fmt.Errorf("finding the current folder: %w", err)
	}
	path := os.Getenv("AGENTOS_CONFIG")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return env{}, fmt.Errorf("finding home dir: %w", err)
		}
		path = filepath.Join(home, ".config", "agentos", "config.yaml")
	}
	cfg, err := project.Load(path)
	if err != nil {
		return env{}, fmt.Errorf("loading config: %w", err)
	}
	stateDir, err := bus.DefaultDir()
	if err != nil {
		return env{}, err
	}
	socket := os.Getenv("AGENTOS_TMUX_SOCKET")
	if socket == "" {
		socket = defaultTmuxSocket
	}
	tmux, err := term.NewTmux(socket, stateDir)
	if err != nil {
		return env{}, err
	}
	return env{project: cfg.Resolve(cwd), stateDir: stateDir, tmux: tmux}, nil
}
