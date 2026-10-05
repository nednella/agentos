package control

import (
	"bufio"
	"context"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCallAndListen(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := SocketPath(dir)

	if _, err := Call(context.Background(), socket, Request{Cmd: "x"}); err != ErrNotRunning {
		t.Errorf("without a server: %v", err)
	}
	srv, err := Listen(socket, func(ctx context.Context, req Request) Response {
		switch req.Cmd {
		case "echo":
			return Response{OK: true, Out: strings.Join(req.Args, " ") + "|" + req.Opts["k"] + "|" + req.Session}
		case "slow":
			<-ctx.Done()
			return Response{Error: "timed out"}
		}
		return Response{Error: "unknown command"}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	tests := []struct {
		name string
		req  Request
		want Response
	}{
		{"echo", Request{Cmd: "echo", Args: []string{"a", "b"}, Opts: map[string]string{"k": "v"}, Session: "p/1"}, Response{OK: true, Out: "a b|v|p/1"}},
		{"unknown", Request{Cmd: "nope"}, Response{Error: "unknown command"}},
		{"deadline", Request{Cmd: "slow", TimeoutMs: 100}, Response{Error: "timed out"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := time.Now()
			got, err := Call(context.Background(), socket, tt.req)
			if err != nil || got != tt.want {
				t.Errorf("got %+v, %v; want %+v", got, err, tt.want)
			}
			if time.Since(start) > 3*time.Second {
				t.Error("took too long")
			}
		})
	}
	if _, err := Listen(socket, nil); err == nil {
		t.Error("a second server took the socket")
	}
	if _, err := Listen(filepath.Join(dir, strings.Repeat("x", 120), "c.sock"), nil); err == nil {
		t.Error("a path that is too long was accepted")
	}
}

func TestTimeout(t *testing.T) {
	tests := []struct {
		ms   int
		want time.Duration
	}{{0, DefaultTimeout}, {-1, DefaultTimeout}, {500, 500 * time.Millisecond}, {10 * 60 * 1000, MaxTimeout}}
	for _, tt := range tests {
		if got := (Request{TimeoutMs: tt.ms}).Timeout(); got != tt.want {
			t.Errorf("Timeout(%d) = %v, want %v", tt.ms, got, tt.want)
		}
	}
}

func TestCloseCancelsCommandsInFlight(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := SocketPath(dir)
	started := make(chan struct{})
	srv, err := Listen(socket, func(ctx context.Context, _ Request) Response {
		close(started)
		<-ctx.Done()
		return Response{Error: "cancelled"}
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = Call(context.Background(), socket, Request{Cmd: "slow", TimeoutMs: 60_000}) }()
	<-started

	closed := make(chan struct{})
	go func() { srv.Close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("Close waited for a command that was not cancelled")
	}
}

func TestOversizedRequestIsRefused(t *testing.T) {
	dir, err := os.MkdirTemp("", "aosc")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	socket := SocketPath(dir)
	srv, err := Listen(socket, func(context.Context, Request) Response { return Response{OK: true} })
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	conn, err := net.Dial("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(strings.Repeat("a", maxLine+1))); err != nil {
		t.Fatal(err)
	}
	reply, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || !strings.Contains(reply, "not valid") {
		t.Errorf("reply = %q, %v", reply, err)
	}
}
