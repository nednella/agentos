package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
)

const (
	digestEvery    = 7 * 24 * time.Hour
	digestKept     = 30
	digestPerRun   = 5
	digestTimeout  = 10 * time.Minute
	digestRetry    = 6 * time.Hour
	digestTools    = "WebSearch WebFetch Read Glob Grep Bash(agentos digest add:*)"
	digestProjects = "AGENTOS_DIGEST_PROJECT"
)

// digestPrompt is what the weekly run asks Claude to do.
const digestPrompt = `Look at what this project depends on: the package.json files, go.mod and similar files, and the Claude Code harness (CLAUDE.md, the .claude folder). Find what changed in roughly the last two weeks that is worth the owner's attention: releases, new features, deprecations and security fixes in those tools, and in Claude Code itself. Pick at most five items, the ones most relevant to how this project is built. For each one, run:

agentos digest add --title "<short title>" --why "<one line: why it matters here>" --url "<link>" --source "<the tool>"

Do nothing else. Do not edit any file.`

// DigestItem is one thing the digest found.
type DigestItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Why    string `json:"why"`
	URL    string `json:"url"`
	Source string `json:"source"`
	At     int64  `json:"at"`
	NoteID string `json:"noteId"`
}

// Digest is a project's digest as the front end sees it.
type Digest struct {
	Running   bool         `json:"running"`
	LastRunAt int64        `json:"lastRunAt"`
	NextRunAt int64        `json:"nextRunAt"`
	Error     string       `json:"error"`
	Items     []DigestItem `json:"items"`
}

type digestFile struct {
	LastRunAt     int64        `json:"lastRunAt"`
	LastAttemptAt int64        `json:"lastAttemptAt"`
	Items         []DigestItem `json:"items"`
}

// Digests keeps each project's digest in a file of the data dir.
type Digests struct {
	dir     string
	started time.Time

	mu      sync.Mutex
	running map[string]int // project key -> items added by the run so far
	errors  map[string]string
}

func newDigests(dataDir string) *Digests {
	return &Digests{dir: dataDir, started: time.Now(), running: map[string]int{}, errors: map[string]string{}}
}

func (d *Digests) path(key string) string { return filepath.Join(d.dir, key, "digest.json") }

// read needs mu.
func (d *Digests) read(key string) digestFile {
	var f digestFile
	if data, err := os.ReadFile(d.path(key)); err == nil {
		_ = json.Unmarshal(data, &f)
	}
	return f
}

// write needs mu.
func (d *Digests) write(key string, f digestFile) error {
	data, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("encoding the digest: %w", err)
	}
	return atomicfile.Write(d.path(key), data, 0o600)
}

// View is the project's digest. auto says whether it runs by itself.
func (d *Digests) View(key string, auto bool) Digest {
	d.mu.Lock()
	defer d.mu.Unlock()
	f := d.read(key)
	_, running := d.running[key]
	v := Digest{Running: running, LastRunAt: f.LastRunAt, Error: d.errors[key], Items: f.Items}
	if v.Items == nil {
		v.Items = []DigestItem{}
	}
	if auto {
		next := d.started.Add(time.Minute).UnixMilli()
		if f.LastRunAt > 0 {
			next = max(next, time.UnixMilli(f.LastRunAt).Add(digestEvery).UnixMilli())
		}
		v.NextRunAt = next
	}
	return v
}

// Due says whether an automatic run should start now.
func (d *Digests) Due(key string, now time.Time) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, running := d.running[key]; running {
		return false
	}
	f := d.read(key)
	return (f.LastRunAt == 0 || now.Sub(time.UnixMilli(f.LastRunAt)) > digestEvery) &&
		now.Sub(time.UnixMilli(f.LastAttemptAt)) > digestRetry
}

// Begin marks a run as started.
func (d *Digests) Begin(key string, now time.Time) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, running := d.running[key]; running {
		return errors.New("a digest run is already going")
	}
	d.running[key] = 0
	delete(d.errors, key)
	f := d.read(key)
	f.LastAttemptAt = now.UnixMilli()
	return d.write(key, f)
}

// Finish ends a run. A failed run keeps the old digest and records why.
func (d *Digests) Finish(key string, now time.Time, failure string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.running, key)
	if failure != "" {
		d.errors[key] = failure
		return
	}
	f := d.read(key)
	f.LastRunAt = now.UnixMilli()
	if err := d.write(key, f); err != nil {
		d.errors[key] = err.Error()
	}
}

// Add files an item found by the run in progress. It reports whether it was new.
func (d *Digests) Add(key string, item DigestItem) (bool, error) {
	item.Title, item.Why, item.URL, item.Source = strings.TrimSpace(item.Title), strings.TrimSpace(item.Why), strings.TrimSpace(item.URL), strings.TrimSpace(item.Source)
	if item.Title == "" || item.URL == "" {
		return false, errors.New("an item needs a --title and a --url")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	added, running := d.running[key]
	if !running {
		return false, errors.New("no digest run is going for this project")
	}
	if added >= digestPerRun {
		return false, fmt.Errorf("this run already added %d items", digestPerRun)
	}
	f := d.read(key)
	if slices.ContainsFunc(f.Items, func(it DigestItem) bool { return it.URL == item.URL }) {
		return false, nil
	}
	id, err := newEvidenceID()
	if err != nil {
		return false, err
	}
	item.ID, item.At = id, time.Now().UnixMilli()
	// The run's items stay at the front, in the order they were added.
	f.Items = slices.Insert(f.Items, added, item)
	f.Items = f.Items[:min(len(f.Items), digestKept)]
	d.running[key] = added + 1
	return true, d.write(key, f)
}

// Item returns one item.
func (d *Digests) Item(key, id string) (DigestItem, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, it := range d.read(key).Items {
		if it.ID == id {
			return it, true
		}
	}
	return DigestItem{}, false
}

// Update changes one item.
func (d *Digests) Update(key, id string, change func(*DigestItem)) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	f := d.read(key)
	i := slices.IndexFunc(f.Items, func(it DigestItem) bool { return it.ID == id })
	if i < 0 {
		return errors.New("no such digest item")
	}
	change(&f.Items[i])
	return d.write(key, f)
}

func (d *Digests) Dismiss(key, id string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	f := d.read(key)
	i := slices.IndexFunc(f.Items, func(it DigestItem) bool { return it.ID == id })
	if i < 0 {
		return errors.New("no such digest item")
	}
	f.Items = slices.Delete(f.Items, i, i+1)
	return d.write(key, f)
}
