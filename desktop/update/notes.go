package update

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/nednella/agentos"
	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/update"
)

const seenFile = "seen-version"

// Release is one version's notes from the changelog.
type Release struct {
	Version  string    `json:"version"`
	Date     string    `json:"date"` // "2026-10-07"
	Sections []Section `json:"sections"`
}

// Section is a group of changes, such as "Features" or "Bug Fixes".
type Section struct {
	Title   string   `json:"title"`
	Changes []Change `json:"changes"`
}

// Change is one line of a release.
type Change struct {
	Scope string `json:"scope"` // "desktop", or "" when the commit named none
	Text  string `json:"text"`
	Issue int    `json:"issue"` // 0 when it closed none
	URL   string `json:"url"`   // the issue's page, when it has one
}

var (
	releaseHeading = regexp.MustCompile(`^## \[?(\d+\.\d+\.\d+)\]?(?:\([^)]*\))?(?: \((\d{4}-\d{2}-\d{2})\))?`)
	scoped         = regexp.MustCompile(`^\*\*([^*]+):\*\* `)
	issueRef       = regexp.MustCompile(`\[#(\d+)\]\(([^)]+)\)`)
)

// ParseChangelog reads release-please's CHANGELOG.md into its releases, newest first.
func ParseChangelog(text string) []Release {
	var releases []Release
	for _, line := range strings.Split(text, "\n") {
		if m := releaseHeading.FindStringSubmatch(line); m != nil {
			releases = append(releases, Release{Version: m[1], Date: m[2]})
			continue
		}
		if len(releases) == 0 {
			continue
		}
		r := &releases[len(releases)-1]
		if title, ok := strings.CutPrefix(line, "### "); ok {
			r.Sections = append(r.Sections, Section{Title: strings.TrimSpace(title)})
			continue
		}
		item, ok := strings.CutPrefix(line, "* ")
		if !ok || len(r.Sections) == 0 {
			continue
		}
		s := &r.Sections[len(r.Sections)-1]
		s.Changes = append(s.Changes, parseChange(item))
	}
	return releases
}

func parseChange(item string) Change {
	var c Change
	if m := issueRef.FindStringSubmatch(item); m != nil {
		c.Issue, _ = strconv.Atoi(m[1])
		c.URL = m[2]
	}
	if m := scoped.FindStringSubmatch(item); m != nil {
		c.Scope = m[1]
		item = item[len(m[0]):]
	}
	if i := strings.Index(item, " (["); i >= 0 {
		item = item[:i]
	}
	c.Text = item
	return c
}

// PatchNotes returns the releases since the version the app last ran, newest first, and
// remembers the running one so they show once. On the first run it returns only the running
// release; a dev build has no notes.
func (u *Updater) PatchNotes() ([]Release, error) {
	path := filepath.Join(u.stateDir, seenFile)
	raw, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	seen := strings.TrimSpace(string(raw))
	notes := []Release{}
	if seen == u.current || u.current == "dev" {
		return notes, nil
	}
	for _, r := range ParseChangelog(agentos.Changelog) {
		since := update.Newer(seen, r.Version) && !update.Newer(u.current, r.Version)
		if since || (seen == "" && r.Version == u.current) {
			notes = append(notes, r)
		}
	}
	return notes, atomicfile.Write(path, []byte(u.current+"\n"), 0o644)
}
