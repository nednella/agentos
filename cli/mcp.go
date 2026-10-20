package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/agent"
	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/mcp"
	"github.com/nednella/agentos/internal/session"
)

// newMCPBrowserCmd is the browser's MCP server as a session's agent sees it: it lists the page tools and starts the real
// server only when the session first uses one. Claude names the real server after --; agy's global entry names
// nothing, and the command asks the app for the session's server.
func newMCPBrowserCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "mcp-browser [-- <server command>]",
		Short:  "Stand in for the browser's MCP server until a tool is used (called by the agents)",
		Hidden: true,
		Args:   cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			server, argv, env := mcp.Browser, args, []string(nil)
			if len(args) == 0 {
				server, argv, env = browserServer(cmd.Context(), cmd.ErrOrStderr())
			}
			return mcp.Serve(cmd.Context(), server, argv, env, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

// browserServer is the server of the asking session's page tools, from the app. Outside a session, or when the app
// cannot say or the session has no browser, it is the server with no tools: agy loads its global entry for every
// conversation, the owner's own too.
func browserServer(ctx context.Context, errOut io.Writer) (mcp.Server, []string, []string) {
	id := os.Getenv("AGENTOS_SESSION")
	if _, err := session.ParseName(id); err != nil {
		return mcp.None, nil, nil
	}
	socket, err := controlSocket()
	if err != nil {
		return mcp.None, nil, nil
	}
	resp, err := control.Call(ctx, socket, control.Request{Cmd: "mcp-browser", Session: id})
	if err == nil && !resp.OK {
		err = fmt.Errorf("%s", resp.Error)
	}
	var m agent.MCPServer
	if err == nil {
		err = json.Unmarshal([]byte(resp.Out), &m)
	}
	if err != nil {
		fmt.Fprintf(errOut, "agentos mcp-browser: no browser for this session: %v\n", err)
		return mcp.None, nil, nil
	}
	if m.Command == "" {
		return mcp.None, nil, nil
	}
	env := make([]string, 0, len(m.Env))
	for k, v := range m.Env {
		env = append(env, k+"="+v)
	}
	sort.Strings(env)
	return mcp.Browser, append([]string{m.Command}, m.Args...), env
}
