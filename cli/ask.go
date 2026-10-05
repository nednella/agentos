package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/control"
)

// The commands that talk to the running app share these helpers: each sends one
// request from its session and prints what the app answers.

func controlSocket() (string, error) {
	if socket := os.Getenv("AGENTOS_SOCKET"); socket != "" {
		return control.SocketPath(bus.DirOf(socket)), nil
	}
	dir, err := bus.DefaultDir()
	if err != nil {
		return "", err
	}
	return control.SocketPath(dir), nil
}

// askApp sends req from this session and prints the answer.
func askApp(cmd *cobra.Command, req control.Request) error {
	socket, err := controlSocket()
	if err != nil {
		return err
	}
	req.Session = os.Getenv("AGENTOS_SESSION")
	req.Project = firstSet(os.Getenv("AGENTOS_DIGEST_PROJECT"), os.Getenv("AGENTOS_PROJECT"))
	resp, err := control.Call(cmd.Context(), socket, req)
	if err != nil {
		return err
	}
	if !resp.OK {
		return errors.New(resp.Error)
	}
	if resp.Out != "" {
		fmt.Fprintln(cmd.OutOrStdout(), strings.TrimRight(resp.Out, "\n"))
	}
	return nil
}

func firstSet(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// parseFlags splits args into positional words and flags. Flags in values take
// a value; flags in switches do not. A lone -- ends the flags.
func parseFlags(args []string, values, switches []string) ([]string, map[string]string, error) {
	var pos []string
	opts := map[string]string{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		name, ok := strings.CutPrefix(a, "--")
		if !ok || a == "--" {
			pos = append(pos, a)
			continue
		}
		name, val, hasVal := strings.Cut(name, "=")
		switch {
		case contains(switches, name):
			opts[name] = "1"
		case contains(values, name):
			if !hasVal {
				if i+1 >= len(args) {
					return nil, nil, fmt.Errorf("--%s needs a value", name)
				}
				i++
				val = args[i]
			}
			opts[name] = val
		default:
			return nil, nil, fmt.Errorf("unknown flag --%s", name)
		}
	}
	return pos, opts, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
