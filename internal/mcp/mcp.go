// Package mcp stands in for an MCP server over stdio until a tool is called, so a session that never uses the
// tools never starts the server.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// Server is what an MCP server answers to initialize and tools/list, enough to stand in for it.
type Server struct {
	Capabilities json.RawMessage `json:"capabilities"`
	ServerInfo   struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo"`
	Tools json.RawMessage `json:"tools"`
}

// browserJSON is chrome-devtools-mcp's answer to initialize and tools/list, captured from the version agentos pins.
// Its version is the one agentos runs, so a new pin needs a new capture.
//
//go:embed browser.json
var browserJSON []byte

// Browser stands in for chrome-devtools-mcp.
var Browser = func() Server {
	var s Server
	if err := json.Unmarshal(browserJSON, &s); err != nil {
		panic(err)
	}
	return s
}()

// None is a server with no tools, for a session that has no browser.
var None = func() Server {
	s := Server{Capabilities: json.RawMessage(`{"tools":{}}`), Tools: json.RawMessage(`[]`)}
	s.ServerInfo.Name, s.ServerInfo.Version = "agentos", "0"
	return s
}()

type message struct {
	ID     json.RawMessage `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// stopWait is how long the real server has to exit after its input closes or it gets SIGTERM.
const stopWait = 3 * time.Second

// startID is the id of the initialize request replayed to the real server, whose answer the client already had.
const startID = `"agentos-start"`

// Serve answers initialize, tools/list and ping from s, and starts the real server, argv with env added to its
// environment, at the first other request. The real server then gets the client's initialize and every message
// from that one on, and its output goes to out. Serve returns when in ends, the real server exits or ctx ends,
// and stops the real server's whole process group first.
func Serve(ctx context.Context, s Server, argv, env []string, in io.Reader, out, errOut io.Writer) error {
	lines := readLines(in)
	var initialize []byte
	for {
		var line []byte
		select {
		case <-ctx.Done():
			return nil
		case l, ok := <-lines:
			if !ok {
				return nil
			}
			line = l
		}
		var m message
		if json.Unmarshal(line, &m) != nil {
			continue
		}
		var err error
		switch {
		case m.Method == "initialize":
			initialize = line
			err = reply(out, m.ID, initResult(s, m.Params))
		case m.Method == "tools/list":
			err = reply(out, m.ID, map[string]json.RawMessage{"tools": s.Tools})
		case m.Method == "ping":
			err = reply(out, m.ID, struct{}{})
		case m.ID == nil:
			// A notification, such as initialized, means nothing to a server that is not running yet.
		default:
			real, startErr := start(argv, env, initialize, errOut)
			if startErr == nil {
				return real.relay(ctx, line, lines, out)
			}
			fmt.Fprintf(errOut, "agentos mcp: %v\n", startErr)
			err = replyError(out, m.ID, "the tools could not start: "+startErr.Error())
		}
		if err != nil {
			return err
		}
	}
}

// readLines sends each non-empty line of in, then closes.
func readLines(in io.Reader) <-chan []byte {
	lines := make(chan []byte)
	go func() {
		defer close(lines)
		reader := bufio.NewReader(in)
		for {
			line, err := reader.ReadBytes('\n')
			if len(bytes.TrimSpace(line)) > 0 {
				lines <- line
			}
			if err != nil {
				return
			}
		}
	}()
	return lines
}

// initResult is the answer to initialize. It takes the client's protocol version, as the real server does for a
// version it knows.
func initResult(s Server, params json.RawMessage) any {
	var p struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	_ = json.Unmarshal(params, &p)
	return map[string]any{"protocolVersion": p.ProtocolVersion, "capabilities": s.Capabilities, "serverInfo": s.ServerInfo}
}

type realServer struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	exited chan struct{}
	once   sync.Once
}

// start runs argv in its own process group and gives it the client's initialize, waiting for its answer.
func start(argv, env []string, initialize []byte, errOut io.Writer) (*realServer, error) {
	if len(argv) == 0 {
		return nil, errors.New("no server command")
	}
	if initialize == nil {
		return nil, errors.New("a request came before initialize")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	// npx refuses to run under a package.json whose devEngines names another package manager, and the server
	// gets the project through --workspace, so it starts where no package.json is above it.
	cmd.Dir = "/"
	cmd.Env = append(os.Environ(), env...)
	cmd.Stderr = errOut
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	r := &realServer{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout), exited: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(r.exited)
	}()
	if err := r.handshake(initialize); err != nil {
		r.stop()
		return nil, err
	}
	return r, nil
}

func (r *realServer) handshake(initialize []byte) error {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(initialize, &m); err != nil {
		return err
	}
	m["id"] = json.RawMessage(startID)
	replay, _ := json.Marshal(m)
	if _, err := r.stdin.Write(append(replay, '\n')); err != nil {
		return err
	}
	for {
		line, err := r.stdout.ReadBytes('\n')
		if err != nil {
			return errors.New("the server stopped before it answered initialize")
		}
		var answer message
		if json.Unmarshal(line, &answer) == nil && string(answer.ID) == startID {
			break
		}
	}
	_, err := io.WriteString(r.stdin, `{"jsonrpc":"2.0","method":"notifications/initialized"}`+"\n")
	return err
}

// relay sends first and then every line to the real server, and the server's output to out, until the lines end,
// the server's output ends or ctx ends. It then stops the server.
func (r *realServer) relay(ctx context.Context, first []byte, lines <-chan []byte, out io.Writer) error {
	copied := make(chan error, 1)
	go func() {
		_, err := io.Copy(out, r.stdout)
		copied <- err
	}()
	defer r.stop()
	if _, err := r.stdin.Write(first); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-copied:
			return err
		case line, ok := <-lines:
			if !ok {
				return nil
			}
			if _, err := r.stdin.Write(line); err != nil {
				return err
			}
		}
	}
}

// stop ends the real server's process group: SIGTERM, then SIGKILL for anything left once the server exits or
// stopWait passes. npx runs the server as a child of its own, so the group, not the one process, has to go.
func (r *realServer) stop() {
	r.once.Do(func() {
		_ = r.stdin.Close()
		pgid := r.cmd.Process.Pid
		_ = syscall.Kill(-pgid, syscall.SIGTERM)
		select {
		case <-r.exited:
		case <-time.After(stopWait):
		}
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	})
}

func reply(out io.Writer, id json.RawMessage, result any) error {
	return write(out, map[string]any{"jsonrpc": "2.0", "id": id, "result": result})
}

func replyError(out io.Writer, id json.RawMessage, text string) error {
	return write(out, map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32603, "message": text}})
}

func write(out io.Writer, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}
