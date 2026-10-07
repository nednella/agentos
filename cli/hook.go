package cli

import (
	"context"
	"io"
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
			reportHook(args[0], cmd.InOrStdin())
		},
	}
}

// reportHook saves the session's new state and tells the app, within a short
// budget: the file is already saved, so a miss costs only a delay.
func reportHook(event string, stdin io.Reader) { reportHookAt(event, stdin, time.Now()) }

// reportHookAt records the event as of at, the moment the hook started: a hook that waited
// for the lock must not undo a later event that got in first.
func reportHookAt(event string, stdin io.Reader, at time.Time) {
	name := os.Getenv("AGENTOS_SESSION")
	socket := os.Getenv("AGENTOS_SOCKET")
	if _, err := session.ParseName(name); err != nil || socket == "" {
		return
	}
	dir := bus.DirOf(socket)
	ev := session.ParseEvent(event, stdin)
	rec, ok := saveHook(dir, name, ev, at)
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), hookBudget)
	defer cancel()
	_ = bus.Send(ctx, socket, rec)
}

// saveHook folds the event into the saved state under the file's lock. It reports whether the state changed.
func saveHook(dir, name string, ev session.Event, at time.Time) (session.Record, bool) {
	unlock, err := bus.LockState(dir, name)
	if err != nil {
		return session.Record{}, false
	}
	defer unlock()
	prev, _ := bus.ReadState(dir, name)
	if prev.Session == "" && ev.Name != "SessionStart" {
		return session.Record{}, false // a hook that fires after the session was dismissed must not bring its file back
	}
	if prev.At.After(at) {
		return session.Record{}, false
	}
	rec := session.Apply(name, prev, ev, at)
	if rec == prev || bus.WriteState(dir, rec) != nil {
		return session.Record{}, false
	}
	return rec, true
}
