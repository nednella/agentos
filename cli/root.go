// Package cli is the agentos command line.
package cli

import (
	"context"

	"github.com/spf13/cobra"
)

const (
	groupWork    = "work"
	groupViews   = "views"
	groupProject = "project"
	groupSession = "session"
)

// Execute runs the agentos command line.
func Execute(ctx context.Context) error {
	return newRootCmd().ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "agentos",
		Short:         "One screen for all your coding agents",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddGroup(
		&cobra.Group{ID: groupWork, Title: "Work:"},
		&cobra.Group{ID: groupViews, Title: "Views:"},
		&cobra.Group{ID: groupProject, Title: "Projects:"},
		&cobra.Group{ID: groupSession, Title: "From inside a session:"},
	)
	root.AddCommand(newBrowserCmd(), newDigestCmd(), newHookCmd(), newKillCmd(), newNoteCmd(), newShowCmd(), newStatsCmd(), newVersionCmd())
	root.AddCommand(newAppCmds()...)
	return root
}
