package update

import (
	"context"

	ctl "github.com/nednella/agentos/internal/control"
)

// Commands answers agentos update, which installs the release itself and then asks the running
// app to start again from the new bundle.
type Commands struct{ u *Updater }

func NewCommands(u *Updater) *Commands { return &Commands{u: u} }

// Relaunch opens the app again and quits this one.
func (c *Commands) Relaunch(context.Context, ctl.Request) (string, error) {
	return "ok", c.u.Relaunch()
}
