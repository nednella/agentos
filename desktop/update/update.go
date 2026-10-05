// Package update keeps the app current: it checks the releases on start and every hour,
// tells the front end about a newer one, and installs it on request.
package update

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/update"
	"github.com/nednella/agentos/internal/version"
)

const (
	checkEvery = time.Hour
	// quitDelay lets the answer to an update or relaunch reach its caller before the app goes.
	quitDelay = 300 * time.Millisecond
)

// Available is the payload of the update event: a release newer than the running one.
type Available struct {
	Version string `json:"version"`
}

// Updater checks for releases and installs one.
type Updater struct {
	run      update.Runner
	stateDir string
	bundle   string // the .app the app runs from; "" when it runs from none
	emit     func(event string, payload any)
	relaunch func(bundle string) error
	quit     func()
	first    time.Duration
	every    time.Duration

	mu     sync.Mutex
	latest update.Release
}

// New makes an updater for the app at bundle. relaunch opens it again once the app has quit.
func New(run update.Runner, stateDir, bundle string, emit func(string, any), relaunch func(string) error, quit func()) *Updater {
	return &Updater{run: run, stateDir: stateDir, bundle: bundle, emit: emit, relaunch: relaunch, quit: quit, every: checkEvery}
}

// SetLoop sets how long to wait before the first check, and between checks.
func (u *Updater) SetLoop(first, every time.Duration) { u.first, u.every = first, every }

// SetRelaunch replaces how the app is opened again, for tests.
func (u *Updater) SetRelaunch(relaunch func(string) error) { u.relaunch = relaunch }

// Loop checks for a newer release now and then every hour, until ctx ends.
func (u *Updater) Loop(ctx context.Context) {
	wait := u.first
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		u.check(ctx)
		wait = u.every
	}
}

func (u *Updater) check(ctx context.Context) {
	rel, err := update.Check(ctx, u.run, u.stateDir, 0, time.Now())
	if err != nil {
		log.Printf("agentos: checking for a newer release: %v", err)
		return
	}
	if !update.Newer(version.Version, rel.Version) {
		return
	}
	u.mu.Lock()
	changed := u.latest != rel
	u.latest = rel
	u.mu.Unlock()
	if changed {
		u.emit("update", Available{Version: rel.Version})
	}
}

// Available is the version of the newer release the last check found, or "".
func (u *Updater) Available() string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.latest.Version
}

// Apply installs the newer release and relaunches the app; sessions live in tmux and survive.
func (u *Updater) Apply(ctx context.Context) error {
	if u.bundle == "" {
		return errors.New("agentos runs from no .app bundle, so it cannot replace itself")
	}
	u.mu.Lock()
	rel := u.latest
	u.mu.Unlock()
	if rel.Version == "" {
		var err error
		if rel, err = update.Latest(ctx, u.run); err != nil {
			return err
		}
		if !update.Newer(version.Version, rel.Version) {
			return errors.New("agentos is up to date")
		}
	}
	if err := update.Install(ctx, u.run, rel, u.bundle); err != nil {
		return err
	}
	return u.Relaunch()
}

// Relaunch arranges for the app to open again, then quits it shortly after.
func (u *Updater) Relaunch() error {
	if u.bundle == "" {
		return errors.New("agentos runs from no .app bundle, so it cannot relaunch")
	}
	if err := u.relaunch(u.bundle); err != nil {
		return err
	}
	time.AfterFunc(quitDelay, u.quit)
	return nil
}
