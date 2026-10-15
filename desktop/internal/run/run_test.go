package run

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestExecEnvRunsAndReports(t *testing.T) {
	out, err := ExecEnv(context.Background(), "", nil, "echo", "hi")
	if err != nil || string(out) != "hi\n" {
		t.Errorf("echo = %q, %v", out, err)
	}
	_, err = ExecEnv(context.Background(), "", nil, "sh", "-c", "echo broken >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Errorf("failure = %v", err)
	}
}

func TestExecEnvWithoutArgumentsOrCommand(t *testing.T) {
	if _, err := ExecEnv(context.Background(), "", nil, "true"); err != nil {
		t.Errorf("a command with no arguments failed: %v", err)
	}
	if _, err := ExecEnv(context.Background(), "", nil, "false"); err == nil {
		t.Error("false succeeded")
	}
	if _, err := ExecEnv(context.Background(), "", nil, ""); err == nil {
		t.Error("an empty command succeeded")
	}
}

func TestExecEnvCapsOutput(t *testing.T) {
	out, err := ExecEnv(context.Background(), "", nil, "sh", "-c", "head -c 6000000 /dev/zero")
	if err != nil || len(out) != outputCap {
		t.Errorf("got %d bytes, %v; want %d", len(out), err, outputCap)
	}
}

func TestExecEnvKillsChildrenWhenCancelled(t *testing.T) {
	pidFile := t.TempDir() + "/pid"
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	// The grandchild holds stdout open and outlives the shell.
	_, err := ExecEnv(ctx, "", nil, "sh", "-c", "sleep 300 & echo $! > "+pidFile+"; wait")
	if err == nil {
		t.Error("a cancelled command succeeded")
	}
	if took := time.Since(start); took > 3*time.Second {
		t.Errorf("took %v: the grandchild kept the call open", took)
	}
	data, rerr := os.ReadFile(pidFile)
	if rerr != nil {
		t.Fatal(rerr)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	t.Cleanup(func() { _ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run() })
	deadline := time.Now().Add(2 * time.Second)
	for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if syscall.Kill(pid, 0) == nil {
		t.Errorf("the grandchild %d is still running", pid)
	}
}

func TestStreamReadsOutputAsItComes(t *testing.T) {
	out, err := Stream(context.Background(), t.TempDir(), "sh", "-c", "echo one; sleep 0.2; echo two; echo oops >&2; exit 3")
	if err != nil {
		t.Fatal(err)
	}
	sc := bufio.NewScanner(out)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if want := []string{"one", "two"}; !slices.Equal(lines, want) {
		t.Errorf("lines = %q, want %q", lines, want)
	}
	err = out.Close()
	if err == nil || !strings.Contains(err.Error(), "exit status 3") || !strings.Contains(err.Error(), "oops") {
		t.Errorf("Close = %v, want the exit status and stderr", err)
	}
}

func TestStreamCloseKillsTheCommand(t *testing.T) {
	out, err := Stream(context.Background(), t.TempDir(), "sh", "-c", "sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- out.Close() }()
	select {
	case err := <-done:
		if err == nil {
			t.Error("Close reported a killed command as a clean exit")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}
}

func TestStopAllKillsEveryRunningCommandAndItsChildren(t *testing.T) {
	dir := t.TempDir()
	out, err := Stream(context.Background(), dir, "sh", "-c", "sleep 300 & echo $! > stream; wait")
	if err != nil {
		t.Fatal(err)
	}
	ran := make(chan error, 1)
	go func() {
		_, err := ExecEnv(context.Background(), dir, nil, "sh", "-c", "sleep 300 & echo $! > exec; wait")
		ran <- err
	}()
	var pids []int
	for _, file := range []string{"stream", "exec"} {
		for range 100 {
			data, _ := os.ReadFile(dir + "/" + file)
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil {
				pids = append(pids, pid)
				break
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
	if len(pids) != 2 {
		t.Fatalf("the commands wrote pids %v", pids)
	}
	t.Cleanup(func() {
		for _, pid := range pids {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	})

	StopAll()
	for _, pid := range pids {
		deadline := time.Now().Add(2 * time.Second)
		for syscall.Kill(pid, 0) == nil && time.Now().Before(deadline) {
			time.Sleep(20 * time.Millisecond)
		}
		if syscall.Kill(pid, 0) == nil {
			t.Errorf("process %d still runs after StopAll", pid)
		}
	}
	select {
	case err := <-ran:
		if err == nil {
			t.Error("a killed command succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Error("ExecEnv did not return")
	}
	_ = out.Close()
}
