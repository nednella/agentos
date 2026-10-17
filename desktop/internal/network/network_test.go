package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func newWatch() (*Watch, *[]bool) {
	var events []bool
	w := New(func(event string, payload any) {
		s, ok := payload.(Status)
		if event != "network" || !ok {
			panic(fmt.Sprintf("emitted %q %v", event, payload))
		}
		events = append(events, s.Online)
	})
	return w, &events
}

func TestRunnerTellsNoAnswerFromAnAnswer(t *testing.T) {
	noHost := errors.New("exit status 1: error connecting to api.github.com\ncheck your internet connection or https://githubstatus.com")
	refused := errors.New(`exit status 1: Get "https://api.github.com/graphql": dial tcp 140.82.121.6:443: connect: connection refused`)
	notFound := errors.New("exit status 1: GraphQL: Could not resolve to a Repository with the name 'me/new'. (repository)")
	limited := errors.New("exit status 1: gh: API rate limit exceeded (HTTP 403)")
	gitNoHost := errors.New("exit status 128: fatal: unable to access 'https://github.com/me/x/': Could not resolve host: github.com")
	gitDenied := errors.New("exit status 128: fatal: unable to access 'https://github.com/me/x/': The requested URL returned error: 403")
	sshNoHost := errors.New("exit status 128: ssh: Could not resolve hostname github.com: nodename nor servname provided, or not known")
	cases := []struct {
		name string
		args []string
		err  error
		want []bool
	}{
		{"gh", []string{"issue", "list"}, noHost, []bool{false}},
		{"gh", []string{"issue", "list"}, refused, []bool{false}},
		{"gh", []string{"issue", "list"}, context.DeadlineExceeded, []bool{false}},
		{"gh", []string{"repo", "view"}, notFound, nil},
		{"gh", []string{"issue", "list"}, limited, nil},
		{"gh", []string{"issue", "list"}, nil, nil},
		{"gh", []string{"issue", "list"}, fmt.Errorf("gh issue: %w", context.Canceled), nil},
		{"git", []string{"fetch", "origin"}, gitNoHost, []bool{false}},
		{"git", []string{"push", "-u", "origin", "x"}, sshNoHost, []bool{false}},
		{"git", []string{"-C", "/repo", "ls-remote", "origin"}, gitNoHost, []bool{false}},
		{"git", []string{"-c", "core.x=1", "fetch"}, gitNoHost, []bool{false}},
		{"git", []string{"fetch", "origin"}, gitDenied, nil},
		{"git", []string{"rev-parse", "fetch"}, gitNoHost, nil},
		{"git", []string{"-C", "fetch", "status"}, gitNoHost, nil},
		{"git", []string{"worktree", "list"}, gitNoHost, nil},
		{"tmux", []string{"fetch"}, gitNoHost, nil},
	}
	for _, c := range cases {
		w, events := newWatch()
		run := w.Runner(func(context.Context, string, string, ...string) ([]byte, error) { return nil, c.err })
		_, err := run(context.Background(), "", c.name, c.args...)
		if err != c.err {
			t.Errorf("%s %v: the runner changed the error to %v", c.name, c.args, err)
		}
		if !slices.Equal(*events, c.want) {
			t.Errorf("%s %v failing with %v: events %v, want %v", c.name, c.args, c.err, *events, c.want)
		}
	}
}

func TestAnAnswerAfterNoAnswerSetsOnlineOnce(t *testing.T) {
	w, events := newWatch()
	var err error
	run := w.Runner(func(context.Context, string, string, ...string) ([]byte, error) { return nil, err })
	call := func() { _, _ = run(context.Background(), "", "gh", "pr", "list") }

	err = errors.New("exit status 1: error connecting to api.github.com")
	call()
	call()
	err = errors.New("exit status 1: gh: Not Found (HTTP 404)")
	call()
	err = nil
	call()
	if want := []bool{false, false, true}; !slices.Equal(*events, want) {
		t.Errorf("events %v, want %v", *events, want)
	}
}

func TestTransportTellsNoAnswerFromAnAnswer(t *testing.T) {
	live := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer live.Close()
	gone := httptest.NewServer(http.NotFoundHandler())
	gone.Close()
	w, events := newWatch()
	client := &http.Client{Transport: w.Transport(http.DefaultTransport)}
	get := func(url string) {
		if resp, err := client.Get(url); err == nil {
			resp.Body.Close()
		}
	}

	get(live.URL)
	get(gone.URL)
	get(live.URL)
	if want := []bool{false, true}; !slices.Equal(*events, want) {
		t.Errorf("an error status, no answer, an error status: events %v, want %v", *events, want)
	}
}
