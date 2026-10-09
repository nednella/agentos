package news

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/util"
)

const (
	feedURL      = "https://tldr.tech/api/rss/dev"
	fetchTimeout = 2 * time.Minute
	maxBody      = 4 << 20
	// TLDR blocks clients that do not look like a browser.
	userAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15"
)

// Fetch returns the body of a web page.
type Fetch func(ctx context.Context, url string) ([]byte, error)

// HTTPFetch fetches with client.
func HTTPFetch(client *http.Client) Fetch {
	return func(ctx context.Context, url string) ([]byte, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("%s answered %s", url, resp.Status)
		}
		return io.ReadAll(io.LimitReader(resp.Body, maxBody))
	}
}

// Manager keeps the news up to date.
type Manager struct {
	store *Store
	fetch Fetch
	emit  func(event string, payload any)
	ctx   func() context.Context
	gap   time.Duration // between two page requests, because TLDR limits the rate hard
	first time.Duration // wait before the first automatic check
	tick  time.Duration // between checks after that
}

func NewManager(s *Store, fetch Fetch, emit func(string, any), ctx func() context.Context) *Manager {
	return &Manager{store: s, fetch: fetch, emit: emit, ctx: ctx, gap: time.Second, first: 5 * time.Second, tick: time.Hour}
}

// SetLoop sets how long to wait before the first automatic check, and between checks.
func (m *Manager) SetLoop(first, tick time.Duration) { m.first, m.tick = first, tick }

// SetFetch replaces how pages are fetched, for tests.
func (m *Manager) SetFetch(fetch Fetch) { m.fetch = fetch }

func (m *Manager) view() News { return m.store.View() }

// start fetches in the background; the progress arrives as news events.
func (m *Manager) start() error {
	if err := m.store.Begin(); err != nil {
		return err
	}
	m.emit("news", m.view())
	go func() {
		ctx, cancel := context.WithTimeout(m.ctx(), fetchTimeout)
		defer cancel()
		fresh, err := m.refresh(ctx)
		failure := ""
		switch {
		case errors.Is(ctx.Err(), context.DeadlineExceeded):
			failure = "the fetch took longer than 2 minutes"
		case err != nil:
			failure = util.FirstLine(err.Error())
		}
		m.store.Finish(time.Now(), fresh, failure)
		m.emit("news", m.view())
	}()
	return nil
}

// refresh gets the newest issues that are not stored. It returns the ones it got before an error.
func (m *Manager) refresh(ctx context.Context) ([]Issue, error) {
	body, err := m.get(ctx, feedURL)
	if err != nil {
		return nil, err
	}
	listed, err := parseFeed(body)
	if err != nil {
		return nil, err
	}
	var fresh []Issue
	for _, issue := range listed {
		if len(fresh) == issuesPerFetch {
			break
		}
		if m.store.Has(issue.Date) {
			continue
		}
		if len(fresh) > 0 {
			select {
			case <-ctx.Done():
				return fresh, ctx.Err()
			case <-time.After(m.gap):
			}
		}
		page, err := m.get(ctx, issue.URL)
		if err != nil {
			return fresh, err
		}
		if issue.Items, err = parsePage(page, issue.Date); err != nil {
			return fresh, fmt.Errorf("%s: %w", issue.Date, err)
		}
		issue.FetchedAt = time.Now().UnixMilli()
		fresh = append(fresh, issue)
	}
	return fresh, nil
}

// get fetches a page. TLDR answers a refused request with a short text and a good status.
func (m *Manager) get(ctx context.Context, url string) ([]byte, error) {
	body, err := m.fetch(ctx, url)
	if err != nil {
		return nil, err
	}
	if len(body) < 1024 && strings.Contains(strings.ToLower(string(body)), "too many requests") {
		return nil, errors.New("TLDR is limiting requests, try again later")
	}
	return body, nil
}

// Loop fetches when the news are stale, checking first after m.first, then every m.tick, until ctx ends.
func (m *Manager) Loop(ctx context.Context) {
	timer := time.NewTimer(m.first)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if m.store.Due(time.Now()) {
				_ = m.start()
			}
			timer.Reset(m.tick)
		}
	}
}
