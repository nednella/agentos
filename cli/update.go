package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/update"
	"github.com/nednella/agentos/internal/version"
)

const (
	noticeEvery  = 24 * time.Hour
	noticeBudget = 3 * time.Second
)

// runCommand is the update.Runner that runs the real curl and ditto.
func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, name, args...).Output()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		msg, _, _ := strings.Cut(strings.TrimSpace(string(exit.Stderr)), "\n")
		return nil, fmt.Errorf("%s: %w: %s", name, err, msg)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

func newUpdateCmd(l launcher) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Install the latest release",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, l.places(), runCommand, version.Version)
		},
	}
}

// runUpdate installs the latest release over the bundle the command finds, then asks a
// running app to start again from it.
func runUpdate(cmd *cobra.Command, places []string, run update.Runner, current string) error {
	out := cmd.OutOrStdout()
	bundle, err := findApp(places)
	if err != nil {
		return err
	}
	rel, err := update.Latest(cmd.Context(), run)
	if err != nil {
		return err
	}
	if !update.Newer(current, rel.Version) {
		if current == "dev" {
			fmt.Fprintf(out, "this is a dev build; the latest release is v%s\n", rel.Version)
		} else {
			fmt.Fprintf(out, "agentos v%s is the latest release\n", current)
		}
		return nil
	}
	fmt.Fprintf(out, "downloading v%s\n", rel.Version)
	if err := update.Install(cmd.Context(), run, rel, bundle); err != nil {
		return err
	}
	fmt.Fprintf(out, "updated %s to v%s\n", bundle, rel.Version)
	switch err := relaunchApp(cmd.Context()); {
	case errors.Is(err, control.ErrNotRunning):
		fmt.Fprintln(out, "run agentos to open it")
	case err != nil:
		return fmt.Errorf("the app is still on the old version: %w", err)
	default:
		fmt.Fprintln(out, "the app is starting again")
	}
	return nil
}

func relaunchApp(ctx context.Context) error {
	socket, err := controlSocket()
	if err != nil {
		return err
	}
	resp, err := control.Call(ctx, socket, control.Request{Cmd: "relaunch"})
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	return nil
}

// notice tells a person at a terminal that a newer release is out, asking GitHub at most once
// a day. Agents and hooks, which have no terminal, never see it and never wait for it.
func notice(cmd *cobra.Command, l launcher) {
	if cmd.Name() == "update" || version.Version == "dev" || !l.attached() {
		return
	}
	stateDir, err := bus.DefaultDir()
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(cmd.Context(), noticeBudget)
	defer cancel()
	printNotice(ctx, cmd.ErrOrStderr(), runCommand, stateDir, version.Version, time.Now())
}

func printNotice(ctx context.Context, w io.Writer, run update.Runner, stateDir, current string, now time.Time) {
	rel, err := update.Check(ctx, run, stateDir, noticeEvery, now)
	if err != nil || !update.Newer(current, rel.Version) {
		return
	}
	fmt.Fprintf(w, "agentos v%s is out: run agentos update\n", rel.Version)
}
