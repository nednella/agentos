package sessions

import (
	"bufio"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/session"
	"github.com/nednella/agentos/internal/util"
)

const (
	reconcileEvery = 5 * time.Minute // a full look at every pull request, in case a watcher missed something
	cleanupsKept   = 100
	gitTimeout     = 30 * time.Second

	closedReason = "app closed during clean-up"
)

// PR is a pull request as the front end sees it.
type PR struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	State     string `json:"state"`
	Checks    string `json:"checks"`
	Comments  int    `json:"comments"`
	UpdatedAt int64  `json:"updatedAt"`

	HeadOid string `json:"-"` // the commit the PR's branch ended on
}

// Cleanup is one entry of the clean-up log.
type Cleanup struct {
	At           int64    `json:"at"`
	SessionTitle string   `json:"sessionTitle"`
	Issue        int      `json:"issue"`
	PR           int      `json:"pr"`
	Merged       bool     `json:"merged"`
	Status       string   `json:"status"`
	Removed      []string `json:"removed"`
	Reason       string   `json:"reason"`
}

// target is a session that works on a branch, with the project it belongs to. It works on the
// branches it reported or was seen on, oldest first; a session of an issue with none works on the
// project's branch pattern, when it sets one.
type target struct {
	id, title    string
	issue        int
	branches     []string
	proj         project.Project
	model        project.Model // what the session ran, so a replacement for an ended one runs the same
	ended        bool
	conversation string // the agent's id for its conversation, "" when unknown
}

// track is what is known about the pull request and clean-up of one session.
type track struct {
	branch    string // the branch the PR was found on, or the newest one when there is none
	worktree  string
	pr        *PR
	attention string
	cleanup   string
	reason    string
	autoTried bool
	sawLive   bool // the PR was seen draft or open while this session ran
	opened    bool // the PR is new to the app; the user hears of it once the session is idle
}

// ack is how much of a PR the user has seen.
type ack struct {
	Comments int  `json:"comments"`
	Checks   bool `json:"checks"` // failing checks were seen
}

// nudge is how much of a PR its session has been woken for.
type nudge struct {
	Comments int    `json:"comments"`
	Head     string `json:"head"` // the commit whose failing checks the session was told about
}

// Repos is what the lifecycle needs of the issues: a project's repo, and a fresh queue when a PR merges or closes.
type Repos interface {
	Repo(ctx context.Context, dir string) string
	Reload(ctx context.Context, proj project.Project)
}

type noRepos struct{}

func (noRepos) Repo(context.Context, string) string     { return "" }
func (noRepos) Reload(context.Context, project.Project) {}

// Lifecycle follows each issue session's branch, worktree and pull request, wakes the
// session when its PR needs work, and cleans up after it when the PR is merged or closed.
type Lifecycle struct {
	run      run.Runner
	stream   run.Streamer
	dataDir  string
	stateDir string
	emit     func(event string, payload any)
	warn     *warn.Warnings
	sessions *Sessions
	repos    Repos

	pollMu sync.Mutex // one reconcile at a time
	logMu  sync.Mutex // the clean-up log files

	mu       sync.Mutex
	tracks   map[string]*track
	gone     map[string]bool     // forgotten ids: a poll that was waiting on gh must not write for them
	files    map[string]*prsFile // by project key
	etags    map[string]string   // by project key: the ETag of the last pull request list
	watchers map[string]*watcher // by project key

	cleanMu   sync.Mutex // guards stopping against cleanups starting
	stopping  bool
	cleanups  sync.WaitGroup // the clean-ups a poll started on its own
	stopped   context.Context
	stopClean context.CancelFunc
}

func newLifecycle(runner run.Runner, stream run.Streamer, dataDir, stateDir string, s *Sessions, emit func(string, any)) *Lifecycle {
	stopped, stop := context.WithCancel(context.Background())
	return &Lifecycle{
		run: runner, stream: stream, dataDir: dataDir, stateDir: stateDir, sessions: s, emit: emit, warn: warn.New(emit), repos: noRepos{},
		tracks: map[string]*track{}, gone: map[string]bool{}, files: map[string]*prsFile{}, etags: map[string]string{}, watchers: map[string]*watcher{},
		stopped: stopped, stopClean: stop,
	}
}

// autoCleanup cleans up after a session in the background, until ctx or stopCleanups ends it.
func (l *Lifecycle) autoCleanup(ctx context.Context, id string) {
	l.cleanMu.Lock()
	defer l.cleanMu.Unlock()
	if l.stopping {
		return
	}
	l.cleanups.Go(func() {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		defer context.AfterFunc(l.stopped, cancel)()
		_ = l.Cleanup(ctx, id, false)
	})
}

// stopCleanups cancels the clean-ups a poll started and waits until each has logged where it stopped.
func (l *Lifecycle) stopCleanups() {
	if l == nil {
		return
	}
	l.cleanMu.Lock()
	l.stopping = true
	l.cleanMu.Unlock()
	l.stopClean()
	l.cleanups.Wait()
}

// sessionFields is what a session row shows of its track.
type sessionFields struct {
	branch, worktree, attention, cleanup, reason string
	pr                                           *PR
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
	f := sessionFields{branch: t.branch, worktree: t.worktree, attention: t.attention, cleanup: t.cleanup, reason: t.reason}
	if t.pr != nil {
		pr := *t.pr
		f.pr = &pr
	}
	return f
}

// opened says, once, that the session's PR has appeared since the user last heard of it.
func (l *Lifecycle) opened(id string) bool {
	if l == nil {
		return false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	t := l.tracks[id]
	if t == nil || !t.opened {
		return false
	}
	t.opened = false
	return true
}

func (l *Lifecycle) forget(id string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.tracks, id)
	l.gone[id] = true
	key := session.ProjectKey(id)
	f := l.prs(key)
	_, worked := f.Branches[id]
	if f.Live[id] || worked {
		delete(f.Live, id)
		delete(f.Branches, id)
		l.savePRs(key)
	}
}

// Run reconciles every pull request now and then, and keeps a watcher per project in between.
func (l *Lifecycle) Run(ctx context.Context) {
	tick := time.NewTicker(reconcileEvery)
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

// group is the sessions of one project that work on a branch.
type group struct {
	proj    project.Project
	targets []target
}

// groups splits the sessions by project.
func groups(targets []target) []group {
	var out []group
	for _, t := range targets {
		i := slices.IndexFunc(out, func(g group) bool { return g.proj.Key() == t.proj.Key() })
		if i < 0 {
			out = append(out, group{proj: t.proj})
			i = len(out) - 1
		}
		out[i].targets = append(out[i].targets, t)
	}
	return out
}

// Poll looks up the worktree and pull request of every session on a branch, live or ended.
func (l *Lifecycle) Poll(ctx context.Context) {
	l.pollMu.Lock()
	defer l.pollMu.Unlock()
	var treeFailure, prFailure string
	for _, g := range groups(l.sessions.workTargets()) {
		treeErr, prErr := l.refresh(ctx, g.proj, g.targets, nil)
		if treeErr != nil && treeFailure == "" {
			treeFailure = util.FirstLine(treeErr.Error())
		}
		if prErr != nil && prFailure == "" {
			prFailure = util.FirstLine(prErr.Error())
		}
	}
	if ctx.Err() == nil {
		l.report("worktrees", treeFailure)
		l.report("pull requests", prFailure)
	}
	l.ensureWatchers(ctx)
}

// refresh fetches the pull requests of the project's sessions, all of them or only those on
// the branches named, and acts on what changed: clean-ups, a fresh queue, and wakes.
func (l *Lifecycle) refresh(ctx context.Context, proj project.Project, targets []target, only []string) (treeErr, prErr error) {
	trees, treeErr := l.worktrees(ctx, proj.Dir)
	prs := map[string]*PR{} // by branch: two sessions of one issue share its PR
	var auto []string
	var wakes []wake
	reload := false
	for _, t := range targets {
		if only != nil && !slices.ContainsFunc(t.branches, func(b string) bool { return slices.Contains(only, b) }) {
			continue
		}
		branch, pr, err := l.latestPR(ctx, proj.Dir, t.branches, prs)
		if err != nil {
			prErr = cmp.Or(prErr, err)
			continue
		}
		o := l.update(t, branch, trees.byBranch[branch], pr)
		if o.auto {
			auto = append(auto, t.id)
		}
		if o.wake != "" {
			wakes = append(wakes, wake{t, o.wake})
		}
		reload = reload || o.queue
	}
	for _, id := range auto {
		l.autoCleanup(ctx, id)
	}
	for _, w := range wakes {
		go l.sessions.wake(ctx, w.target, w.prompt)
	}
	l.sessions.changed()
	if reload && ctx.Err() == nil {
		l.repos.Reload(ctx, proj)
	}
	return treeErr, prErr
}

// latestPR is the PR on the newest of the branches that has one, with its branch; without one it is
// nil, with the newest branch. found caches the lookups by branch.
func (l *Lifecycle) latestPR(ctx context.Context, dir string, branches []string, found map[string]*PR) (string, *PR, error) {
	for _, branch := range slices.Backward(branches) {
		pr, ok := found[branch]
		if !ok {
			var err error
			if pr, err = l.findPR(ctx, dir, branch); err != nil {
				return "", nil, err
			}
			found[branch] = pr
		}
		if pr != nil {
			return branch, pr, nil
		}
	}
	return branches[len(branches)-1], nil, nil
}

// wake is a prompt to send a session.
type wake struct {
	target target
	prompt string
}

// report warns about a failure of the source, or says it works again when failure is "".
func (l *Lifecycle) report(source, failure string) {
	if failure == "" {
		l.warn.Clear(source)
		return
	}
	l.warn.Report(source, failure)
}

// outcome is what a fetched pull request calls for.
type outcome struct {
	auto  bool   // clean up now
	queue bool   // the PR merged or closed, so the issue leaves the queue
	wake  string // a prompt to send the session, or ""
}

// update stores what a fetch found and says what it calls for.
func (l *Lifecycle) update(t target, branch, worktree string, pr *PR) outcome {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.gone[t.id] {
		return outcome{}
	}
	key := t.proj.Key()
	tr := l.tracks[t.id]
	if tr == nil {
		tr = &track{sawLive: l.prs(key).Live[t.id]}
		l.tracks[t.id] = tr
	}
	prev := tr.pr
	tr.branch, tr.worktree, tr.pr = branch, worktree, pr
	var o outcome
	if pr == nil {
		if tr.cleanup == "ask" {
			tr.cleanup = ""
		}
		return o
	}
	live := pr.State == "draft" || pr.State == "open"
	if _, known := l.prs(key).Acks[strconv.Itoa(pr.Number)]; live && !known {
		tr.opened = true
	}
	if live && !tr.sawLive {
		tr.sawLive = true
		l.prs(key).Live[t.id] = true
		l.savePRs(key)
	}
	attention := l.attentionFor(key, pr)
	if attention != "" && attention != tr.attention {
		l.emit("attention", map[string]string{"id": t.id, "state": "pr"})
	}
	tr.attention = attention
	if live {
		o.wake = l.nudgeFor(t.proj, pr)
	}
	o.queue = !live && (prev == nil || prev.State != pr.State)
	switch {
	case live:
		if tr.cleanup == "ask" {
			tr.cleanup = ""
		}
	case tr.cleanup != "":
	case !tr.sawLive || !wantsAuto(t.proj.CleanupMode, pr.State):
		// Without sight of the PR open, the branch may be old: the user decides.
		tr.cleanup = "ask"
	case !tr.autoTried:
		tr.autoTried = true
		o.auto = true
	}
	return o
}

func wantsAuto(c project.Cleanup, state string) bool {
	if state == "merged" {
		return c.OnMerge() == "auto"
	}
	return c.OnClose() == "auto"
}

func (l *Lifecycle) prsPath(key string) string { return filepath.Join(l.dataDir, key, "prs.json") }

// prsFile is what is remembered of a project's PRs: what the user has seen of each,
// which sessions saw their PR open, and what each PR's session was woken for.
type prsFile struct {
	Acks   map[string]ack   `json:"acks"`   // PR number -> ack
	Live   map[string]bool  `json:"live"`   // session id -> its PR was seen draft or open
	Nudged map[string]nudge `json:"nudged"` // PR number -> nudge

	Branches map[string][]string `json:"branches"` // session id -> the branches it worked on, oldest first
}

// prs needs mu.
func (l *Lifecycle) prs(key string) *prsFile {
	if f, ok := l.files[key]; ok {
		return f
	}
	f := &prsFile{}
	if data, err := os.ReadFile(l.prsPath(key)); err == nil {
		_ = json.Unmarshal(data, f)
	}
	if f.Acks == nil {
		f.Acks = map[string]ack{}
	}
	if f.Live == nil {
		f.Live = map[string]bool{}
	}
	if f.Nudged == nil {
		f.Nudged = map[string]nudge{}
	}
	if f.Branches == nil {
		f.Branches = map[string][]string{}
	}
	l.files[key] = f
	return f
}

// savePRs needs mu.
func (l *Lifecycle) savePRs(key string) {
	data, err := json.Marshal(l.prs(key))
	if err == nil {
		err = atomicfile.Write(l.prsPath(key), data, 0o600)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "agentos: saving PR state: %v\n", err)
	}
}

// attentionFor says what about the PR needs the user: failing checks, or more
// comments than they last saw. It needs mu.
func (l *Lifecycle) attentionFor(key string, pr *PR) string {
	acks := l.prs(key).Acks
	n := strconv.Itoa(pr.Number)
	a, seen := acks[n]
	if !seen {
		a = ack{Comments: pr.Comments}
		acks[n] = a
		l.savePRs(key)
	}
	if pr.Checks != "failing" && a.Checks {
		a.Checks = false
		acks[n] = a
		l.savePRs(key)
	}
	switch {
	case pr.Checks == "failing" && !a.Checks:
		return "checks"
	case pr.Comments > a.Comments:
		return "comments"
	}
	return ""
}

// nudgeFor is the prompt the PR's session should get: once per new comment or review, and once
// per commit whose checks fail. Comments a PR had when first seen are not news. A project
// that sets no on_review or on_checks gets no prompt, but the event still counts as seen.
// It needs mu.
func (l *Lifecycle) nudgeFor(proj project.Project, pr *PR) string {
	key := proj.Key()
	nudged := l.prs(key).Nudged
	n := strconv.Itoa(pr.Number)
	seen, known := nudged[n]
	if !known {
		seen = nudge{Comments: pr.Comments}
	}
	head := cmp.Or(pr.HeadOid, "unknown")
	prompt := ""
	switch {
	case pr.Comments > seen.Comments:
		seen.Comments = pr.Comments
		prompt = proj.ReviewCommand(pr.Number)
	case pr.Checks == "failing" && seen.Head != head:
		seen.Head = head
		prompt = proj.ChecksCommand(pr.Number)
	}
	if !known || prompt != "" {
		nudged[n] = seen
		l.savePRs(key)
	}
	return prompt
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
	l.prs(key).Acks[strconv.Itoa(tr.pr.Number)] = ack{Comments: tr.pr.Comments, Checks: tr.pr.Checks == "failing"}
	l.savePRs(key)
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
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := l.run(ctx, dir, "gh", "pr", "list", "--head", branch, "--state", "all", "--json",
		"number,url,state,isDraft,headRefOid,statusCheckRollup,comments,reviews,updatedAt", "--limit", "1")
	if err != nil {
		return nil, fmt.Errorf("finding a pull request: %w", err)
	}
	return parsePR(out)
}

func parsePR(data []byte) (*PR, error) {
	var raw []struct {
		Number  int    `json:"number"`
		URL     string `json:"url"`
		State   string `json:"state"`
		IsDraft bool   `json:"isDraft"`
		HeadOid string `json:"headRefOid"`
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
	pr := &PR{Number: r.Number, URL: r.URL, Comments: len(r.Comments) + len(r.Reviews), UpdatedAt: util.Millis(r.UpdatedAt), Checks: "none", HeadOid: r.HeadOid}
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
	prBranch := tr.branch
	l.mu.Unlock()
	l.sessions.changed()

	removed, reason := l.removeAll(ctx, t, pr, prBranch, force)
	if reason != "" && ctx.Err() != nil {
		reason = closedReason
	}
	entry := Cleanup{At: time.Now().UnixMilli(), SessionTitle: t.title, Issue: t.issue, Status: "done", Removed: removed, Reason: reason}
	if pr != nil {
		entry.PR, entry.Merged = pr.Number, pr.State == "merged"
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
	key := t.proj.Key()
	l.emit("cleanups", scoped.Of(key, l.Cleanups(key)))
	return nil
}

// removeAll runs the clean-up steps in order. A non-empty reason means it stopped before the end.
// A step that finds the context cancelled stops with closedReason.
func (l *Lifecycle) removeAll(ctx context.Context, t target, pr *PR, prBranch string, force bool) (removed []string, reason string) {
	removed = []string{}
	steps := []func(context.Context, target, *cleanupPlan) ([]string, string){l.runCleanupCommand, l.removeSession}
	plan, reason := l.plan(ctx, t, pr, prBranch, force)
	for _, step := range steps {
		if reason != "" {
			break
		}
		if ctx.Err() != nil {
			return removed, closedReason
		}
		var done []string
		done, reason = step(ctx, t, &plan)
		removed = append(removed, done...)
	}
	return removed, reason
}

// cleanupPlan is what a clean-up found out before it runs the project's command.
type cleanupPlan struct {
	branches []branchPlan
	force    bool
}

type branchPlan struct {
	branch       string
	worktree     string // the folder the branch is checked out in, or ""
	branchExists bool
}

// plan looks at the worktree and branch of each branch the session worked on, and unless forced,
// says why cleaning them up could lose work. The PR counts only for the branch it was found on.
func (l *Lifecycle) plan(ctx context.Context, t target, pr *PR, prBranch string, force bool) (cleanupPlan, string) {
	dir := t.proj.Dir
	trees, err := l.worktrees(ctx, dir)
	if err != nil {
		return cleanupPlan{}, fmt.Sprintf("git could not list the worktrees: %v", err)
	}
	p := cleanupPlan{force: force}
	for _, branch := range t.branches {
		b := branchPlan{branch: branch, worktree: trees.byBranch[branch]}
		_, rerr := l.git(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
		b.branchExists = rerr == nil
		p.branches = append(p.branches, b)
		if force {
			continue
		}
		branchPR := pr
		if branch != prBranch {
			branchPR = nil
		}
		if reason := l.unsafe(ctx, dir, b.worktree, branch, b.branchExists, branchPR); reason != "" {
			return p, reason
		}
	}
	return p, ""
}

// runCleanupCommand hands each branch the session worked on to the project's clean-up command.
// With no command the git side stays as it is.
func (l *Lifecycle) runCleanupCommand(ctx context.Context, t target, p *cleanupPlan) (removed []string, reason string) {
	if t.proj.CleanupCommand == "" {
		return nil, ""
	}
	for _, b := range p.branches {
		if b.worktree == "" && !b.branchExists {
			continue
		}
		if err := l.runProjectCommand(ctx, t.proj, b, p.force); err != nil {
			if !l.leftBehind(ctx, t.proj.Dir, b.branch) {
				removed = append(removed, fmt.Sprintf("clean-up command for %s (failed with nothing left: %v)", b.branch, err))
				continue
			}
			return removed, fmt.Sprintf("the clean-up command failed for %s: %v", b.branch, err)
		}
		removed = append(removed, "clean-up command for "+b.branch)
	}
	return removed, ""
}

// leftBehind says whether the branch still exists or any worktree has it checked out. A failed
// git call counts as left behind, so the clean-up blocks as it did.
func (l *Lifecycle) leftBehind(ctx context.Context, dir, branch string) bool {
	trees, err := l.worktrees(ctx, dir)
	if err != nil {
		return true
	}
	if _, ok := trees.byBranch[branch]; ok {
		return true
	}
	_, err = l.git(ctx, dir, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	var exit *exec.ExitError
	return !errors.As(err, &exit) || exit.ExitCode() != 1
}

// removeSession removes what agentos itself keeps for the session, then the session.
func (l *Lifecycle) removeSession(ctx context.Context, t target, _ *cleanupPlan) (removed []string, reason string) {
	// Only the state file is ours: Claude Code's own files and anything in /tmp stay.
	if rec, _ := bus.ReadState(l.stateDir, t.id); rec.Session != "" {
		if err := bus.RemoveState(l.stateDir, t.id); err != nil {
			return removed, fmt.Sprintf("removing the state file failed: %v", err)
		}
		removed = append(removed, "temp files")
	}
	if l.sessions.browsers.Has(t.id) {
		l.sessions.browsers.Close(ctx, t.id)
		removed = append(removed, "browser tab")
	}
	if l.sessions.evidence.Purge(t.id) > 0 {
		removed = append(removed, "evidence")
	}
	if !t.ended {
		if err := l.sessions.Kill(t.id); err != nil {
			return removed, fmt.Sprintf("ending the session failed: %v", err)
		}
	}
	if err := l.sessions.Dismiss(t.id); err != nil {
		return removed, fmt.Sprintf("removing the session failed: %v", err)
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
	merged := pr != nil && pr.State == "merged"
	out, err := l.git(ctx, dir, "rev-list", "origin/"+branch+".."+branch)
	switch {
	case err != nil && merged:
		// GitHub deletes the remote branch on merge, so compare with what the PR merged instead.
		return l.afterMergedHead(ctx, dir, branch, pr.HeadOid)
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

// afterMergedHead says why the branch holds commits the merged pull request does not, or "".
func (l *Lifecycle) afterMergedHead(ctx context.Context, dir, branch, head string) string {
	if head == "" {
		return "GitHub did not say which commit the merged pull request ended on"
	}
	_, err := l.git(ctx, dir, "merge-base", "--is-ancestor", branch, head)
	var exit *exec.ExitError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &exit) && exit.ExitCode() == 1:
		return "the branch has commits that are not in the merged pull request"
	}
	return fmt.Sprintf("git could not compare the branch with the merged pull request: %v", err)
}

func (l *Lifecycle) runProjectCommand(ctx context.Context, proj project.Project, b branchPlan, force bool) error {
	forceFlag := ""
	if force {
		forceFlag = "--force"
	}
	cmd := strings.NewReplacer(
		"{branch}", util.ShellQuote(b.branch),
		"{worktree}", util.ShellQuote(b.worktree),
		"{dir}", util.ShellQuote(proj.Dir),
		"{force}", forceFlag,
	).Replace(proj.CleanupCommand)
	ctx, cancel := context.WithTimeout(ctx, 2*gitTimeout)
	defer cancel()
	_, err := l.run(ctx, proj.Dir, "sh", "-c", cmd)
	return err
}

func (l *Lifecycle) cleanupPath(key string) string {
	return filepath.Join(l.dataDir, key, "cleanups.json")
}

// readCleanups is the project's log, oldest first.
func (l *Lifecycle) readCleanups(key string) []Cleanup {
	var log []Cleanup
	if data, err := os.ReadFile(l.cleanupPath(key)); err == nil {
		_ = json.Unmarshal(data, &log)
	}
	return log
}

func (l *Lifecycle) appendCleanup(key string, c Cleanup) error {
	l.logMu.Lock()
	defer l.logMu.Unlock()
	data, err := json.Marshal(append(l.readCleanups(key), c))
	if err != nil {
		return fmt.Errorf("encoding clean-up log: %w", err)
	}
	return atomicfile.Write(l.cleanupPath(key), data, 0o600)
}

// Cleanups is the project's clean-up log, newest first.
func (l *Lifecycle) Cleanups(key string) []Cleanup {
	l.logMu.Lock()
	out := l.readCleanups(key)
	l.logMu.Unlock()
	if out == nil {
		out = []Cleanup{}
	}
	for i := range out {
		if out[i].Removed == nil {
			out[i].Removed = []string{}
		}
	}
	slices.Reverse(out)
	return out[:min(len(out), cleanupsKept)]
}

// branchesOf are the branches a session worked on, oldest first.
func (l *Lifecycle) branchesOf(proj project.Project, id string) []string {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.prs(proj.Key()).Branches[id])
}

// learn tracks the branch checked out in cwd, when cwd is in the project's repo and the branch is
// not the main one.
func (l *Lifecycle) learn(ctx context.Context, id string, proj project.Project, cwd string, turnEnded bool) {
	if branch, ok := l.workBranch(ctx, proj, cwd); ok {
		l.track(ctx, id, proj, branch, turnEnded)
	}
}

// track notes branch as the newest one the session works on and looks up its pull request.
// A branch it already knows is looked up again only when the turn ended: the agent may just have opened the PR.
func (l *Lifecycle) track(ctx context.Context, id string, proj project.Project, branch string, turnEnded bool) {
	key := proj.Key()
	l.mu.Lock()
	if l.gone[id] {
		l.mu.Unlock()
		return
	}
	f := l.prs(key)
	known := f.Branches[id]
	if len(known) > 0 && known[len(known)-1] == branch {
		l.mu.Unlock()
		if turnEnded {
			l.refreshOnly(ctx, proj, l.targetsOf(proj), []string{branch})
		}
		return
	}
	f.Branches[id] = append(slices.DeleteFunc(slices.Clone(known), func(b string) bool { return b == branch }), branch)
	l.savePRs(key)
	l.mu.Unlock()
	l.ensureWatchers(ctx)
	l.refreshOnly(ctx, proj, l.targetsOf(proj), []string{branch})
}

// workBranch is the branch checked out in cwd, unless cwd is detached, on the main branch or in another repo.
func (l *Lifecycle) workBranch(ctx context.Context, proj project.Project, cwd string) (string, bool) {
	out, err := l.git(ctx, cwd, "branch", "--show-current")
	branch := strings.TrimSpace(string(out))
	if err != nil || branch == "" || branch == l.mainBranch(ctx, proj.Dir) {
		return "", false
	}
	common := func(dir string) string {
		out, err := l.git(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
		if err != nil {
			return ""
		}
		return strings.TrimSpace(string(out))
	}
	here := common(cwd)
	return branch, here != "" && samePath(here, common(proj.Dir))
}

// mainBranch is the branch origin calls its default, or main when origin does not say.
func (l *Lifecycle) mainBranch(ctx context.Context, dir string) string {
	out, err := l.git(ctx, dir, "symbolic-ref", "--short", "refs/remotes/origin/HEAD")
	if err != nil {
		return "main"
	}
	return cmp.Or(strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/"), "main")
}
