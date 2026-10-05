// Package terminal streams tmux sessions to the front end, one PTY per opened session.
package terminal

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"

	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/term"
)

const (
	flushEvery = 8 * time.Millisecond
	flushSize  = 32 << 10
	oscLimit   = 1 << 20
)

var oscClipboard = []byte("\x1b]52;")

// Terms streams tmux sessions to the front end, one PTY per opened session.
type Terms struct {
	tmux *term.Tmux
	emit func(event string, payload any)
	clip func(text string)

	mu      sync.Mutex
	streams map[string]*stream
}

// New streams the sessions of tmux; emit sends the output and clip gets what an agent copies.
func New(tmux *term.Tmux, emit func(string, any), clip func(string)) *Terms {
	return &Terms{tmux: tmux, emit: emit, clip: clip, streams: map[string]*stream{}}
}

// Open attaches a PTY of the given size to the session. Opening an open session
// attaches afresh, which makes tmux redraw the whole screen.
func (t *Terms) Open(id string, cols, rows int) error {
	name, err := session.ParseName(id)
	if err != nil {
		return err
	}
	argv := t.tmux.AttachArgv(name)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(t.tmux.Env(), "TERM=xterm-256color", "COLORTERM=truecolor")
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)})
	if err != nil {
		return fmt.Errorf("attaching to %s: %w", id, err)
	}
	st := &stream{id: id, pty: f, cmd: cmd, emit: t.emit, clip: t.clip}
	t.mu.Lock()
	old := t.streams[id]
	t.streams[id] = st
	t.mu.Unlock()
	if old != nil {
		old.close()
	}
	go func() {
		st.pump()
		t.mu.Lock()
		if t.streams[id] == st {
			delete(t.streams, id)
		}
		t.mu.Unlock()
		if !st.closed() {
			t.emit("term:exit", map[string]string{"id": id})
		}
	}()
	return nil
}

func (t *Terms) get(id string) *stream {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.streams[id]
}

func (t *Terms) Write(id, data string) error {
	st := t.get(id)
	if st == nil {
		return fmt.Errorf("terminal %s is not open", id)
	}
	if _, err := io.WriteString(st.pty, data); err != nil {
		return fmt.Errorf("writing to %s: %w", id, err)
	}
	return nil
}

func (t *Terms) Resize(id string, cols, rows int) error {
	st := t.get(id)
	if st == nil {
		return fmt.Errorf("terminal %s is not open", id)
	}
	if err := pty.Setsize(st.pty, &pty.Winsize{Cols: uint16(cols), Rows: uint16(rows)}); err != nil {
		return fmt.Errorf("resizing %s: %w", id, err)
	}
	return nil
}

func (t *Terms) Close(id string) {
	t.mu.Lock()
	st := t.streams[id]
	delete(t.streams, id)
	t.mu.Unlock()
	if st != nil {
		st.close()
	}
}

func (t *Terms) CloseAll() {
	t.mu.Lock()
	all := t.streams
	t.streams = map[string]*stream{}
	t.mu.Unlock()
	for _, st := range all {
		st.close()
	}
}

// stream is one attach client in a PTY.
type stream struct {
	id   string
	pty  *os.File
	cmd  *exec.Cmd
	emit func(string, any)
	clip func(string)

	mu      sync.Mutex
	pending []byte
	timer   *time.Timer
	done    bool
	osc     []byte // the unfinished tail of an OSC 52 sequence
}

func (s *stream) closed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.done
}

func (s *stream) close() {
	s.mu.Lock()
	s.done = true
	s.mu.Unlock()
	_ = s.pty.Close()
	_ = s.cmd.Process.Kill()
}

// pump forwards output until the attach client exits.
func (s *stream) pump() {
	defer func() {
		s.mu.Lock()
		s.flush()
		s.mu.Unlock()
		_ = s.pty.Close()
		_ = s.cmd.Wait()
	}()
	buf := make([]byte, flushSize)
	for {
		n, err := s.pty.Read(buf)
		if n > 0 {
			s.add(buf[:n])
		}
		if err != nil {
			return
		}
	}
}

func (s *stream) add(b []byte) {
	s.copyOSC52(b)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending = append(s.pending, b...)
	switch {
	case len(s.pending) >= flushSize:
		s.flush()
	case s.timer == nil:
		s.timer = time.AfterFunc(flushEvery, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			s.flush()
		})
	}
}

// flush needs mu.
func (s *stream) flush() {
	if s.timer != nil {
		s.timer.Stop()
		s.timer = nil
	}
	if len(s.pending) == 0 || s.done {
		s.pending = s.pending[:0]
		return
	}
	s.emit("term:data", map[string]string{"id": s.id, "data": base64.StdEncoding.EncodeToString(s.pending)})
	s.pending = s.pending[:0]
}

// copyOSC52 puts the text of any OSC 52 "set clipboard" sequence in b on the
// system clipboard. A sequence may be split across reads. Only the pump calls it.
func (s *stream) copyOSC52(b []byte) {
	data := append(s.osc, b...)
	s.osc = nil
	for {
		start := bytes.Index(data, oscClipboard)
		if start < 0 {
			// Keep a possible start of the marker that the next read completes.
			s.osc = append([]byte(nil), data[max(len(data)-len(oscClipboard)+1, 0):]...)
			return
		}
		body := data[start+len(oscClipboard):]
		end, termLen := oscEnd(body)
		if end < 0 {
			if len(body) < oscLimit {
				s.osc = append([]byte(nil), data[start:]...)
			}
			return
		}
		if _, b64, ok := bytes.Cut(body[:end], []byte(";")); ok {
			if text, err := base64.StdEncoding.DecodeString(string(b64)); err == nil && len(text) > 0 {
				s.clip(string(text))
			}
		}
		data = body[end+termLen:]
	}
}

// oscEnd finds the BEL or ESC \ that ends an OSC body.
func oscEnd(body []byte) (end, termLen int) {
	bel := bytes.IndexByte(body, 0x07)
	st := bytes.Index(body, []byte("\x1b\\"))
	switch {
	case bel >= 0 && (st < 0 || bel < st):
		return bel, 1
	case st >= 0:
		return st, 2
	}
	return -1, 0
}
