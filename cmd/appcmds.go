package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/control"
)

const (
	groupWork    = "work"
	groupViews   = "views"
	groupProject = "project"
	groupSession = "session"
)

// appCmd is a command that asks the running app to do something and prints its answer.
func appCmd(use, short, group string, args cobra.PositionalArgs) *cobra.Command {
	name, _, _ := strings.Cut(use, " ")
	return &cobra.Command{
		Use: use, Short: short, GroupID: group, Args: args,
		RunE: func(cmd *cobra.Command, args []string) error {
			return askApp(cmd, control.Request{Cmd: name, Args: args})
		},
	}
}

func newAppCmds() []*cobra.Command {
	cmds := []*cobra.Command{
		appCmd("issue <number...>", "Start a session for each issue", groupWork, cobra.MinimumNArgs(1)),
		appCmd("new [title]", "Start a session", groupWork, nil),
		appCmd("open <n|title>", "Show a session in the terminal", groupWork, cobra.MinimumNArgs(1)),
		appCmd("next", "Show the session that needs you most", groupWork, cobra.NoArgs),
		appCmd("refresh", "Reload the issues and the pull requests", groupWork, cobra.NoArgs),
		appCmd("pr [n]", "Show the pull requests of the sessions", groupWork, cobra.MaximumNArgs(1)),
		appCmd("cleanup [n]", "Clean up after session n, or list what waits for it", groupWork, cobra.MaximumNArgs(1)),
		appCmd("harness", "Start a session that reviews the project's harness", groupWork, cobra.NoArgs),
		appCmd("queue", "Show the queue", groupViews, cobra.NoArgs),
		appCmd("notes", "Show the notes", groupViews, cobra.NoArgs),
		appCmd("evidence", "Show the evidence of the session on screen", groupViews, cobra.NoArgs),
		appCmd("term", "Show the terminal", groupViews, cobra.NoArgs),
		appCmd("filter [query]", "Filter the queue, for example: filter @me type:bug", groupViews, nil),
	}
	project := appCmd("project [name]", "List projects, or switch to one", groupProject, nil)
	project.RunE = func(cmd *cobra.Command, args []string) error {
		return askApp(cmd, control.Request{Cmd: "project", Args: args})
	}
	project.AddCommand(&cobra.Command{
		Use: "add [path]", Short: "Add a folder as a project (the current folder by default)", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("finding %s: %w", path, err)
			}
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("%s: %w", abs, err)
			}
			return askApp(cmd, control.Request{Cmd: "project", Args: []string{"add", abs}})
		},
	}, &cobra.Command{
		Use: "remove <name>", Short: "Forget a project", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return askApp(cmd, control.Request{Cmd: "project", Args: []string{"remove", args[0]}})
		},
	})
	return append(cmds, project)
}
