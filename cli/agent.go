package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/control"
)

// The commands in this file are for agents running in a session. Each one asks
// the running desktop app to do the work and prints what the app answers.

func newNoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "note <text>",
		Short: "Add a note to this project",
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")
			if len(args) == 0 {
				data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 1<<20))
				if err != nil {
					return fmt.Errorf("reading the note: %w", err)
				}
				text = string(data)
			}
			return askApp(cmd, control.Request{Cmd: "note", Args: []string{text}})
		},
	}
}

func newStatsCmd() *cobra.Command {
	var days int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Print what interrupted the owner most in this project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := map[string]string{"days": strconv.Itoa(days)}
			if asJSON {
				opts["json"] = "1"
			}
			return askApp(cmd, control.Request{Cmd: "stats", Opts: opts})
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "how many days to look back")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	return cmd
}
