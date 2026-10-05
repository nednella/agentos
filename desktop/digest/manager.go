package digest

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/notes"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/prompts"
	"github.com/nednella/agentos/internal/util"
)

// Projects knows the current project.
type Projects interface{ Current() project.Project }

// Notes saves digest items as notes.
type Notes interface {
	Add(key, text string) (notes.Note, error)
	List(key string) ([]notes.Note, error)
}

// Manager runs the digests of the current project.
type Manager struct {
	digests  *Digests
	project  Projects
	notes    Notes
	run      run.EnvRunner
	emit     func(event string, payload any)
	stateDir string
	ctx      func() context.Context
	first    time.Duration // wait before the first automatic check
	tick     time.Duration // between checks after that
}

// NewManager runs the digests with run (claude) in the project's folder.
func NewManager(d *Digests, p Projects, n Notes, runner run.EnvRunner, emit func(string, any), stateDir string, ctx func() context.Context) *Manager {
	return &Manager{digests: d, project: p, notes: n, run: runner, emit: emit, stateDir: stateDir, ctx: ctx, first: time.Minute, tick: time.Hour}
}

// SetLoop sets how long to wait before the first automatic check, and between checks.
func (m *Manager) SetLoop(first, tick time.Duration) { m.first, m.tick = first, tick }

func (m *Manager) emitNotes(key string) {
	if list, err := m.notes.List(key); err == nil {
		m.emit("notes", scoped.Of(key, list))
	}
}

func (m *Manager) view() Digest {
	cur := m.project.Current()
	return m.digests.View(cur.Key(), cur.DigestOn())
}

func (m *Manager) emitDigest(key string) {
	if cur := m.project.Current(); cur.Key() == key {
		m.emit("digest", m.digests.View(key, cur.DigestOn()))
	}
}

// runCurrent starts the current project's digest run in the background.
func (m *Manager) runCurrent() error { return m.start(m.project.Current()) }

func (m *Manager) start(proj project.Project) error {
	key := proj.Key()
	if err := m.digests.Begin(key, time.Now()); err != nil {
		return err
	}
	m.emitDigest(key)
	go func() {
		ctx, cancel := context.WithTimeout(m.ctx(), digestTimeout)
		defer cancel()
		_, err := m.claude(ctx, proj, key)
		failure := ""
		switch {
		case ctx.Err() != nil && errors.Is(ctx.Err(), context.DeadlineExceeded):
			failure = "the run took longer than 10 minutes"
		case errors.Is(err, exec.ErrNotFound):
			failure = "claude was not found on PATH"
		case err != nil:
			failure = util.FirstLine(err.Error())
		}
		m.digests.Finish(key, time.Now(), failure)
		m.emitDigest(key)
	}()
	return nil
}

// claude runs the digest prompt in an empty folder, with the few tools the digest needs and
// only the environment claude needs: the run reads the web, so it must not see the project or its secrets.
func (m *Manager) claude(ctx context.Context, proj project.Project, key string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "agentos-digest")
	if err != nil {
		return nil, fmt.Errorf("making the digest folder: %w", err)
	}
	defer os.RemoveAll(dir)
	env := append(claudeEnv(os.Environ()), digestProjects+"="+key, "AGENTOS_SOCKET="+bus.SocketPath(m.stateDir))
	return m.run(ctx, dir, env, "claude", "-p", prompts.Digest(dependencies(proj.Dir)), "--allowedTools", digestTools)
}

// claudeEnv keeps what claude needs to start, find its login and reach its provider.
func claudeEnv(env []string) []string {
	exact := []string{"PATH", "HOME", "USER", "LOGNAME", "TMPDIR", "LANG", "LC_ALL", "SSL_CERT_FILE", "NODE_EXTRA_CA_CERTS",
		"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "no_proxy", "GOOGLE_APPLICATION_CREDENTIALS"}
	prefixes := []string{"ANTHROPIC_", "CLAUDE_", "AWS_", "CLOUD_ML_", "VERTEX_"}
	var out []string
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if slices.Contains(exact, name) || slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(name, p) }) {
			out = append(out, kv)
		}
	}
	return out
}

// Loop starts the current project's digest when it is due, checking first after m.first, then every m.tick, until ctx ends.
func (m *Manager) Loop(ctx context.Context) {
	timer := time.NewTimer(m.first)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			if cur := m.project.Current(); cur.DigestOn() && m.digests.Due(cur.Key(), time.Now()) {
				_ = m.start(cur)
			}
			timer.Reset(m.tick)
		}
	}
}

func (m *Manager) toNote(itemID string) (notes.Note, error) {
	key := m.project.Current().Key()
	item, ok := m.digests.Item(key, itemID)
	if !ok {
		return notes.Note{}, errors.New("no such digest item")
	}
	n, err := m.notes.Add(key, item.Title+"\n\n"+item.Why+"\n"+item.URL)
	if err != nil {
		return notes.Note{}, err
	}
	if err := m.digests.Update(key, itemID, func(it *DigestItem) { it.NoteID = n.ID }); err != nil {
		return notes.Note{}, err
	}
	m.emitNotes(key)
	m.emitDigest(key)
	return n, nil
}

func (m *Manager) dismiss(itemID string) error {
	key := m.project.Current().Key()
	if err := m.digests.Dismiss(key, itemID); err != nil {
		return err
	}
	m.emitDigest(key)
	return nil
}
