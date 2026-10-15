package term

import (
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// treeWait is how long a killed session's processes have to exit after SIGTERM before they get SIGKILL.
const treeWait = 2 * time.Second

type proc struct{ pid, ppid, pgid int }

// processes are the live processes on the machine. A zombie is left out: it has exited and only waits for its
// parent to read its status.
func processes(ctx context.Context) (map[int]proc, error) {
	out, err := exec.CommandContext(ctx, "ps", "-axo", "pid=,ppid=,pgid=,stat=").Output()
	if err != nil {
		return nil, err
	}
	procs := map[int]proc{}
	for line := range strings.SplitSeq(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 4 || strings.HasPrefix(f[3], "Z") {
			continue
		}
		pid, err1 := strconv.Atoi(f[0])
		ppid, err2 := strconv.Atoi(f[1])
		pgid, err3 := strconv.Atoi(f[2])
		if err1 == nil && err2 == nil && err3 == nil {
			procs[pid] = proc{pid, ppid, pgid}
		}
	}
	return procs, nil
}

// tree is root and every process under it, with the members of each process group one of them leads: a child
// that left its parent's group, or whose parent exited, is still found. This process is never part of it.
func tree(procs map[int]proc, root int) map[int]proc {
	in := map[int]proc{}
	if p, ok := procs[root]; ok {
		in[root] = p
	}
	for grew := true; grew; {
		grew = false
		for pid, p := range procs {
			if _, ok := in[pid]; ok || pid == os.Getpid() {
				continue
			}
			_, parent := in[p.ppid]
			leader, led := in[p.pgid]
			if parent || led && leader.pgid == leader.pid {
				in[pid] = p
				grew = true
			}
		}
	}
	return in
}

// stopTree sends SIGTERM to every process of the tree, and SIGKILL to those still there after treeWait. A process
// counts as still there only while its pid has the same process group, so a pid taken by a new process is left
// alone.
func stopTree(ctx context.Context, procs map[int]proc) {
	signalAll(procs, syscall.SIGTERM)
	deadline := time.Now().Add(treeWait)
	for len(procs) > 0 {
		time.Sleep(50 * time.Millisecond)
		now, err := processes(ctx)
		if err != nil {
			return
		}
		for pid, p := range procs {
			if q, ok := now[pid]; !ok || q.pgid != p.pgid {
				delete(procs, pid)
			}
		}
		if time.Now().After(deadline) {
			signalAll(procs, syscall.SIGKILL)
			return
		}
	}
}

func signalAll(procs map[int]proc, sig syscall.Signal) {
	for pid := range procs {
		_ = syscall.Kill(pid, sig)
	}
}
