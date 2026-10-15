package sessions

import (
	"bytes"
	"context"
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

// Memory is what agentos and its agents hold, in bytes, rounded up to memoryStep. It is resident memory
// summed over processes, which counts shared pages more than once: an estimate.
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

// SampleMemory reads what every process holds, and tells the front end when a rounded figure moved.
// The poll does it every sampleEvery.
func (s *Sessions) SampleMemory() {
	ctx, cancel := context.WithTimeout(s.context(), tmuxTimeout)
	defer cancel()
	out, err := s.run(ctx, "", "ps", "-axo", "pid=,ppid=,rss=")
	if err != nil {
		s.warn.Report("memory", util.FirstLine(err.Error()))
		return
	}
	server, err := s.tmux.ServerPID(ctx)
	if err != nil {
		s.warn.Report("memory", util.FirstLine(err.Error()))
		return
	}
	s.warn.Clear("memory")
	machine := s.Memory().Machine
	if machine == 0 {
		if out, err := s.run(ctx, "", "sysctl", "-n", "hw.memsize"); err == nil {
			machine, _ = strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
		}
	}
	procs := readProcesses(out)
	var browsers int64
	for _, pid := range s.browsers.PIDs() {
		browsers += procs.held(pid)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	clear(s.held)
	for _, in := range s.info {
		if in.PID > 0 {
			s.held[in.Name.String()] = roundUp(procs.held(in.PID))
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
	s.sync()
}

// processes is what ps reported: each process's resident memory in bytes, and its children.
type processes struct {
	rss      map[int]int64
	children map[int][]int
}

// readProcesses reads lines of `ps -axo pid=,ppid=,rss=`, rss in kilobytes.
func readProcesses(out []byte) processes {
	p := processes{rss: map[int]int64{}, children: map[int][]int{}}
	for line := range bytes.Lines(out) {
		f := strings.Fields(string(line))
		if len(f) != 3 {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		kb, err3 := strconv.ParseInt(f[2], 10, 64)
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		p.rss[pid] = kb << 10
		if pid != ppid {
			p.children[ppid] = append(p.children[ppid], pid)
		}
	}
	return p
}

// held is what pid and every process under it hold; 0 when pid is gone.
func (p processes) held(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	total := p.rss[pid]
	for _, child := range p.children[pid] {
		total += p.held(child)
	}
	return total
}

func roundUp(n int64) int64 { return (n + memoryStep - 1) / memoryStep * memoryStep }
