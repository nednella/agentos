package network

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newWatch() (*Watch, *int, *time.Time) {
	events := 0
	now := time.Unix(1000, 0)
	w := New(func(event string, payload any) {
		if event != "network" || payload != nil {
			panic(fmt.Sprintf("emitted %q %v", event, payload))
		}
		events++
	})
	w.now = func() time.Time { return now }
	return w, &events, &now
}

func TestRunnerReportsOnlyNetworkCommands(t *testing.T) {
	failed := errors.New("exit status 1")
	cases := []struct {
		name string
		args []string
		err  error
		want int
	}{
		{"gh", []string{"issue", "list"}, failed, 1},
		{"gh", []string{"issue", "list"}, nil, 0},
		{"gh", []string{"issue", "list"}, fmt.Errorf("gh issue: %w", context.Canceled), 0},
		{"gh", []string{"issue", "list"}, context.DeadlineExceeded, 1},
		{"git", []string{"fetch", "origin"}, failed, 1},
		{"git", []string{"push", "-u", "origin", "x"}, failed, 1},
		{"git", []string{"-C", "/repo", "ls-remote", "origin"}, failed, 1},
		{"git", []string{"-c", "core.x=1", "fetch"}, failed, 1},
		{"git", []string{"rev-parse", "fetch"}, failed, 0},
		{"git", []string{"-C", "fetch", "status"}, failed, 0},
		{"git", []string{"worktree", "list"}, failed, 0},
		{"tmux", []string{"fetch"}, failed, 0},
	}
	for _, c := range cases {
		w, events, _ := newWatch()
		run := w.Runner(func(context.Context, string, string, ...string) ([]byte, error) { return nil, c.err })
		_, err := run(context.Background(), "", c.name, c.args...)
		if err != c.err {
			t.Errorf("%s %v: the runner changed the error to %v", c.name, c.args, err)
		}
		if *events != c.want {
			t.Errorf("%s %v failing with %v: %d events, want %d", c.name, c.args, c.err, *events, c.want)
		}
	}
}

func TestFailuresEmitOnceInEachGap(t *testing.T) {
	w, events, now := newWatch()
	failed := errors.New("down")
	w.Failed(failed)
	w.Failed(failed)
	*now = now.Add(gap - time.Second)
	w.Failed(failed)
	if *events != 1 {
		t.Fatalf("a burst emitted %d events, want 1", *events)
	}
	*now = now.Add(time.Second)
	w.Failed(failed)
	if *events != 2 {
		t.Errorf("a failure after the gap emitted %d events in all, want 2", *events)
	}
}

func TestTransportReportsOnlyRequestsWithNoAnswer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.WriteHeader(http.StatusServiceUnavailable)
	}))
	w, events, _ := newWatch()
	client := &http.Client{Transport: w.Transport(http.DefaultTransport)}

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if *events != 0 {
		t.Errorf("an error status emitted %d events, want 0", *events)
	}

	server.Close()
	if _, err := client.Get(server.URL); err == nil {
		t.Fatal("a closed server answered")
	}
	if *events != 1 {
		t.Errorf("a request with no answer emitted %d events, want 1", *events)
	}
}
