package sessions

import (
	"bytes"
	"context"
	"maps"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/util"
)

const (
	sampleEvery = 10 * time.Second
	// memoryStep is what figures are rounded up to, so a session list that only moved by a few
	// megabytes is not sent again.
	memoryStep = (1 << 30) / 10
)

// Memory is what agentos and its agents use, in bytes, rounded up to memoryStep. It is each process's
// footprint summed over the processes, so memory two processes share is not counted: an estimate.
type Memory struct {
	App      int64 `json:"app"`      // the app and the commands it runs, without its browsers
	Browsers int64 `json:"browsers"` // the projects' browsers and their helper processes
	Sessions int64 `json:"sessions"` // the tmux server and everything under it: every project's agents and shells
	Total    int64 `json:"total"`
	Machine  int64 `json:"machine"` // the Mac's memory, 0 when not known
}

// Memory is the last sample.
func (s *Sessions) Memory() Memory {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.memory
}

// SampleMemory reads what every process uses, and tells the front end when a rounded figure moved.
// The poll does it every sampleEvery.
func (s *Sessions) SampleMemory() {
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	out, err := s.run(ctx, "", "ps", "-axo", "pid=,ppid=")
	if err != nil {
		s.warn.Report("memory", util.FirstLine(err.Error()))
		return
	}
	procs := readProcesses(out, footprint)
	s.mu.Lock()
	server, machine := s.serverPID, s.memory.Machine
	running := len(s.info) > 0 || len(s.shells) > 0 // tmux ends its server when its last session goes
	s.mu.Unlock()
	if !procs.has(server) {
		server = 0
		if running {
			if server, err = s.tmux.ServerPID(ctx); err != nil {
				s.warn.Report("memory", util.FirstLine(err.Error()))
				return
			}
		}
	}
	s.warn.Clear("memory")
	if machine == 0 {
		if out, err := s.run(ctx, "", "sysctl", "-n", "hw.memsize"); err == nil {
			machine, _ = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		}
	}
	var browsers int64
	for _, pid := range s.browsers.PIDs() {
		browsers += procs.held(pid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.serverPID = server
	held := map[string]int64{}
	for _, in := range s.info {
		if in.PID > 0 {
			held[in.Name.String()] = roundUp(procs.held(in.PID))
		}
	}
	m := Memory{
		App:      roundUp(max(procs.held(os.Getpid())-browsers, 0)),
		Browsers: roundUp(browsers),
		Sessions: roundUp(procs.held(server)),
		Machine:  machine,
	}
	m.Total = m.App + m.Browsers + m.Sessions
	if m != s.memory {
		s.memory = m
		s.emit("memory", m)
	}
	if !maps.Equal(held, s.held) {
		s.held = held
		s.sync()
	}
}

// processes is the process tree ps reported, and how much memory a process uses.
type processes struct {
	pids     map[int]bool
	children map[int][]int
	size     func(pid int) int64
}

// readProcesses reads lines of `ps -axo pid=,ppid=`.
func readProcesses(out []byte, size func(pid int) int64) processes {
	p := processes{pids: map[int]bool{}, children: map[int][]int{}, size: size}
	for line := range bytes.Lines(out) {
		f := strings.Fields(string(line))
		if len(f) != 2 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		if err1 != nil || err2 != nil {
			continue
		}
		p.pids[pid] = true
		if pid != ppid {
			p.children[ppid] = append(p.children[ppid], pid)
		}
	}
	return p
}

// has says whether pid is running.
func (p processes) has(pid int) bool { return p.pids[pid] }

// held is what pid and every process under it use; 0 when pid is gone.
func (p processes) held(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	total := p.size(pid)
	for _, child := range p.children[pid] {
		total += p.held(child)
	}
	return total
}

func roundUp(n int64) int64 { return (n + memoryStep - 1) / memoryStep * memoryStep }
