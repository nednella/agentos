package browser

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

const loadLimit = 30 * time.Second

// waitLoaded waits for the page to finish loading after a navigation that began when loadSeq was before.
func (t *tab) waitLoaded(ctx context.Context, before int) error {
	ctx, cancel := context.WithTimeout(ctx, loadLimit)
	defer cancel()
	start := time.Now()
	for {
		t.mu.Lock()
		loaded := t.loadSeq > before && !t.state.Loading
		loading := t.state.Loading
		t.mu.Unlock()
		if loaded {
			return nil
		}
		// A navigation inside the page fires no load event.
		if !loading && time.Since(start) > 500*time.Millisecond {
			if raw, err := t.eval(ctx, "document.readyState"); err == nil && string(raw) == `"complete"` {
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

func (t *tab) seq() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.loadSeq
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

// pageLine is the page's title and address, read from the page itself.
func (t *tab) pageLine(ctx context.Context) string {
	raw, err := t.eval(ctx, `document.title + "\n" + location.href`)
	var both string
	if err != nil || json.Unmarshal(raw, &both) != nil {
		s := t.snapshotState()
		return s.Title + " - " + s.URL
	}
	title, addr, _ := strings.Cut(both, "\n")
	return t.pageTitle(title) + " - " + addr
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
	before := t.seq()
	if err := b.Goto(ctx, id, address); err != nil {
		return "", err
	}
	if err := t.waitLoaded(ctx, before); err != nil {
		return "", err
	}
	if front {
		if err := b.Show(ctx, id); err != nil {
			return "", err
		}
	}
	return "opened " + t.pageLine(ctx) + t.pageHelp(), nil
}
