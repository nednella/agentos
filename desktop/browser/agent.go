package browser

import (
	"context"
	"errors"
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

// AgentOpen is agentos browser open: it loads the page in the session's window and prints it.
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
	return "opened " + pg.line(ctx), nil
}
