package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
)

func newKillCmd() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "kill",
		Short: "Stop this project's agents (--all for every project)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			e, err := loadEnv()
			if err != nil {
				return err
			}
			infos, err := e.tmux.List(cmd.Context())
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
			if stopped == 0 {
				fmt.Fprintln(out, "no agents to stop")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "stop the agents of every project")
	return cmd
}
