package browser

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"
)

const loadLimit = 30 * time.Second

// waitLoaded waits for the page to finish loading after a navigation that began when loadSeq was before.
func (pg *page) waitLoaded(ctx context.Context, before int) error {
	ctx, cancel := context.WithTimeout(ctx, loadLimit)
	defer cancel()
	start := time.Now()
	for {
		pg.t.mu.Lock()
		loaded := pg.loadSeq > before && !pg.loading
		loading := pg.loading
		pg.t.mu.Unlock()
		if loaded {
			return nil
		}
		// A navigation inside the page fires no load event.
		if !loading && time.Since(start) > 500*time.Millisecond {
			if raw, err := pg.eval(ctx, "document.readyState"); err == nil && string(raw) == `"complete"` {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.New("the page did not finish loading")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (pg *page) seq() int {
	pg.t.mu.Lock()
	defer pg.t.mu.Unlock()
	return pg.loadSeq
}

// tabFor returns the session's tab, opening one if there is none.
func (b *Browsers) tabFor(ctx context.Context, id string) (*tab, error) {
	if t := b.tab(id); t != nil {
		return t, nil
	}
	if _, err := b.Open(ctx, id, ""); err != nil {
		return nil, err
	}
	return b.need(id)
}

// pageHelp is the last line of what the agent commands print for a page in a headed window.
func (t *tab) pageHelp() string {
	if t.prefix == "" {
		return ""
	}
	return "\npage: " + t.prefix + " - find it in list_pages by this title prefix"
}

// AgentOpen is agentos browser open: it loads the page in the session's window and says how to find it.
func (b *Browsers) AgentOpen(ctx context.Context, id, address string, front bool) (string, error) {
	t, err := b.tabFor(ctx, id)
	if err != nil {
		return "", err
	}
	pg, err := t.refreshActive(ctx)
	if err != nil {
		return "", err
	}
	before := pg.seq()
	if err := pg.goTo(ctx, address); err != nil {
		return "", err
	}
	if err := pg.waitLoaded(ctx, before); err != nil {
		return "", err
	}
	if front {
		if err := pg.activate(ctx); err != nil {
			return "", err
		}
	}
	return "opened " + pg.line(ctx) + t.pageHelp(), nil
}

// AgentTab is agentos browser tab: it opens the address in a new tab of the session's window. The newest page opens
// it with window.open, run as if the user had clicked so the popup blocker lets it through and the browser puts the
// tab in the opener's window; the new page joins the session as the opener's child.
func (b *Browsers) AgentTab(ctx context.Context, id, address string) (string, error) {
	address, err := webAddress(address)
	if err != nil {
		return "", err
	}
	t, err := b.tabFor(ctx, id)
	if err != nil {
		return "", err
	}
	opener, err := t.newest()
	if err != nil {
		return "", err
	}
	known := t.targets()
	quoted, _ := json.Marshal(address)
	opened, err := opener.evaluate(ctx, "window.open("+string(quoted)+", '_blank') !== null", true)
	if err != nil {
		return "", err
	}
	if string(opened) != "true" {
		return "", errors.New("the browser blocked the new tab")
	}
	pg, err := t.waitNewPage(ctx, known)
	if err != nil {
		return "", err
	}
	if err := pg.waitSettled(ctx, address); err != nil {
		return "", err
	}
	t.refreshAll(ctx)
	return "opened tab " + pg.line(ctx) + t.pageHelp(), nil
}

// targets are the targets of the session's pages.
func (t *tab) targets() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	ids := make([]string, len(t.pages))
	for i, pg := range t.pages {
		ids[i] = pg.targetID
	}
	return ids
}

// waitNewPage waits for the session to take in a page that is not among known.
func (t *tab) waitNewPage(ctx context.Context, known []string) (*page, error) {
	ctx, cancel := context.WithTimeout(ctx, newPageLimit)
	defer cancel()
	for {
		t.mu.Lock()
		for _, pg := range t.pages {
			if !slices.Contains(known, pg.targetID) && pg.sessionID != "" {
				t.mu.Unlock()
				return pg, nil
			}
		}
		t.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, errors.New("the new tab did not open")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// waitSettled waits until a page opened by a script has left its first empty document and finished loading. The
// load events of a popup can pass before agentos listens, so it reads the document instead.
func (pg *page) waitSettled(ctx context.Context, address string) error {
	ctx, cancel := context.WithTimeout(ctx, loadLimit)
	defer cancel()
	for {
		raw, err := pg.eval(ctx, `location.href + "\n" + document.readyState`)
		var both string
		if err == nil && json.Unmarshal(raw, &both) == nil {
			addr, state, _ := strings.Cut(both, "\n")
			if state == "complete" && (addr != "about:blank" || strings.HasPrefix(address, "about:")) {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.New("the page did not finish loading")
		case <-time.After(100 * time.Millisecond):
		}
	}
}
