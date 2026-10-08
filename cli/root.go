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

// launcher is how the bare command reaches the app.
type launcher struct {
	open     opener          // shows the bundle at a path, as Finder would
	places   func() []string // where the bundle may be, in the order to look
	window   func() error    // runs the app in this process; nil when this build has no window
	attached func() bool     // whether a person typed the command at a terminal
}

// Execute runs the agentos command line. window runs the app in this process: the
// app's binary is the command, and starts as the app when nothing typed it.
func Execute(ctx context.Context, window func() error) error {
	return newRootCmdWith(launcher{open: openWithMac, places: appPlaces, window: window, attached: stdinIsTerminal}).ExecuteContext(ctx)
}

func newRootCmd() *cobra.Command {
	return newRootCmdWith(launcher{open: openWithMac, places: appPlaces, attached: stdinIsTerminal})
}

func newRootCmdWith(l launcher) *cobra.Command {
	root := &cobra.Command{
		Use:               "agentos",
		Short:             "One screen for all your coding agents",
		Long:              "One screen for all your coding agents.\n\nWith no command, agentos opens the app.",
		Args:              cobra.NoArgs,
		RunE:              openApp(l),
		PersistentPostRun: func(cmd *cobra.Command, _ []string) { notice(cmd, l) },
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddGroup(
		&cobra.Group{ID: groupWork, Title: "Work:"},
		&cobra.Group{ID: groupViews, Title: "Views:"},
		&cobra.Group{ID: groupProject, Title: "Projects:"},
		&cobra.Group{ID: groupSession, Title: "From inside a session:"},
	)
	root.AddCommand(newBrowserCmd(), newDigestCmd(), newHookCmd(), newKillCmd(), newNoteCmd(), newShowCmd(), newStatsCmd(), newTrackCmd(), newUpdateCmd(l), newVersionCmd())
	root.AddCommand(newAppCmds()...)
	return root
}
