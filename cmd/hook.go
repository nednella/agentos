package cmd

import (
	"context"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
)

const hookBudget = 200 * time.Millisecond

// newHookCmd is what Claude Code runs on each hook event. It must never break a
// turn, so it reports nothing on stdout and always exits 0.
func newHookCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "hook <event>",
		Short:  "Record an agent hook event (called by the agents)",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			reportHook(args[0])
		},
	}
}

func reportHook(event string) {
	name := os.Getenv("AGENTOS_SESSION")
	socket := os.Getenv("AGENTOS_SOCKET")
	if _, err := session.ParseName(name); err != nil || socket == "" {
		return
	}
	dir := bus.DirOf(socket)
	prev, _ := bus.ReadState(dir, name)
	rec := session.Apply(name, prev, session.ParseEvent(event, os.Stdin), time.Now())
	if rec == prev {
		return
	}
	if err := bus.WriteState(dir, rec); err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), hookBudget)
	defer cancel()
	_ = bus.Send(ctx, socket, rec)
}
