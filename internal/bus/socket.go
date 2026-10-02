package bus

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/session"
)

// unix socket paths are capped near 104 bytes on macOS.
const maxSocketPath = 100

const connReadLimit = time.Second

// Send writes rec as one JSON line to the TUI listening at socket. It gives up
// when ctx ends; the caller has already saved the state file, so a miss costs
// only a delay.
func Send(ctx context.Context, socket string, rec session.Record) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return fmt.Errorf("dialing %s: %w", socket, err)
	}
	defer conn.Close()
	if dl, ok := ctx.Deadline(); ok {
		_ = conn.SetWriteDeadline(dl)
	}
	line, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("encoding record: %w", err)
	}
	if _, err := conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing to %s: %w", socket, err)
	}
	return nil
}

// ErrInUse means another agentos already listens on the socket.
var ErrInUse = errors.New("another agentos is listening on this socket")

// Listener receives records sent by hooks.
type Listener struct {
	ln      net.Listener
	deliver func(session.Record)
	wg      sync.WaitGroup
}

// Listen starts accepting hook connections and calls deliver for each record.
// Close stops it.
func Listen(socket string, deliver func(session.Record)) (*Listener, error) {
	if len(socket) > maxSocketPath {
		return nil, fmt.Errorf("socket path %q is longer than %d bytes", socket, maxSocketPath)
	}
	if err := os.MkdirAll(DirOf(socket), 0o700); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}
	if conn, err := net.DialTimeout("unix", socket, 100*time.Millisecond); err == nil {
		conn.Close()
		return nil, ErrInUse
	}
	_ = os.Remove(socket)
	ln, err := net.Listen("unix", socket)
	if err != nil {
		return nil, fmt.Errorf("listening on %s: %w", socket, err)
	}
	_ = os.Chmod(socket, 0o600)
	l := &Listener{ln: ln, deliver: deliver}
	l.wg.Go(l.accept)
	return l, nil
}

func (l *Listener) accept() {
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			return
		}
		l.wg.Go(func() { l.serve(conn) })
	}
}

func (l *Listener) serve(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(connReadLimit))
	sc := bufio.NewScanner(conn)
	for sc.Scan() {
		var rec session.Record
		if json.Unmarshal(sc.Bytes(), &rec) == nil && rec.Session != "" {
			l.deliver(rec)
		}
	}
}

func (l *Listener) Close() {
	_ = l.ln.Close()
	l.wg.Wait()
}
