package browser

import (
	"context"
	"errors"

	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/internal/control"
)

// Asker checks which session a command comes from.
type Asker interface {
	AskerSession(req control.Request) (string, error)
}

// Commands answers agentos browser.
type Commands struct {
	browsers *Browsers
	asker    Asker
	store    *evidence.Store
	changes  evidence.Changes
}

func NewCommands(b *Browsers, a Asker, s *evidence.Store, c evidence.Changes) *Commands {
	return &Commands{browsers: b, asker: a, store: s, changes: c}
}

// Browser runs one agentos browser command in the asking session's tab.
func (c *Commands) Browser(ctx context.Context, req control.Request) (string, error) {
	if len(req.Args) == 0 {
		return control.BrowserHelp, nil
	}
	sub, args := req.Args[0], req.Args[1:]
	if sub == "help" {
		return control.BrowserHelp, nil
	}
	id, err := c.asker.AskerSession(req)
	if err != nil {
		return "", err
	}
	if !c.browsers.Available() {
		return "", errors.New("no Brave, Chrome, Chromium or Edge browser found")
	}
	if sub != "screenshot" {
		return c.browsers.agent(ctx, id, sub, args, req.Opts)
	}
	ref := ""
	if len(args) > 0 {
		ref = args[0]
	}
	if _, err := c.browsers.tabFor(ctx, id); err != nil {
		return "", err
	}
	png, err := c.browsers.Screenshot(ctx, id, req.Opts["full"] != "", ref)
	if err != nil {
		return "", err
	}
	item, err := c.store.AddImage(id, png, req.Opts["caption"], "agent")
	if err != nil {
		return "", err
	}
	c.changes.Changed(c.store, id, true)
	return c.store.File(item.URL), nil
}
