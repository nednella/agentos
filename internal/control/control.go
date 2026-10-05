// Package control carries commands from the agentos command line to the running
// desktop app: one JSON request line, one JSON response line, over a unix socket.
package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	socketFile     = "control.sock"
	maxSocketPath  = 100
	maxLine        = 8 << 20
	requestTimeout = 5 * time.Second

	// DefaultTimeout bounds a command that names no time of its own.
	DefaultTimeout = 30 * time.Second
	// MaxTimeout is the longest any command may run.
	MaxTimeout = 2 * time.Minute
)

// Request is one command. Session and Project say who asks; Opts carry the flags.
type Request struct {
	Cmd       string            `json:"cmd"`
	Session   string            `json:"session,omitempty"`
	Project   string            `json:"project,omitempty"`
	Args      []string          `json:"args,omitempty"`
	Opts      map[string]string `json:"opts,omitempty"`
	TimeoutMs int               `json:"timeoutMs,omitempty"`
}

// Timeout is how long the command may run.
func (r Request) Timeout() time.Duration {
	d := time.Duration(r.TimeoutMs) * time.Millisecond
	if d <= 0 {
		return DefaultTimeout
	}
	return min(d, MaxTimeout)
}

// Response is what the app printed, or why it could not.
type Response struct {
	OK    bool   `json:"ok"`
	Out   string `json:"out,omitempty"`
	Error string `json:"error,omitempty"`
}

// SocketPath is where the app listens for commands, given the state dir.
func SocketPath(stateDir string) string { return filepath.Join(stateDir, socketFile) }

// ErrNotRunning means no app is listening.
var ErrNotRunning = errors.New("agentos is not running: open the app and try again")

// Call sends req and waits for the answer.
func Call(ctx context.Context, socket string, req Request) (Response, error) {
	ctx, cancel := context.WithTimeout(ctx, req.Timeout()+requestTimeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return Response{}, ErrNotRunning
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(dl)
	}
	line, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("encoding the request: %w", err)
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return Response{}, fmt.Errorf("sending the request: %w", err)
	}
	reply, err := bufio.NewReaderSize(conn, 1<<16).ReadBytes('\n')
	if err != nil && !(errors.Is(err, io.EOF) && len(reply) > 0) {
		return Response{}, fmt.Errorf("waiting for the app: %w", err)
	}
	var resp Response
	if err := json.Unmarshal(reply, &resp); err != nil {
		return Response{}, fmt.Errorf("reading the app's answer: %w", err)
	}
	return resp, nil
}

// Handler answers one request. Its context ends when the request's time is up.
type Handler func(ctx context.Context, req Request) Response

// Server answers commands until closed.
type Server struct {
	ln net.Listener
	wg sync.WaitGroup
}

// Listen starts answering at socket with handle. Close stops it.
func Listen(socket string, handle Handler) (*Server, error) {
	if len(socket) > maxSocketPath {
		return nil, fmt.Errorf("socket path %q is longer than %d bytes", socket, maxSocketPath)
	}
	if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}
	if conn, err := net.DialTimeout("unix", socket, 100*time.Millisecond); err == nil {
		conn.Close()
		return nil, errors.New("another agentos answers commands on this socket")
	}
	_ = os.Remove(socket)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", socket, err)
	}
	_ = os.Chmod(socket, 0o600)
	s := &Server{ln: ln}
	s.wg.Go(func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			s.wg.Go(func() { serve(conn, handle) })
		}
	})
	return s, nil
}

func serve(conn net.Conn, handle Handler) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(requestTimeout))
	line, err := bufio.NewReaderSize(conn, 1<<16).ReadBytes('\n')
	if err != nil && len(line) == 0 {
		return
	}
	var req Request
	resp := Response{}
	if err := json.Unmarshal(line, &req); err != nil || len(line) > maxLine {
		resp.Error = "the request is not valid"
	} else {
		ctx, cancel := context.WithTimeout(context.Background(), req.Timeout())
		defer cancel()
		resp = handle(ctx, req)
	}
	out, err := json.Marshal(resp)
	if err != nil {
		return
	}
	_ = conn.SetWriteDeadline(time.Now().Add(requestTimeout))
	_, _ = conn.Write(append(out, '\n'))
}

// Close stops the server and waits for the commands in flight.
func (s *Server) Close() {
	_ = s.ln.Close()
	s.wg.Wait()
}

// BrowserHelp is what an agent reads first.
const BrowserHelp = `agentos browser: drive the session's browser. Every command acts on this session's tab.

  open <url>                  go to a page and wait for it to load
  back | reload               move in the page history
  url                         print the current address and title
  snapshot                    outline of the page: text, headings, and every control with a ref (e1, e2, ...)
  click <ref>                 click an element
  type <ref> <text> [--append]  replace the element's text (or add to it)
  press <key>                 Enter, Tab, Escape, ArrowDown, Backspace, or a chord like Control+a
  select <ref> <option>       pick an option of a select by its text or value
  hover <ref>                 move the pointer over an element
  scroll <up|down|ref> [px]   scroll the page, or bring an element into view
  wait <ms>                   pause
  wait-for <text> [--timeout <ms>]  wait until the text is on the page (10s by default)
  text [ref]                  plain text of the page or of an element
  eval <js>                   run JavaScript, print the JSON result
  console                     console errors and failed requests since the last call
  screenshot [--caption <text>] [--full] [ref]
                              save a PNG as evidence for this session and print its path

Refs come from the latest snapshot and stop working when you run snapshot again. After a
page change, run snapshot again before you click or type.
Use agentos show <file> [--caption <text>] for other evidence.`
