package news

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	feedPage = "https://tldr.tech/api/rss/dev"
	dayPage  = "https://tldr.tech/dev/"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type site struct {
	t     *testing.T
	pages map[string][]byte
	err   error

	mu   sync.Mutex
	hits []string
}

func (s *site) fetch(_ context.Context, url string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits = append(s.hits, url)
	if s.err != nil {
		return nil, s.err
	}
	body, ok := s.pages[url]
	if !ok {
		s.t.Errorf("fetched %s", url)
		return nil, errors.New("no such page")
	}
	return body, nil
}

func (s *site) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.hits)
}

type recorder struct {
	mu       sync.Mutex
	started  int
	finished int
	last     News
}

func (r *recorder) emit(event string, payload any) {
	n, ok := payload.(News)
	if !ok || event != "news" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if n.Running {
		r.started++
	} else {
		r.finished++
	}
	r.last = n
}

func newSite(t *testing.T) *site {
	page := fixture(t, "issue.html")
	return &site{t: t, pages: map[string][]byte{
		feedPage:               fixture(t, "feed.xml"),
		dayPage + "2026-10-09": page,
		dayPage + "2026-10-08": page,
		dayPage + "2026-10-07": page,
		dayPage + "2026-10-06": page,
	}}
}

func newManager(t *testing.T, s *site) (*Manager, *recorder) {
	t.Helper()
	rec := &recorder{}
	m := NewManager(NewStore(t.TempDir()), s.fetch, rec.emit, context.Background)
	m.gap = 0
	return m, rec
}

// settled waits until the nth fetch has ended and returns the news it ended with.
func settled(t *testing.T, rec *recorder, n int) News {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rec.mu.Lock()
		done, last := rec.finished >= n, rec.last
		rec.mu.Unlock()
		if done {
			return last
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("fetch %d did not finish", n)
	return News{}
}

func TestParsePage(t *testing.T) {
	items, err := parsePage(fixture(t, "issue.html"), "2026-10-09")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 (the sponsor is skipped): %+v", len(items), items)
	}
	first := items[0]
	if first.ID != "2026-10-09-0" || first.Title != "We built our own cloud agents runtime. Here's what we learned" || first.ReadTime != "11 minute read" {
		t.Errorf("first = %+v", first)
	}
	if first.URL != "https://posthog.com/blog/cloud-agents-runtime" {
		t.Errorf("tracking parameter left in %q", first.URL)
	}
	if strings.ContainsAny(first.Summary, "<>") || strings.Contains(first.Summary, "  ") || !strings.HasPrefix(first.Summary, "PostHog describes") {
		t.Errorf("summary = %q", first.Summary)
	}
	if _, err := parsePage([]byte("<html><body>nothing</body></html>"), "2026-10-09"); err == nil {
		t.Error("a page with no stories was accepted")
	}
}

func TestCleanURL(t *testing.T) {
	for in, want := range map[string]string{
		"https://a.com/x?utm_source=tldrdev":                 "https://a.com/x",
		"https://a.com/x?id=1&utm_source=tldrnewsletter&b=2": "https://a.com/x?id=1&b=2",
		"https://a.com/x?utm_source=other&utm_medium=email":  "https://a.com/x?utm_source=other&utm_medium=email",
		"https://a.com/x": "https://a.com/x",
	} {
		if got, ok := cleanURL(in); !ok || got != want {
			t.Errorf("cleanURL(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	if _, ok := cleanURL("javascript:alert(1)"); ok {
		t.Error("a script link was accepted")
	}
}

func TestParseFeed(t *testing.T) {
	issues, err := parseFeed(fixture(t, "feed.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 4 || issues[0].Date != "2026-10-09" || issues[0].Title != "ChatGPT's intelligent UI 🧩, cost of open source 💲, software judgment 🧠" {
		t.Errorf("issues = %+v", issues)
	}
	if _, err := parseFeed([]byte(`{"error":"too many requests"}`)); err == nil {
		t.Error("a body that is not a feed was accepted")
	}
}

func TestRefreshFetchesNewestThreeAndKeepsThem(t *testing.T) {
	s := newSite(t)
	m, rec := newManager(t, s)
	if err := m.start(); err != nil {
		t.Fatal(err)
	}
	got := settled(t, rec, 1)
	if rec.started != 1 {
		t.Errorf("%d start events, want 1", rec.started)
	}
	if got.Error != "" || got.LastFetchAt == 0 || len(got.Issues) != 3 || got.Issues[0].Date != "2026-10-09" || got.Issues[2].Date != "2026-10-07" {
		t.Fatalf("news = %+v", got)
	}
	if got.Issues[0].FetchedAt == 0 {
		t.Error("a fetched issue has no FetchedAt")
	}
	if s.count() != 4 {
		t.Errorf("made %d requests, want the feed and 3 pages", s.count())
	}

	if err := m.start(); err != nil {
		t.Fatal(err)
	}
	got = settled(t, rec, 2)
	if len(got.Issues) != 4 || got.Issues[3].Date != "2026-10-06" {
		t.Errorf("the second fetch left %d issues, want the fourth day added", len(got.Issues))
	}
	if s.count() != 6 {
		t.Errorf("made %d requests, want only the feed and the one new page more", s.count())
	}
}

func TestIssuesKeptAreTheLastSeven(t *testing.T) {
	st := NewStore(t.TempDir())
	var fresh []Issue
	for _, d := range []string{"01", "02", "03", "04", "05", "06", "07", "08", "09"} {
		fresh = append(fresh, Issue{Date: "2026-10-" + d})
	}
	if err := st.Begin(); err != nil {
		t.Fatal(err)
	}
	st.Finish(time.Now(), fresh, "")
	got := st.View().Issues
	if len(got) != 7 || got[0].Date != "2026-10-09" || got[6].Date != "2026-10-03" {
		t.Errorf("issues = %+v", got)
	}
}

func TestFailureKeepsOldNews(t *testing.T) {
	s := newSite(t)
	m, rec := newManager(t, s)
	if err := m.start(); err != nil {
		t.Fatal(err)
	}
	before := settled(t, rec, 1)

	for i, c := range []struct {
		name string
		body []byte
	}{
		{"a refused request", []byte(`{"error":"too many requests"}`)},
		{"a broken feed", []byte("<html>oops</html>")},
	} {
		s.pages[feedPage] = c.body
		if err := m.start(); err != nil {
			t.Fatal(err)
		}
		got := settled(t, rec, i+2)
		if got.Error == "" || got.LastFetchAt != before.LastFetchAt || len(got.Issues) != len(before.Issues) {
			t.Errorf("%s: news = %+v", c.name, got)
		}
	}

	s.err = errors.New("offline")
	if err := m.start(); err != nil {
		t.Fatal(err)
	}
	if got := settled(t, rec, 4); got.Error != "offline" {
		t.Errorf("error = %q", got.Error)
	}
}

func TestPartialFetchKeepsWhatItGot(t *testing.T) {
	s := newSite(t)
	s.pages[dayPage+"2026-10-08"] = []byte("<html></html>")
	m, rec := newManager(t, s)
	if err := m.start(); err != nil {
		t.Fatal(err)
	}
	got := settled(t, rec, 1)
	if len(got.Issues) != 1 || got.Issues[0].Date != "2026-10-09" || got.Error == "" || got.LastFetchAt != 0 {
		t.Errorf("news = %+v", got)
	}
}

func TestSecondFetchWhileRunningIsRefused(t *testing.T) {
	st := NewStore(t.TempDir())
	if err := st.Begin(); err != nil {
		t.Fatal(err)
	}
	if err := st.Begin(); err == nil {
		t.Error("two fetches ran at once")
	}
	if st.Due(time.Now()) {
		t.Error("a fetch is due while one runs")
	}
}

func TestLoopFetchesOnlyStaleNews(t *testing.T) {
	s := newSite(t)
	m, rec := newManager(t, s)
	m.SetLoop(0, time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go m.Loop(ctx)
	got := settled(t, rec, 1)
	if len(got.Issues) != 3 {
		t.Fatalf("the loop did not fetch: %+v", got)
	}

	hits := s.count()
	m2 := NewManager(m.store, s.fetch, rec.emit, context.Background)
	m2.SetLoop(0, time.Hour)
	go m2.Loop(ctx)
	time.Sleep(100 * time.Millisecond)
	if s.count() != hits {
		t.Error("fresh news were fetched again")
	}
	if !m.store.Due(time.Now().Add(4 * time.Hour)) {
		t.Error("news older than 3 hours are not due")
	}
}
