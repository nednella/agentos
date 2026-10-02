package cmd

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/guard"
)

// newGuardCmd is a PreToolUse hook for Bash. It denies commands that match the
// session's guard patterns, and says nothing otherwise. It never fails a command
// for its own sake: any error leaves the tool call to the normal flow.
func newGuardCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "guard",
		Short:  "Deny shell commands the owner reserves (called by the agents)",
		Hidden: true,
		Args:   cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprint(cmd.OutOrStdout(), guardVerdict(cmd.InOrStdin()))
		},
	}
}

// guardVerdict is the hook output for the call on r: a deny decision or nothing.
func guardVerdict(r io.Reader) string {
	var in struct {
		ToolInput struct {
			Command string `json:"command"`
		} `json:"tool_input"`
	}
	if json.NewDecoder(io.LimitReader(r, 1<<20)).Decode(&in) != nil || !guard.Denied(guard.FromEnv(), in.ToolInput.Command) {
		return ""
	}
	out, err := json.Marshal(map[string]any{"hookSpecificOutput": map[string]string{
		"hookEventName":            "PreToolUse",
		"permissionDecision":       "deny",
		"permissionDecisionReason": guard.Reason,
	}})
	if err != nil {
		return ""
	}
	return string(out) + "\n"
}
