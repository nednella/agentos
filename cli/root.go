// Package cli is the agentos command line.
package cli

import (
	"context"

	"github.com/spf13/cobra"
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
	root.AddCommand(newHookCmd(), newVersionCmd())
	return root
}
