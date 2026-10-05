package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/session"
)

func newKillCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:     "kill [n]",
		Short:   "Stop session n in the app, or without n this project's agents (--all: every project, shells too)",
		GroupID: groupWork,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return askApp(cmd, control.Request{Cmd: "kill", Args: args})
			}
			e, err := loadEnv()
			if err != nil {
				return err
			}
			infos, shells, err := e.tmux.ListAll(cmd.Context())
			if err != nil {
				return fmt.Errorf("listing agents: %w", err)
			}
			out := cmd.OutOrStdout()
			stopped := 0
			for _, in := range infos {
				if !all && in.Name.Project != e.project.Key() {
					continue
				}
				fmt.Fprintf(out, "stopping %s (%s)\n", in.Name, in.Title)
				if err := e.tmux.Kill(cmd.Context(), in.Name); err != nil {
					return err
				}
				if err := bus.RemoveState(e.stateDir, in.Name.String()); err != nil {
					return err
				}
				stopped++
			}
			if all { // a shell is only stopped when everything is
				for _, key := range shells {
					name := session.Name{Project: key}
					fmt.Fprintf(out, "stopping %s\n", name)
					if err := e.tmux.Kill(cmd.Context(), name); err != nil {
						return err
					}
					stopped++
				}
			}
			if stopped == 0 {
				fmt.Fprintln(out, "no agents to stop")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "stop the agents of every project, and the shells")
	return cmd
}
