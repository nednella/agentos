package cli

import (
	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/mcp"
)

// newMCPBrowserCmd is the browser's MCP server as a session's Claude sees it: it lists the page tools and starts the real
// server, the command after --, only when the session first uses one.
func newMCPBrowserCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "mcp-browser -- <server command>",
		Short:  "Stand in for the browser's MCP server until a tool is used (called by the agents)",
		Hidden: true,
		Args:   cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return mcp.Serve(cmd.Context(), mcp.Browser, args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}
