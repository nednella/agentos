// Package awake keeps the Mac from idle-sleeping while an agent works.
package awake

import (
	"os"
	"os/exec"
	"strconv"
	"sync"
)

// Awake holds one idle-sleep assertion at a time.
type Awake struct {
	start func() (stop func(), err error)
	emit  func(event string, payload any)

	mu   sync.Mutex
	stop func()
}

// New holds the assertion that start takes; emit tells the front end when it is taken or let go.
func New(start func() (stop func(), err error), emit func(event string, payload any)) *Awake {
	return &Awake{start: start, emit: emit}
}

// Caffeinate takes the assertion with `caffeinate -i`: idle sleep is off, the display may still sleep.
// -w ends it with the app, even when the app is killed.
func Caffeinate() (func(), error) {
	cmd := exec.Command("caffeinate", "-i", "-w", strconv.Itoa(os.Getpid()))
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}, nil
}

// Hold takes the assertion when on and lets it go when not. It does nothing when that is already so.
func (a *Awake) Hold(on bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	switch {
	case on && a.stop == nil:
		stop, err := a.start()
		if err != nil {
			return
		}
		a.stop = stop
	case !on && a.stop != nil:
		a.stop()
		a.stop = nil
	default:
		return
	}
	a.emit("awake", a.stop != nil)
}

// Held says whether the assertion is held.
func (a *Awake) Held() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stop != nil
}
