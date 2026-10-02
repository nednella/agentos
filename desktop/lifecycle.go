package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
)

const (
	prPollEvery  = 3 * time.Minute
	cleanupsKept = 100
	gitTimeout   = 30 * time.Second
)

// PR is a pull request as the front end sees it.
type PR struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	State     string `json:"state"`
	Checks    string `json:"checks"`
	Comments  int    `json:"comments"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Cleanup is one entry of the clean-up log.
type Cleanup struct {
	At           int64    `json:"at"`
	SessionTitle string   `json:"sessionTitle"`
	Issue        int      `json:"issue"`
	PR           int      `json:"pr"`
	Status       string   `json:"status"`
	Removed      []string `json:"removed"`
	Reason       string   `json:"reason"`
}

// target is a session that works on an issue, with the project it belongs to.
type target struct {
	id, title string
	issue     int
	proj      project.Project
}

// track is what is known about the pull request and clean-up of one session.
type track struct {
	worktree  string
	pr        *PR
	attention string
	cleanup   string
	reason    string
	autoTried bool
	sawLive   bool // the PR was seen draft or open while this session ran
}

// ack is how much of a PR the user has seen.
type ack struct {
	Comments int  `json:"comments"`
	Checks   bool `json:"checks"` // failing checks were seen
}

// Lifecycle follows each issue session's branch, worktree and pull request, and
// cleans up after the session when the PR is merged or closed.
type Lifecycle struct {
	run       runner
	dataDir   string
	stateDir  string
	emit      func(event string, payload any)
	sessions  *Sessions
	closeTerm func(id string)
	release   func(ctx context.Context, id string) []string // closes the session's browser tab, removes its evidence

	pollMu sync.Mutex // one poll at a time

	mu     sync.Mutex
	tracks map[string]*track
	acks   map[string]map[string]ack  // project key -> PR number -> ack
	live   map[string]map[string]bool // project key -> session id -> its PR was seen open
}

func newLifecycle(run runner, dataDir, stateDir string, s *Sessions, emit func(string, any), closeTerm func(string)) *Lifecycle {
	return &Lifecycle{
		run: run, dataDir: dataDir, stateDir: stateDir, sessions: s, emit: emit, closeTerm: closeTerm,
		tracks: map[string]*track{}, acks: map[string]map[string]ack{}, live: map[string]map[string]bool{},
	}
}

// sessionFields is what a session row shows of its track.
type sessionFields struct {
	worktree, attention, cleanup, reason string
	pr                                   *PR
}

func (l *Lifecycle) fields(id string) sessionFields {
	if l == nil {
		return sessionFields{}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.tracks[id]
	if t == nil {
		return sessionFields{}
	}
	f := sessionFields{worktree: t.worktree, attention: t.attention, cleanup: t.cleanup, reason: t.reason}
	if t.pr != nil {
		pr := *t.pr
		f.pr = &pr
	}
	return f
}

func (l *Lifecycle) forget(id string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tracks, id)
	key := projectKeyOf(id)
	if live := l.liveFile(key); live[id] {
		delete(live, id)
		l.saveLive(key)
	}
}

// Run polls until ctx ends.
func (l *Lifecycle) Run(ctx context.Context) {
	tick := time.NewTicker(prPollEvery)
	defer tick.Stop()
	for {
		l.Poll(ctx)
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// Poll looks up the worktree and pull request of every issue session.
func (l *Lifecycle) Poll(ctx context.Context) {
	l.pollMu.Lock()
	defer l.pollMu.Unlock()
	targets := l.sessions.issueTargets()
	trees := map[string]worktrees{}
	var auto []string
	for _, t := range targets {
		if _, ok := trees[t.proj.Dir]; !ok {
			trees[t.proj.Dir], _ = l.worktrees(ctx, t.proj.Dir)
		}
		branch := t.proj.BranchFor(t.issue)
		pr, err := l.findPR(ctx, t.proj.Dir, branch)
		if err != nil {
			continue
		}
		if l.update(t, trees[t.proj.Dir].byBranch[branch], pr) {
			auto = append(auto, t.id)
		}
	}
	for _, id := range auto {
		go func() { _ = l.Cleanup(ctx, id, false) }()
	}
	l.sessions.changed()
}

// update stores what a poll found. It reports whether the session should be cleaned up now.
func (l *Lifecycle) update(t target, worktree string, pr *PR) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	key := t.proj.Key()
	tr := l.tracks[t.id]
	if tr == nil {
		tr = &track{sawLive: l.liveFile(key)[t.id]}
		l.tracks[t.id] = tr
	}
	tr.worktree, tr.pr = worktree, pr
	if pr != nil && (pr.State == "draft" || pr.State == "open") && !tr.sawLive {
		tr.sawLive = true
		l.liveFile(key)[t.id] = true
		l.saveLive(key)
	}
	attention := ""
	if pr != nil {
		attention = l.attentionFor(key, pr)
	}
	if attention != "" && attention != tr.attention {
		l.emit("attention", map[string]string{"id": t.id, "state": "pr"})
	}
	tr.attention = attention
	switch {
	case pr == nil || (pr.State != "closed" && pr.State != "merged"):
		if tr.cleanup == "ask" {
			tr.cleanup = ""
		}
	case pr.State == "closed" && tr.cleanup == "":
		tr.cleanup = "ask"
	case pr.State == "merged" && tr.cleanup == "" && !tr.sawLive:
		// Merged before this session saw it open: the branch may be old, so the user decides.
		tr.cleanup = "ask"
	case pr.State == "merged" && tr.cleanup == "" && !tr.autoTried:
		tr.autoTried = true
		return true
	}
	return false
}

func (l *Lifecycle) livePath(key string) string {
	return filepath.Join(l.dataDir, "prs", key+".live.json")
}

// liveFile is the sessions of a project whose PR was seen open, by session id. It needs mu.
func (l *Lifecycle) liveFile(key string) map[string]bool {
	if live, ok := l.live[key]; ok {
		return live
	}
	live := map[string]bool{}
	if data, err := os.ReadFile(l.livePath(key)); err == nil {
		_ = json.Unmarshal(data, &live)
	}
	l.live[key] = live
	return live
}

// saveLive needs mu.
func (l *Lifecycle) saveLive(key string) {
	data, err := json.Marshal(l.live[key])
	if err == nil {
		err = atomicfile.Write(l.livePath(key), data, 0o600)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: saving PR state: %v\n", err)
	}
}

// attentionFor says what about the PR needs the user: failing checks, or more
// comments than they last saw. It needs mu.
func (l *Lifecycle) attentionFor(key string, pr *PR) string {
	acks := l.ackFile(key)
	n := strconv.Itoa(pr.Number)
	a, seen := acks[n]
	if !seen {
		a = ack{Comments: pr.Comments}
		acks[n] = a
		l.saveAcks(key)
	}
	if pr.Checks != "failing" && a.Checks {
		a.Checks = false
		acks[n] = a
		l.saveAcks(key)
	}
	switch {
	case pr.Checks == "failing" && !a.Checks:
		return "checks"
	case pr.Comments > a.Comments:
		return "comments"
	}
	return ""
}

// ackFile needs mu.
func (l *Lifecycle) ackFile(key string) map[string]ack {
	if acks, ok := l.acks[key]; ok {
		return acks
	}
	acks := map[string]ack{}
	if data, err := os.ReadFile(l.ackPath(key)); err == nil {
		_ = json.Unmarshal(data, &acks)
	}
	l.acks[key] = acks
	return acks
}

func (l *Lifecycle) ackPath(key string) string { return filepath.Join(l.dataDir, "prs", key+".json") }

// saveAcks needs mu.
func (l *Lifecycle) saveAcks(key string) {
	data, err := json.Marshal(l.acks[key])
	if err == nil {
		err = atomicfile.Write(l.ackPath(key), data, 0o600)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: saving PR acknowledgements: %v\n", err)
	}
}

// Ack marks the session's PR comments and failing checks as seen.
func (l *Lifecycle) Ack(id string) error {
	t, err := l.sessions.target(id)
	if err != nil {
		return err
	}
	l.mu.Lock()
	tr := l.tracks[id]
	if tr == nil || tr.pr == nil {
		l.mu.Unlock()
		return nil
	}
	key := t.proj.Key()
	l.ackFile(key)[strconv.Itoa(tr.pr.Number)] = ack{Comments: tr.pr.Comments, Checks: tr.pr.Checks == "failing"}
	l.saveAcks(key)
	tr.attention = ""
	l.mu.Unlock()
	l.sessions.changed()
	return nil
}

// worktrees are a project's git worktrees by branch; main is the project folder's own.
type worktrees struct {
	main     string
	byBranch map[string]string
}

func (l *Lifecycle) git(ctx context.Context, dir string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, gitTimeout)
	defer cancel()
	return l.run(ctx, dir, "git", args...)
}

func (l *Lifecycle) worktrees(ctx context.Context, dir string) (worktrees, error) {
	out, err := l.git(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return worktrees{}, err
	}
	return parseWorktrees(string(out)), nil
}

func parseWorktrees(out string) worktrees {
	w := worktrees{byBranch: map[string]string{}}
	path := ""
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			path = strings.TrimPrefix(line, "worktree ")
			if w.main == "" {
				w.main = path
			}
		case strings.HasPrefix(line, "branch refs/heads/"):
			w.byBranch[strings.TrimPrefix(line, "branch refs/heads/")] = path
		}
	}
	return w
}

func samePath(a, b string) bool {
	real := func(p string) string {
		if r, err := filepath.EvalSymlinks(p); err == nil {
			return r
		}
		return filepath.Clean(p)
	}
	return real(a) == real(b)
}

func (l *Lifecycle) findPR(ctx context.Context, dir, branch string) (*PR, error) {
	ctx, cancel := context.WithTimeout(ctx, ghTimeout)
	defer cancel()
	out, err := l.run(ctx, dir, "gh", "pr", "list", "--head", branch, "--state", "all", "--json",
		"number,url,state,isDraft,statusCheckRollup,comments,reviews,updatedAt", "--limit", "1")
	if err != nil {
		return nil, fmt.Errorf("finding the PR of %s: %w", branch, err)
	}
	return parsePR(out)
}

func parsePR(data []byte) (*PR, error) {
	var raw []struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
		Rollup  []struct {
			Status     string `json:"status"`
			Conclusion string `json:"conclusion"`
			State      string `json:"state"`
		} `json:"statusCheckRollup"`
		Comments  []json.RawMessage `json:"comments"`
		Reviews   []json.RawMessage `json:"reviews"`
		UpdatedAt time.Time         `json:"updatedAt"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("reading gh pr list: %w", err)
	}
	if len(raw) == 0 {
		return nil, nil
	}
	r := raw[0]
	pr := &PR{Number: r.Number, URL: r.URL, Comments: len(r.Comments) + len(r.Reviews), UpdatedAt: millis(r.UpdatedAt), Checks: "none"}
	switch {
	case r.State == "MERGED":
		pr.State = "merged"
	case r.State == "CLOSED":
		pr.State = "closed"
	case r.IsDraft:
		pr.State = "draft"
	default:
		pr.State = "open"
	}
	for _, c := range r.Rollup {
		switch {
		case slices.Contains([]string{"FAILURE", "TIMED_OUT", "CANCELLED", "ACTION_REQUIRED", "STARTUP_FAILURE", "ERROR"}, c.Conclusion),
			c.State == "FAILURE" || c.State == "ERROR":
			pr.Checks = "failing"
		case pr.Checks != "failing" && (c.State == "PENDING" || c.State == "EXPECTED" || (c.State == "" && c.Status != "" && c.Status != "COMPLETED")):
			pr.Checks = "pending"
		case pr.Checks == "none":
			pr.Checks = "passing"
		}
	}
	return pr, nil
}

// Cleanup removes a session's worktree, branch and temp files and ends the session.
// Without force it first checks that nothing would be lost.
func (l *Lifecycle) Cleanup(ctx context.Context, id string, force bool) error {
	t, err := l.sessions.target(id)
	if err != nil {
		return err
	}
	l.mu.Lock()
	tr := l.tracks[id]
	if tr == nil {
		tr = &track{}
		l.tracks[id] = tr
	}
	if tr.cleanup == "pending" {
		l.mu.Unlock()
		return errors.New("clean-up is already running")
	}
	tr.cleanup, tr.reason = "pending", ""
	var pr *PR
	if tr.pr != nil {
		c := *tr.pr
		pr = &c
	}
	l.mu.Unlock()
	l.sessions.changed()

	removed, reason := l.removeAll(ctx, t, pr, force)
	entry := Cleanup{At: time.Now().UnixMilli(), SessionTitle: t.title, Issue: t.issue, Status: "done", Removed: removed, Reason: reason}
	if pr != nil {
		entry.PR = pr.Number
	}
	if reason != "" {
		entry.Status = "blocked"
		l.mu.Lock()
		tr.cleanup, tr.reason = "blocked", reason
		l.mu.Unlock()
	}
	if err := l.appendCleanup(t.proj.Key(), entry); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: logging a clean-up: %v\n", err)
	}
	l.sessions.changed()
	l.emitCleanups()
	return nil
}

// removeAll runs the clean-up steps in order. A non-empty reason means it stopped before the end.
func (l *Lifecycle) removeAll(ctx context.Context, t target, pr *PR, force bool) (removed []string, reason string) {
	removed = []string{}
	dir, branch := t.proj.Dir, t.proj.BranchFor(t.issue)
	trees, err := l.worktrees(ctx, dir)
	if err != nil {
		return removed, fmt.Sprintf("git could not list the worktrees: %v", err)
	}
	worktree := trees.byBranch[branch]
	inMain := worktree != "" && samePath(worktree, trees.main)
	_, rerr := l.git(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	branchExists := rerr == nil

	if !force && !inMain {
		if reason := l.unsafe(ctx, dir, worktree, branch, branchExists, pr); reason != "" {
			return removed, reason
		}
	}

	if worktree != "" && !inMain {
		if err := l.removeWorktree(ctx, t.proj, branch, worktree, force); err != nil {
			return removed, fmt.Sprintf("removing the worktree failed: %v", err)
		}
		removed = append(removed, "worktree "+display(dir, worktree))
	}

	switch {
	case inMain:
		removed = append(removed, "branch "+branch+" kept: it is checked out in the main working tree")
	case branchExists:
		if _, err := l.git(ctx, dir, "branch", "-D", branch); err != nil {
			return removed, fmt.Sprintf("deleting the branch failed: %v", err)
		}
		removed = append(removed, "branch "+branch)
	}

	// Only the state file is ours: Claude Code's own files and anything in /tmp stay.
	if rec, _ := bus.ReadState(l.stateDir, t.id); rec.Session != "" {
		if err := bus.RemoveState(l.stateDir, t.id); err != nil {
			return removed, fmt.Sprintf("removing the state file failed: %v", err)
		}
		removed = append(removed, "temp files")
	}

	if l.release != nil {
		removed = append(removed, l.release(ctx, t.id)...)
	}
	l.closeTerm(t.id)
	if err := l.sessions.Kill(t.id); err != nil {
		return removed, fmt.Sprintf("ending the session failed: %v", err)
	}
	return append(removed, "session"), ""
}

// unsafe says why removing the worktree and branch could lose work, or "".
func (l *Lifecycle) unsafe(ctx context.Context, dir, worktree, branch string, branchExists bool, pr *PR) string {
	if worktree != "" {
		out, err := l.git(ctx, dir, "-C", worktree, "status", "--porcelain")
		switch {
		case err != nil:
			return fmt.Sprintf("git could not read the worktree: %v", err)
		case strings.TrimSpace(string(out)) != "":
			return "the worktree has uncommitted changes"
		}
	}
	if !branchExists {
		return ""
	}
	out, err := l.git(ctx, dir, "rev-list", "origin/"+branch+".."+branch)
	switch {
	case err != nil && pr != nil && pr.State == "merged":
		return "" // GitHub deletes the remote branch on merge
	case err != nil:
		return "the branch has no copy on origin to compare with"
	}
	switch n := len(strings.Fields(string(out))); {
	case n == 1:
		return "the branch has a commit that is not on origin"
	case n > 1:
		return fmt.Sprintf("the branch has %d commits that are not on origin", n)
	}
	return ""
}

func (l *Lifecycle) removeWorktree(ctx context.Context, proj project.Project, branch, worktree string, force bool) error {
	if proj.Cleanup != "" {
		cmd := strings.NewReplacer("{branch}", shellQuote(branch), "{worktree}", shellQuote(worktree)).Replace(proj.Cleanup)
		ctx, cancel := context.WithTimeout(ctx, 2*gitTimeout)
		defer cancel()
		_, err := l.run(ctx, proj.Dir, "sh", "-c", cmd)
		return err
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	_, err := l.git(ctx, proj.Dir, append(args, worktree)...)
	return err
}

func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// display shows path relative to dir when it lies inside it. Git prints real
// paths, so symlinks in dir are resolved first.
func display(dir, path string) string {
	if real, err := filepath.EvalSymlinks(dir); err == nil {
		dir = real
	}
	if rel, err := filepath.Rel(dir, path); err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func (l *Lifecycle) cleanupPath(key string) string {
	return filepath.Join(l.dataDir, "cleanups", key+".jsonl")
}

func (l *Lifecycle) appendCleanup(key string, c Cleanup) error {
	line, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("encoding clean-up: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(l.cleanupPath(key)), 0o700); err != nil {
		return fmt.Errorf("creating clean-up folder: %w", err)
	}
	f, err := os.OpenFile(l.cleanupPath(key), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("opening clean-up log: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("writing clean-up log: %w", err)
	}
	return nil
}

// Cleanups is the project's clean-up log, newest first.
func (l *Lifecycle) Cleanups(key string) []Cleanup {
	out := []Cleanup{}
	f, err := os.Open(l.cleanupPath(key))
	if errors.Is(err, fs.ErrNotExist) || err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(nil, 1<<20)
	for sc.Scan() {
		var c Cleanup
		if json.Unmarshal(sc.Bytes(), &c) == nil {
			if c.Removed == nil {
				c.Removed = []string{}
			}
			out = append(out, c)
		}
	}
	slices.Reverse(out)
	return out[:min(len(out), cleanupsKept)]
}

func (l *Lifecycle) emitCleanups() {
	l.emit("cleanups", l.Cleanups(l.sessions.Current().Key()))
}
