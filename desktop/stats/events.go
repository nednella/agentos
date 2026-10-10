package stats

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// Log records what the app does as events, a file per project, machine and local day. A file has one
// writer, so a data folder two Macs share through iCloud never has both appending to one file.
//
// Lines are appended in place, not by a temp file and rename: iCloud uploads a changed file whole.
// A crash can leave a broken last line, which the reader skips.
type Log struct {
	dir     string
	machine string

	mu    sync.Mutex
	cache map[string]map[string]fileDays // by project key, then file path
}

// fileDays is what one event file added up to when it had this size and mod time.
type fileDays struct {
	size int64
	mod  time.Time
	days map[string]*Day // by local day
}

// NewLog keeps the events under dir, in the files of the machine given. Nothing is read until Load or Days.
func NewLog(dir, machine string) *Log {
	return &Log{dir: dir, machine: machine, cache: map[string]map[string]fileDays{}}
}

func (l *Log) folder(key string) string { return filepath.Join(l.dir, key, eventsFolder) }

// Record appends the event to the project's file of today.
func (l *Log) Record(key string, e activity.Event) {
	line, err := json.Marshal(e)
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: encoding an event: %v\n", err)
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	name := localDay(time.Now()) + "." + l.machine + ".jsonl"
	if err := appendLine(filepath.Join(l.folder(key), name), line); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: recording an event: %v\n", err)
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

// Load gzips this machine's files of the days before yesterday, then reads every project's files.
func (l *Log) Load() {
	yesterday := localDay(time.Now().AddDate(0, 0, -1))
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range l.keys() {
		l.compress(key, yesterday)
		l.refresh(key)
	}
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

// eventFile is an event file of a project.
type eventFile struct {
	path    string
	day     string
	machine string // "" for a file from before each machine had its own
	zipped  bool
	info    os.FileInfo
}

// parseName reads "<day>.jsonl", "<day>.<machine>.jsonl" and either with ".gz". Any other name, such
// as an iCloud conflict copy "<day> 2.jsonl", is not an event file.
func parseName(name string) (day, machine string, zipped, ok bool) {
	rest, zipped := strings.CutSuffix(name, ".gz")
	rest, ok = strings.CutSuffix(rest, ".jsonl")
	if !ok {
		return "", "", false, false
	}
	day, machine, _ = strings.Cut(rest, ".")
	if _, err := time.Parse(time.DateOnly, day); err != nil {
		return "", "", false, false
	}
	if machine != "" && !ValidMachine(machine) {
		return "", "", false, false
	}
	return day, machine, zipped, true
}

// ValidMachine says whether id has the shape of a machine id: 8 characters, a-z and 2-7.
func ValidMachine(id string) bool {
	return len(id) == 8 && strings.Trim(id, "abcdefghijklmnopqrstuvwxyz234567") == ""
}

// files lists the project's event files of the days that match.
func (l *Log) files(key string, match func(day string) bool) []eventFile {
	entries, err := os.ReadDir(l.folder(key))
	if err != nil {
		return nil
	}
	var out []eventFile
	for _, e := range entries {
		day, machine, zipped, ok := parseName(e.Name())
		if !ok || !match(day) {
			continue
		}
		path := filepath.Join(l.folder(key), e.Name())
		if _, err := os.Stat(path + ".gz"); !zipped && err == nil {
			continue // a gzip that stopped before it removed the plain file
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, eventFile{path: path, day: day, machine: machine, zipped: zipped, info: info})
	}
	return out
}

// compress gzips this machine's files of the days before the day given, about ten times smaller.
// Another machine's files are left to it: two gzips of one file would conflict.
func (l *Log) compress(key, before string) {
	if l.machine == "" {
		return
	}
	for _, f := range l.files(key, func(day string) bool { return day < before }) {
		if f.machine != l.machine || f.zipped {
			continue
		}
		if err := gzipFile(f.path); err != nil {
			fmt.Fprintf(os.Stderr, "agentos: compressing events: %v\n", err)
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

// Since calls each for every event in the project's files from the local day given on.
func (l *Log) Since(key, from string, each func(activity.Event)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, f := range l.files(key, func(day string) bool { return day >= from }) {
		if err := readFile(f.path, f.zipped, each); err != nil {
			fmt.Fprintf(os.Stderr, "agentos: reading events: %v\n", err)
		}
	}
}

// refresh reads the project's files that are new or changed since the last time, and forgets the
// ones that are gone, such as a plain file its gzip replaced. The caller holds l.mu.
func (l *Log) refresh(key string) {
	old := l.cache[key]
	cur := map[string]fileDays{}
	for _, f := range l.files(key, func(string) bool { return true }) {
		if c, ok := old[f.path]; ok && c.size == f.info.Size() && c.mod.Equal(f.info.ModTime()) {
			cur[f.path] = c
			continue
		}
		days := map[string]*Day{}
		if err := readFile(f.path, f.zipped, func(e activity.Event) { apply(days, e) }); err != nil {
			fmt.Fprintf(os.Stderr, "agentos: reading events: %v\n", err)
			continue
		}
		cur[f.path] = fileDays{size: f.info.Size(), mod: f.info.ModTime(), days: days}
	}
	l.cache[key] = cur
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

// Day is what one project's agents did on one local day.
type Day struct {
	Prompts       int
	Sessions      int
	IssueSessions int   // of Sessions, those started from an issue
	WorkMs        int64 // time sessions spent working

	Ended        int   // sessions that ended
	SessionMs    int64 // how long the ended sessions ran
	PRsOpened    int
	PRsMerged    int
	PRsClosed    int   // closed without a merge
	LeadMs       int64 // of the merged PRs, the time from first seen open to merged
	IssuesFiled  int
	ClosedIssues []int // the issues whose session's PR merged
}

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
func apply(days map[string]*Day, e activity.Event) {
	day := localDay(time.UnixMilli(e.At))
	d := days[day]
	if d == nil {
		d = &Day{}
		days[day] = d
	}
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

// Days is the project's rollup by local day, from the files as they are now.
func (l *Log) Days(key string) map[string]Day {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.refresh(key)
	out := map[string]Day{}
	for _, f := range l.cache[key] {
		for day, d := range f.days {
			sum := out[day]
			sum.add(*d)
			for _, n := range d.ClosedIssues {
				if !slices.Contains(sum.ClosedIssues, n) {
					sum.ClosedIssues = append(sum.ClosedIssues, n)
				}
			}
			out[day] = sum
		}
	}
	return out
}
