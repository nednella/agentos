// Package network tells the front end when a call that needs the network fails, so it can check
// whether GitHub is reachable. It only suggests a check: the front end's probe decides.
package network

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
)

// gap is the shortest time between two network events, so a burst of failures starts one check.
const gap = 5 * time.Second

// Watch emits the network event when a call that needs the network fails.
type Watch struct {
	emit func(event string, payload any)
	now  func() time.Time
	mu   sync.Mutex
	last time.Time
}

func New(emit func(event string, payload any)) *Watch {
	return &Watch{emit: emit, now: time.Now}
}

// Failed reports a failed call. A call its caller cancelled says nothing about the network.
func (w *Watch) Failed(err error) {
	if err == nil || errors.Is(err, context.Canceled) {
		return
	}
	w.mu.Lock()
	now := w.now()
	if !w.last.IsZero() && now.Sub(w.last) < gap {
		w.mu.Unlock()
		return
	}
	w.last = now
	w.mu.Unlock()
	w.emit("network", nil)
}

// Runner reports each failed gh call, and each failed git command that talks to a remote.
func (w *Watch) Runner(r run.Runner) run.Runner {
	return func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		out, err := r(ctx, dir, name, args...)
		if err != nil && (name == "gh" || name == "git" && remote(args)) {
			w.Failed(err)
		}
		return out, err
	}
}

// remote is whether the git command, after its options, is one that talks to a remote.
func remote(args []string) bool {
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-C", "-c":
			i++
		case "fetch", "push", "ls-remote":
			return true
		default:
			if args[i] == "" || args[i][0] != '-' {
				return false
			}
		}
	}
	return false
}

// Transport reports each request that got no answer. Any answer, an error status too, means the
// network works.
func (w *Watch) Transport(rt http.RoundTripper) http.RoundTripper {
	return transport{rt: rt, w: w}
}

type transport struct {
	rt http.RoundTripper
	w  *Watch
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.rt.RoundTrip(req)
	if err != nil {
		t.w.Failed(err)
	}
	return resp, err
}
