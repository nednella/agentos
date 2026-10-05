package sessions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/util"
)

const (
	webhookEvents = "pull_request,pull_request_review,issue_comment,check_suite"
	webhookSettle = 10 * time.Second // a stream that lasts this long worked; one that ends sooner never did
	webhookRetry  = 30 * time.Second
	repoRetry     = 30 * time.Second
	payloadCap    = 4 << 20
)

// watcher follows one project's pull requests between reconciles: a webhook stream where
// gh webhook forward works, else a conditional poll of the REST list.
type watcher struct {
	cancel context.CancelFunc
	config string // the pr_watch and pr_poll it was started with
}

// ensureWatchers keeps a watcher running for every project with issue sessions, and stops the rest.
func (l *Lifecycle) ensureWatchers(ctx context.Context) {
	want := map[string]project.Project{}
	for _, g := range groups(l.sessions.issueTargets()) {
		want[g.proj.Key()] = g.proj
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, w := range l.watchers {
		if proj, ok := want[key]; !ok || w.config != watchConfig(proj) {
			w.cancel()
			delete(l.watchers, key)
		}
	}
	for key, proj := range want {
		if _, ok := l.watchers[key]; ok || ctx.Err() != nil {
			continue
		}
		wctx, cancel := context.WithCancel(ctx)
		l.watchers[key] = &watcher{cancel: cancel, config: watchConfig(proj)}
		go l.watch(wctx, proj)
	}
}

func watchConfig(proj project.Project) string { return proj.PRWatch + " " + proj.PRPoll }

// watch runs until ctx ends. Unless the project says poll, the webhook stream is tried first;
// a stream that never got going leaves the poll.
func (l *Lifecycle) watch(ctx context.Context, proj project.Project) {
	for ctx.Err() == nil {
		repo := l.repos.Repo(ctx, proj.Dir)
		if repo == "" {
			if !sleep(ctx, repoRetry) {
				return
			}
			continue
		}
		if proj.PRWatch != "poll" && l.forward(ctx, proj, repo) {
			continue
		}
		l.pollPRs(ctx, proj, repo)
	}
}

// sleep waits for d, and says whether ctx is still live.
func sleep(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// forward streams the repo's events through gh webhook forward and refreshes the pull requests they name.
// It returns true when the stream worked and dropped, so it is worth another try after a pause.
func (l *Lifecycle) forward(ctx context.Context, proj project.Project, repo string) bool {
	started := time.Now()
	out, err := l.stream(ctx, proj.Dir, "gh", "webhook", "forward", "--repo", repo, "--events", webhookEvents)
	if err == nil {
		sc := bufio.NewScanner(out)
		sc.Buffer(make([]byte, 64<<10), payloadCap)
		for sc.Scan() {
			l.onEvent(ctx, proj, sc.Bytes())
		}
		err = out.Close()
	}
	if ctx.Err() != nil {
		return false
	}
	ran := time.Since(started) >= webhookSettle
	switch {
	case proj.PRWatch == "webhook":
		l.report("pull requests", fmt.Sprintf("gh webhook forward for %s stopped: %s", repo, util.FirstLine(endedWith(err))))
	case !ran:
		fmt.Fprintf(os.Stderr, "agentos: gh webhook forward does not work for %s, polling its pull requests instead: %v\n", repo, err)
		return false
	}
	return sleep(ctx, webhookRetry)
}

func endedWith(err error) string {
	if err == nil {
		return "it ended"
	}
	return err.Error()
}

// onEvent refreshes the pull requests a webhook payload names. A number no session tracks yet,
// as when a PR is opened, refreshes them all.
func (l *Lifecycle) onEvent(ctx context.Context, proj project.Project, payload []byte) {
	var ev struct {
		PullRequest *struct{ Number int } `json:"pull_request"`
		Issue       *struct {
			Number      int             `json:"number"`
			PullRequest json.RawMessage `json:"pull_request"`
		} `json:"issue"`
		CheckSuite *struct {
			PullRequests []struct{ Number int } `json:"pull_requests"`
		} `json:"check_suite"`
	}
	if err := json.Unmarshal(payload, &ev); err != nil {
		return
	}
	var numbers []int
	switch {
	case ev.PullRequest != nil:
		numbers = append(numbers, ev.PullRequest.Number)
	case ev.Issue != nil && ev.Issue.PullRequest != nil:
		numbers = append(numbers, ev.Issue.Number)
	case ev.CheckSuite != nil:
		for _, pr := range ev.CheckSuite.PullRequests {
			numbers = append(numbers, pr.Number)
		}
	default:
		return
	}
	targets := l.targetsOf(proj)
	var only []string
	for _, n := range numbers {
		branch, ok := l.branchOfPR(proj, targets, n)
		if !ok {
			only = nil
			break
		}
		only = append(only, branch)
	}
	l.refreshOnly(ctx, proj, targets, only)
}

// branchOfPR is the branch of the session that tracks the pull request.
func (l *Lifecycle) branchOfPR(proj project.Project, targets []target, number int) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, t := range targets {
		if tr := l.tracks[t.id]; tr != nil && tr.pr != nil && tr.pr.Number == number {
			return proj.BranchFor(t.issue), true
		}
	}
	return "", false
}

// targetsOf are the project's issue sessions.
func (l *Lifecycle) targetsOf(proj project.Project) []target {
	var out []target
	for _, t := range l.sessions.issueTargets() {
		if t.proj.Key() == proj.Key() {
			out = append(out, t)
		}
	}
	return out
}

// refreshOnly refreshes the branches named, or all of them when only is nil, and warns when gh fails.
func (l *Lifecycle) refreshOnly(ctx context.Context, proj project.Project, targets []target, only []string) {
	if only != nil && len(only) == 0 {
		return
	}
	_, err := l.refresh(ctx, proj, targets, only)
	if err != nil && ctx.Err() == nil {
		l.report("pull requests", util.FirstLine(err.Error()))
	}
}

// pollPRs asks GitHub for the repo's pull request list at the project's interval, with the ETag of
// the last answer so an unchanged list costs nothing, and refreshes the branches whose PR moved.
func (l *Lifecycle) pollPRs(ctx context.Context, proj project.Project, repo string) {
	every, err := proj.PRPollEvery()
	if err != nil {
		every = project.DefaultPRPoll
	}
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		targets := l.targetsOf(proj)
		changed, err := l.changedBranches(ctx, proj, repo, targets)
		if err != nil {
			if ctx.Err() == nil {
				l.report("pull requests", util.FirstLine(err.Error()))
			}
			continue
		}
		// Checks do not touch a PR's updated_at, so pending ones are looked at every time.
		l.refreshOnly(ctx, proj, targets, append(changed, l.pendingBranches(proj, targets)...))
	}
}

// changedBranches are the tracked branches whose pull request was updated since the last fetch,
// as the repo's PR list says. A list GitHub reports unchanged (304) names none.
func (l *Lifecycle) changedBranches(ctx context.Context, proj project.Project, repo string, targets []target) ([]string, error) {
	key := proj.Key()
	l.mu.Lock()
	etag := l.etags[key]
	l.mu.Unlock()
	args := []string{"api", "-i", fmt.Sprintf("repos/%s/pulls?state=all&sort=updated&direction=desc&per_page=100", repo)}
	if etag != "" {
		args = append(args, "-H", "If-None-Match: "+etag)
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := l.run(ctx, proj.Dir, "gh", args...)
	if err != nil {
		if strings.Contains(err.Error(), "HTTP 304") {
			return []string{}, nil
		}
		return nil, fmt.Errorf("listing the pull requests of %s: %w", repo, err)
	}
	etag, body := splitResponse(out)
	var list []struct {
		Head      struct{ Ref string } `json:"head"`
		UpdatedAt time.Time            `json:"updated_at"`
	}
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("reading the pull request list of %s: %w", repo, err)
	}
	updated := map[string]int64{}
	for _, pr := range slices.Backward(list) { // the newest PR of a branch wins
		updated[pr.Head.Ref] = util.Millis(pr.UpdatedAt)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.etags[key] = etag
	changed := []string{}
	for _, t := range targets {
		branch := proj.BranchFor(t.issue)
		at, listed := updated[branch]
		tr := l.tracks[t.id]
		known := tr != nil && tr.pr != nil
		if listed && (!known || tr.pr.UpdatedAt != at) && !slices.Contains(changed, branch) {
			changed = append(changed, branch)
		}
	}
	return changed, nil
}

// splitResponse takes the ETag and the body out of what gh api -i printed: the headers, a blank line, the body.
func splitResponse(out []byte) (etag string, body []byte) {
	text := string(out)
	i := strings.Index(text, "\n\n")
	if j := strings.Index(text, "\r\n\r\n"); j >= 0 && (i < 0 || j < i) {
		i = j
	}
	if i < 0 {
		return "", out
	}
	head, rest := text[:i], strings.TrimLeft(text[i:], "\r\n")
	for line := range strings.SplitSeq(head, "\n") {
		if name, value, ok := strings.Cut(line, ":"); ok && strings.EqualFold(name, "etag") {
			etag = strings.TrimSpace(value)
		}
	}
	return etag, []byte(rest)
}

// pendingBranches are the tracked branches whose PR has checks still running.
func (l *Lifecycle) pendingBranches(proj project.Project, targets []target) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []string
	for _, t := range targets {
		if tr := l.tracks[t.id]; tr != nil && tr.pr != nil && tr.pr.Checks == "pending" {
			out = append(out, proj.BranchFor(t.issue))
		}
	}
	return out
}
