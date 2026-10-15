package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestMain runs this test binary as the real server when the tests start it with MCP_FAKE_LOG set: it logs each
// message it gets, answers initialize and tools/call, and leaves a child that ignores SIGTERM in its process group.
func TestMain(m *testing.M) {
	if log := os.Getenv("MCP_FAKE_LOG"); log != "" {
		fakeServer(log)
		return
	}
	os.Exit(m.Run())
}

func fakeServer(log string) {
	child := exec.Command("sh", "-c", `trap "" TERM; sleep 60`)
	if child.Start() != nil {
		os.Exit(1)
	}
	f, _ := os.OpenFile(log, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	fmt.Fprintf(f, "child %d\n", child.Process.Pid)
	in := bufio.NewScanner(os.Stdin)
	for in.Scan() {
		var m message
		_ = json.Unmarshal(in.Bytes(), &m)
		fmt.Fprintln(f, strings.TrimSpace(m.Method+" "+string(m.ID)))
		switch m.Method {
		case "initialize":
			fmt.Printf(`{"jsonrpc":"2.0","id":%s,"result":{"protocolVersion":"2025-06-18"}}`+"\n", m.ID)
		case "tools/call":
			fmt.Printf(`{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"from the real server"}]}}`+"\n", m.ID)
		}
	}
	// Like a server that ignores its input closing: only a signal stops it.
	time.Sleep(time.Hour)
}

type client struct {
	t   *testing.T
	in  *io.PipeWriter
	out *bufio.Reader
	err chan error
}

func serve(t *testing.T, argv []string) *client {
	t.Helper()
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	c := &client{t: t, in: inW, out: bufio.NewReader(outR), err: make(chan error, 1)}
	go func() {
		c.err <- Serve(context.Background(), Browser, argv, inR, outW, io.Discard)
		_ = outW.Close()
	}()
	return c
}

func (c *client) send(line string) {
	c.t.Helper()
	if _, err := io.WriteString(c.in, line+"\n"); err != nil {
		c.t.Fatal(err)
	}
}

func (c *client) answer() map[string]json.RawMessage {
	c.t.Helper()
	line, err := c.out.ReadBytes('\n')
	if err != nil {
		c.t.Fatalf("no answer: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(line, &m); err != nil {
		c.t.Fatalf("answer %q: %v", line, err)
	}
	return m
}

func (c *client) close() {
	c.t.Helper()
	_ = c.in.Close()
	select {
	case err := <-c.err:
		if err != nil {
			c.t.Errorf("Serve = %v", err)
		}
	case <-time.After(2 * stopWait):
		c.t.Fatal("Serve did not return after its input closed")
	}
}

const initialize = `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"claude-code","version":"2"}}}`

func fakeArgv(t *testing.T) (argv []string, log string) {
	log = filepath.Join(t.TempDir(), "log")
	return []string{"/usr/bin/env", "MCP_FAKE_LOG=" + log, os.Args[0]}, log
}

func TestServeListsTheToolsWithoutStartingTheServer(t *testing.T) {
	argv, log := fakeArgv(t)
	c := serve(t, argv)
	c.send(initialize)
	init := c.answer()
	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct{ Name, Version string }
	}
	_ = json.Unmarshal(init["result"], &result)
	if string(init["id"]) != "0" || result.ProtocolVersion != "2025-06-18" || result.ServerInfo.Name != "chrome_devtools" || result.ServerInfo.Version == "" {
		t.Errorf("initialize answer = %s", init["result"])
	}
	c.send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	c.send(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`)
	var tools struct{ Tools []struct{ Name string } }
	_ = json.Unmarshal(c.answer()["result"], &tools)
	if len(tools.Tools) == 0 || tools.Tools[0].Name != "click" {
		t.Errorf("tools = %v", tools.Tools)
	}
	c.send(`{"jsonrpc":"2.0","id":2,"method":"ping"}`)
	if ping := c.answer(); string(ping["id"]) != "2" || string(ping["result"]) != "{}" {
		t.Errorf("ping answer = %v", ping)
	}
	c.close()
	if _, err := os.Stat(log); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the server started: %v", err)
	}
}

func TestServeStartsTheServerAtTheFirstToolCallAndStopsItsGroup(t *testing.T) {
	argv, log := fakeArgv(t)
	c := serve(t, argv)
	c.send(initialize)
	c.answer()
	c.send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	c.send(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_pages","arguments":{}}}`)
	if call := c.answer(); string(call["id"]) != "1" || !strings.Contains(string(call["result"]), "from the real server") {
		t.Errorf("tools/call answer = %v", call)
	}
	c.send(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"list_pages","arguments":{}}}`)
	if call := c.answer(); string(call["id"]) != "2" {
		t.Errorf("second tools/call answer = %v", call)
	}
	c.close()

	data, _ := os.ReadFile(log)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{`initialize "agentos-start"`, "notifications/initialized", "tools/call 1", "tools/call 2"}
	if len(lines) != 5 || strings.Join(lines[1:], "|") != strings.Join(want, "|") {
		t.Fatalf("the server got %q, want a child line then %q", lines, want)
	}
	child, _ := strconv.Atoi(strings.TrimPrefix(lines[0], "child "))
	if !gone(child) {
		t.Errorf("the server's child %d, which ignores SIGTERM, is still running", child)
		_ = syscall.Kill(child, syscall.SIGKILL)
	}
}

func TestServeAnswersAnErrorWhenTheServerCannotStart(t *testing.T) {
	c := serve(t, []string{filepath.Join(t.TempDir(), "missing")})
	c.send(initialize)
	c.answer()
	c.send(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"list_pages"}}`)
	if call := c.answer(); string(call["id"]) != "1" || call["error"] == nil {
		t.Errorf("tools/call answer = %v", call)
	}
	c.send(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`)
	if list := c.answer(); list["result"] == nil {
		t.Errorf("tools/list after the failure = %v", list)
	}
	c.close()
}

// gone waits for pid to stop existing. Its parent has died by then, so launchd reaps it.
func gone(pid int) bool {
	for range 50 {
		if syscall.Kill(pid, 0) != nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
