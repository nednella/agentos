package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/version"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the agentos version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "agentos", version.Version)
		},
	}
}
