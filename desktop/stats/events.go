package stats

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/activity"
	"github.com/nednella/agentos/internal/atomicfile"
)

const eventsFolder = "events"

// Log records what the app does as events, a file per project and local day, and keeps a rollup per
// day in memory so the ledger never reads the files.
//
// Lines are appended in place, not by a temp file and rename: the data folder can be in iCloud, which
// uploads a changed file whole. A crash can leave a broken last line, which the reader skips.
type Log struct {
	dir   string
	ready chan struct{} // closed when Load has built the rollup

	mu     sync.Mutex
	loaded bool
	days   map[string]map[string]*Day // by project key, then local day
}

// NewLog keeps the events under dir. Nothing is read until Load.
func NewLog(dir string) *Log {
	return &Log{dir: dir, ready: make(chan struct{}), days: map[string]map[string]*Day{}}
}

func (l *Log) folder(key string) string { return filepath.Join(l.dir, key, eventsFolder) }

// Record appends the event to the project's file of today and adds it to the rollup.
func (l *Log) Record(key string, e activity.Event) {
	line, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: encoding an event: %v\n", err)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := appendLine(filepath.Join(l.folder(key), localDay(time.Now())+".jsonl"), line); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: recording an event: %v\n", err)
		return
	}
	if l.loaded {
		apply(l.days, key, e)
	}
}

func appendLine(path string, line []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, err = f.Write(append(line, '\n'))
	return errors.Join(err, f.Close())
}

// Load gzips the files of the days before yesterday, then builds the rollup from every project's files.
// The days before today are read first, without the lock, since nothing writes to them; then today's,
// under the lock, so an event recorded meanwhile is counted once.
func (l *Log) Load() {
	now := time.Now()
	today := localDay(now)
	yesterday := localDay(now.AddDate(0, 0, -1))
	days := map[string]map[string]*Day{}
	for _, key := range l.keys() {
		l.compress(key, yesterday)
		l.read(days, key, func(day string) bool { return day < today })
	}
	l.mu.Lock()
	for _, key := range l.keys() {
		l.read(days, key, func(day string) bool { return day >= today })
		l.addActivity(days, key)
	}
	l.days, l.loaded = days, true
	l.mu.Unlock()
	close(l.ready)
}

// keys are the projects that have a folder in dir.
func (l *Log) keys() []string {
	entries, err := os.ReadDir(l.dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}

// compress gzips the project's files of the days before the day given, about ten times smaller.
func (l *Log) compress(key, before string) {
	entries, err := os.ReadDir(l.folder(key))
	if err != nil {
		return
	}
	for _, e := range entries {
		if day, ok := strings.CutSuffix(e.Name(), ".jsonl"); ok && day < before {
			if err := gzipFile(filepath.Join(l.folder(key), e.Name())); err != nil {
				fmt.Fprintf(os.Stderr, "agentos: compressing events: %v\n", err)
			}
		}
	}
}

func gzipFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := atomicfile.Write(path+".gz", buf.Bytes(), 0o600); err != nil {
		return err
	}
	return os.Remove(path)
}

// walk calls each for every event in the project's files of the days that match.
func (l *Log) walk(key string, match func(day string) bool, each func(activity.Event)) {
	entries, err := os.ReadDir(l.folder(key))
	if err != nil {
		return
	}
	for _, e := range entries {
		path := filepath.Join(l.folder(key), e.Name())
		day, zipped := strings.CutSuffix(e.Name(), ".jsonl.gz")
		if !zipped {
			var ok bool
			if day, ok = strings.CutSuffix(e.Name(), ".jsonl"); !ok {
				continue
			}
			if _, err := os.Stat(path + ".gz"); err == nil {
				continue // a gzip that stopped before it removed the plain file
			}
		}
		if !match(day) {
			continue
		}
		if err := readFile(path, zipped, each); err != nil {
			fmt.Fprintf(os.Stderr, "agentos: reading events: %v\n", err)
		}
	}
}

// read adds the project's files of the days that match to days.
func (l *Log) read(days map[string]map[string]*Day, key string, match func(day string) bool) {
	l.walk(key, match, func(e activity.Event) { apply(days, key, e) })
}

// Since calls each for every event in the project's files from the local day given on.
func (l *Log) Since(key, from string, each func(activity.Event)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.walk(key, func(day string) bool { return day >= from }, each)
}

func readFile(path string, zipped bool, each func(activity.Event)) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var r io.Reader = f
	if zipped {
		zr, err := gzip.NewReader(f)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		defer zr.Close()
		r = zr
	}
	readEvents(r, each)
	return nil
}

// readEvents calls each for every event in r. It skips lines that do not parse.
func readEvents(r io.Reader, each func(activity.Event)) {
	sc := bufio.NewScanner(r)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var e activity.Event
		if json.Unmarshal(sc.Bytes(), &e) == nil {
			each(e)
		}
	}
}

// addActivity adds the days of the project's activity.json that come before its first event.
func (l *Log) addActivity(days map[string]map[string]*Day, key string) {
	old, err := readActivity(l.dir, key)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: %v\n", err)
		return
	}
	first := ""
	if len(days[key]) > 0 {
		first = slices.Min(slices.Collect(maps.Keys(days[key])))
	}
	for day, d := range old {
		if first == "" || day < first {
			dayOf(days, key, day).add(d)
		}
	}
}

func dayOf(days map[string]map[string]*Day, key, day string) *Day {
	if days[key] == nil {
		days[key] = map[string]*Day{}
	}
	d := days[key][day]
	if d == nil {
		d = &Day{}
		days[key][day] = d
	}
	return d
}

// add adds o to d, all but the closed issues.
func (d *Day) add(o Day) {
	d.Prompts += o.Prompts
	d.Sessions += o.Sessions
	d.IssueSessions += o.IssueSessions
	d.WorkMs += o.WorkMs
	d.Ended += o.Ended
	d.SessionMs += o.SessionMs
	d.PRsOpened += o.PRsOpened
	d.PRsMerged += o.PRsMerged
	d.PRsClosed += o.PRsClosed
	d.LeadMs += o.LeadMs
	d.IssuesFiled += o.IssuesFiled
}

// apply adds the event to its day. Kinds the rollup does not count are skipped.
func apply(days map[string]map[string]*Day, key string, e activity.Event) {
	d := dayOf(days, key, localDay(time.UnixMilli(e.At)))
	switch e.Kind {
	case activity.Prompt:
		d.Prompts++
	case activity.SessionStart:
		d.Sessions++
		if e.Issue > 0 {
			d.IssueSessions++
		}
	case activity.SessionEnd:
		d.Ended++
		d.SessionMs += e.Ms
	case activity.Worked:
		d.WorkMs += e.Ms
	case activity.PROpened:
		d.PRsOpened++
	case activity.PRMerged:
		d.PRsMerged++
		d.LeadMs += e.Ms
		if e.Issue > 0 && !slices.Contains(d.ClosedIssues, e.Issue) {
			d.ClosedIssues = append(d.ClosedIssues, e.Issue)
		}
	case activity.PRClosed:
		d.PRsClosed++
	case activity.IssueFiled:
		d.IssuesFiled++
	}
}

func localDay(t time.Time) string { return t.In(time.Local).Format(time.DateOnly) }

// Days is the project's rollup by local day. It waits for Load.
func (l *Log) Days(key string) map[string]Day {
	<-l.ready
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make(map[string]Day, len(l.days[key]))
	for day, d := range l.days[key] {
		c := *d
		c.ClosedIssues = slices.Clone(d.ClosedIssues)
		out[day] = c
	}
	return out
}
