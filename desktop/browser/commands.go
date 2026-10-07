package browser

import (
	"context"
	"errors"
	"fmt"

	"github.com/nednella/agentos/desktop/evidence"
	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/prompts"
)

// Asker checks which session a command comes from.
type Asker interface {
	AskerSession(req control.Request) (string, error)
}

// Scene shows things on the screen of the project a command comes from.
type Scene interface {
	UI(req control.Request, name string, args ...string) string
}

// Commands answers agentos browser.
type Commands struct {
	browsers *Browsers
	asker    Asker
	scene    Scene
	store    *evidence.Store
	changes  evidence.Changes
}

func NewCommands(b *Browsers, a Asker, scene Scene, s *evidence.Store, c evidence.Changes) *Commands {
	return &Commands{browsers: b, asker: a, scene: scene, store: s, changes: c}
}

// Browser runs one agentos browser command (open, tab, screenshot) for the asking session; without one it shows the browser view.
func (c *Commands) Browser(ctx context.Context, req control.Request) (string, error) {
	if len(req.Args) == 0 {
		return c.scene.UI(req, "browser"), nil
	}
	sub, args := req.Args[0], req.Args[1:]
	if sub == "help" {
		return prompts.BrowserHelp(), nil
	}
	if sub != "open" && sub != "tab" && sub != "screenshot" {
		return "", fmt.Errorf("unknown browser command %q: run agentos browser help", sub)
	}
	id, err := c.asker.AskerSession(req)
	if err != nil {
		return "", err
	}
	if !c.browsers.Available() {
		return "", errors.New("no Brave, Chrome, Chromium or Edge browser found")
	}
	switch sub {
	case "open":
		if len(args) < 1 {
			return "", errors.New("usage: agentos browser open <url> [--front]")
		}
		return c.browsers.AgentOpen(ctx, id, args[0], req.Opts["front"] != "")
	case "tab":
		if len(args) < 1 {
			return "", errors.New("usage: agentos browser tab <url>")
		}
		return c.browsers.AgentTab(ctx, id, args[0])
	}
	if _, err := c.browsers.tabFor(ctx, id); err != nil {
		return "", err
	}
	png, err := c.browsers.Screenshot(ctx, id, req.Opts["full"] != "")
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
