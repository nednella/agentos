// Package network tells the front end whether the app reaches the network, from the app's own
// calls: a call that got no answer means offline, and any answer, an error status too, means online.
package network

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"

	"github.com/nednella/agentos/desktop/internal/run"
)

// Status is the payload of the network event.
type Status struct {
	Online bool `json:"online"`
}

// Watch emits the network event: at each call that got no answer, and at the first answer after one.
type Watch struct {
	emit    func(event string, payload any)
	mu      sync.Mutex
	offline bool
}

func New(emit func(event string, payload any)) *Watch {
	return &Watch{emit: emit}
}

// failed reports a call that got no answer. It emits each time, so a front end that started after
// the first failure still hears it.
func (w *Watch) failed() {
	w.mu.Lock()
	w.offline = true
	w.mu.Unlock()
	w.emit("network", Status{Online: false})
}

// answered reports a call that got an answer.
func (w *Watch) answered() {
	w.mu.Lock()
	was := w.offline
	w.offline = false
	w.mu.Unlock()
	if was {
		w.emit("network", Status{Online: true})
	}
}

// Runner reports each gh call, and each git command that talks to a remote.
func (w *Watch) Runner(r run.Runner) run.Runner {
	return func(ctx context.Context, dir, name string, args ...string) ([]byte, error) {
		out, err := r(ctx, dir, name, args...)
		if name == "gh" || name == "git" && remote(args) {
			w.report(err)
		}
		return out, err
	}
}

func (w *Watch) report(err error) {
	switch {
	case errors.Is(err, context.Canceled):
	case unanswered(err):
		w.failed()
	default:
		w.answered()
	}
}

// noAnswer are what gh and git print when they reach no server: gh's own DNS message, Go's dial
// and timeout errors that gh passes on, and curl's and ssh's for git.
var noAnswer = []string{
	"error connecting to",
	"dial tcp",
	"i/o timeout",
	"tls handshake timeout",
	"connection reset by peer",
	"could not resolve host",
	"failed to connect to",
	"connection timed out",
	"operation timed out",
	"network is unreachable",
	"connection refused",
}

func unanswered(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, s := range noAnswer {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
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

// Transport reports each request: one with no answer as a failure, any answer as online.
func (w *Watch) Transport(rt http.RoundTripper) http.RoundTripper {
	return transport{rt: rt, w: w}
}

type transport struct {
	rt http.RoundTripper
	w  *Watch
}

func (t transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.rt.RoundTrip(req)
	switch {
	case err == nil:
		t.w.answered()
	case !errors.Is(err, context.Canceled):
		t.w.failed()
	}
	return resp, err
}
