// Package issues reads a project's GitHub issues into queue sections and starts sessions for them.
package issues

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/internal/scoped"
	"github.com/nednella/agentos/desktop/internal/warn"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
	"github.com/nednella/agentos/internal/util"
)

// Sessions is what the issues need of the session list.
type Sessions interface {
	Current() project.Project
	IssueSessions() map[int]string
	List() []sessions.Session
	CreateIssue(title, text string, issue int, action project.Action, labels []string) (sessions.Session, error)
}

// Issue is a GitHub issue placed in a section of the queue.
type Issue struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Section   string   `json:"section"`
	Actions   []string `json:"actions"` // names of what can start a session for it, the default first
	Moves     []string `json:"moves"`   // names of the sections it can be dropped into, in queue order
	URL       string   `json:"url"`
	SessionID string   `json:"sessionId"`
	Author    string   `json:"author"`
	Assignees []string `json:"assignees"`
	Labels    []string `json:"labels"`
	CreatedAt int64    `json:"createdAt"`
	UpdatedAt int64    `json:"updatedAt"`
}

// Issues reads a project's open issues through the gh CLI and caches them per folder.
type Issues struct {
	run      run.Runner
	sessions Sessions
	emit     func(event string, payload any)

	warn *warn.Warnings

	mu         sync.Mutex
	repos      map[string]string
	repoFailed map[string]time.Time // when gh last failed to name a folder's repo for a reason that may pass
	lists      map[string][]Issue
	moved      map[moveKey]relabel // the last move of each issue, for UndoMove
}

type moveKey struct {
	dir    string
	number int
}

// relabel is the label change a move made.
type relabel struct {
	removed []string
	added   string
}

// ErrIssuesDisabled is what List fails with when the repo has issues turned off; the front end matches its text.
var ErrIssuesDisabled = errors.New("issues are disabled for this repo")

// repoRetry is how long a failed lookup of a folder's repo is not repeated.
const repoRetry = 30 * time.Second

// New reads issues through run.
func New(runner run.Runner, sessions Sessions, emit func(string, any)) *Issues {
	return &Issues{run: runner, sessions: sessions, emit: emit, warn: warn.New(emit), repos: map[string]string{}, repoFailed: map[string]time.Time{}, lists: map[string][]Issue{}, moved: map[moveKey]relabel{}}
}

// Repo is "owner/name" of the folder's GitHub repository, or "" when it has none or gh cannot say.
// A folder with no GitHub repo is remembered; a failure that may pass is warned about and tried again later.
func (i *Issues) Repo(ctx context.Context, dir string) string { return i.repo(ctx, dir, false) }

// repo answers from what gh said before unless fresh asks again. A known repo stays known
// while gh is asked and when it fails, so no screen sees the folder lose its repo for a moment.
func (i *Issues) repo(ctx context.Context, dir string, fresh bool) string {
	i.mu.Lock()
	known, ok := i.repos[dir]
	failedAt, failed := i.repoFailed[dir]
	i.mu.Unlock()
	if !fresh && (ok || (failed && time.Since(failedAt) < repoRetry)) {
		return known
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := i.run(ctx, dir, "gh", "repo", "view", "--json", "nameWithOwner")
	var repo string
	if err == nil {
		var v struct {
			NameWithOwner string `json:"nameWithOwner"`
		}
		if err = json.Unmarshal(out, &v); err != nil {
			err = fmt.Errorf("reading gh repo view: %w", err)
		}
		repo = v.NameWithOwner
	}
	if err != nil && !hasNoRepo(err) {
		i.mu.Lock()
		i.repoFailed[dir] = time.Now()
		i.mu.Unlock()
		if ctx.Err() == nil {
			i.warn.Report("github", util.FirstLine(err.Error()))
		}
		return known
	}
	i.mu.Lock()
	i.repos[dir] = repo
	delete(i.repoFailed, dir)
	i.mu.Unlock()
	i.warn.Clear("github")
	return repo
}

// hasNoRepo says whether gh failed because the folder has no GitHub repository, which no retry changes.
func hasNoRepo(err error) bool {
	msg := err.Error()
	for _, s := range []string{"no git remotes", "none of the git remotes", "not a git repository", "Could not resolve to a Repository"} {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}

// Cached returns the folder's issues if they were read before; it never runs gh.
func (i *Issues) Cached(dir string) ([]Issue, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	list, ok := i.lists[dir]
	return slices.Clone(list), ok
}

// List returns the project's open issues; without a repo that is none. Refreshing asks gh again for the repo and the issues.
func (i *Issues) List(ctx context.Context, proj project.Project, refresh bool) ([]Issue, error) {
	dir := proj.Dir
	if refresh {
		i.mu.Lock()
		delete(i.lists, dir)
		i.mu.Unlock()
	} else if cached, ok := i.Cached(dir); ok {
		return cached, nil
	}
	if i.repo(ctx, dir, refresh) == "" {
		return []Issue{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := i.run(ctx, dir, "gh", "issue", "list", "--state", "open", "--limit", "200", "--json", "number,title,labels,url,author,assignees,createdAt,updatedAt")
	if err != nil {
		if strings.Contains(err.Error(), "has disabled issues") {
			return nil, ErrIssuesDisabled
		}
		return nil, fmt.Errorf("listing issues: %w", err)
	}
	list, err := parseIssues(out, proj)
	if err != nil {
		return nil, err
	}
	i.mu.Lock()
	i.lists[dir] = list
	i.mu.Unlock()
	return slices.Clone(list), nil
}

type ghLogin struct {
	Login string `json:"login"`
}

func parseIssues(data []byte, proj project.Project) ([]Issue, error) {
	var raw []struct {
		Number    int       `json:"number"`
		Title     string    `json:"title"`
		URL       string    `json:"url"`
		Author    ghLogin   `json:"author"`
		Assignees []ghLogin `json:"assignees"`
		Labels    []struct {
			Name string `json:"name"`
		} `json:"labels"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("reading gh issue list: %w", err)
	}
	type placed struct {
		section int
		issue   Issue
	}
	list := make([]placed, 0, len(raw))
	for _, r := range raw {
		labels := make([]string, len(r.Labels))
		for k, l := range r.Labels {
			labels[k] = l.Name
		}
		assignees := make([]string, len(r.Assignees))
		for k, a := range r.Assignees {
			assignees[k] = a.Login
		}
		at, section := proj.Section(labels)
		list = append(list, placed{at, Issue{
			Number: r.Number, Title: r.Title, URL: r.URL, Section: section.Name, Actions: actionNames(section), Moves: proj.Moves(labels), Type: typeOf(labels),
			Author: r.Author.Login, Assignees: assignees, Labels: labels,
			CreatedAt: util.Millis(r.CreatedAt), UpdatedAt: util.Millis(r.UpdatedAt),
		}})
	}
	slices.SortStableFunc(list, func(a, b placed) int { return cmp.Compare(a.section, b.section) })
	out := make([]Issue, len(list))
	for i, p := range list {
		out[i] = p.issue
	}
	return out, nil
}

func actionNames(s project.Section) []string {
	var names []string
	for _, a := range s.ActionList() {
		names = append(names, a.Name)
	}
	return names
}

func typeOf(labels []string) string {
	for _, t := range []string{"bug", "feature", "refactor", "chore"} {
		if slices.Contains(labels, "type:"+t) {
			return t
		}
	}
	return ""
}

// CachedRepo is the repo of a folder if gh was asked before; it never runs gh.
func (i *Issues) CachedRepo(dir string) string {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.repos[dir]
}

// WithSessions fills in which live session each issue has.
func (i *Issues) WithSessions(list []Issue) []Issue {
	live := i.sessions.IssueSessions()
	for k := range list {
		list[k].SessionID = live[list[k].Number]
	}
	return list
}

// Emit sends the cached issues of the current project, when there are any, with their current sessions.
func (i *Issues) Emit() {
	cur := i.sessions.Current()
	if list, ok := i.Cached(cur.Dir); ok {
		i.emit("issues", scoped.Of(cur.Key(), i.WithSessions(list)))
	}
}

// Reload reads the project's issues afresh and sends them.
func (i *Issues) Reload(ctx context.Context, proj project.Project) {
	if _, err := i.List(ctx, proj, true); err == nil {
		i.Emit()
	}
}

// Start opens a session for the issue with the action of its section that has the name given, or the
// section's default action when action is "". An issue that already has a live session gets that session
// back instead of a second agent.
func (i *Issues) Start(ctx context.Context, number int, action string) (sessions.Session, error) {
	list, err := i.List(ctx, i.sessions.Current(), false)
	if err != nil {
		return sessions.Session{}, err
	}
	list = i.WithSessions(list)
	for _, is := range list {
		if is.Number != number {
			continue
		}
		if is.SessionID != "" {
			for _, s := range i.sessions.List() {
				if s.ID == is.SessionID {
					return s, nil
				}
			}
		}
		_, section := i.sessions.Current().Section(is.Labels)
		act, ok := pickAction(section, action)
		if !ok {
			return sessions.Session{}, fmt.Errorf("issue #%d has no action %q", number, action)
		}
		return i.sessions.CreateIssue(fmt.Sprintf("#%d %s", number, is.Title), act.Render(number, is.Title), number, act, is.Labels)
	}
	return sessions.Session{}, fmt.Errorf("issue #%d is not open in this project", number)
}

// Move relabels the issue on GitHub so that it falls in the section, then reads the queue again.
func (i *Issues) Move(ctx context.Context, number int, section string) error {
	proj := i.sessions.Current()
	list, err := i.List(ctx, proj, false)
	if err != nil {
		return err
	}
	at := slices.IndexFunc(list, func(is Issue) bool { return is.Number == number })
	if at < 0 {
		return fmt.Errorf("issue #%d is not open in this project", number)
	}
	remove, add, ok := proj.Relabel(list[at].Labels, section)
	if !ok {
		return fmt.Errorf("issue #%d can not move to %q", number, section)
	}
	if err := i.editLabels(ctx, proj.Dir, number, []string{add}, remove); err != nil {
		return fmt.Errorf("moving issue #%d: %w", number, err)
	}
	i.mu.Lock()
	i.moved[moveKey{proj.Dir, number}] = relabel{removed: remove, added: add}
	i.mu.Unlock()
	i.Reload(ctx, proj)
	return nil
}

// UndoMove reverses the last move of the issue, whatever the moves of its section now allow.
func (i *Issues) UndoMove(ctx context.Context, number int) error {
	proj := i.sessions.Current()
	key := moveKey{proj.Dir, number}
	i.mu.Lock()
	last, ok := i.moved[key]
	i.mu.Unlock()
	if !ok {
		return fmt.Errorf("issue #%d has no move to undo", number)
	}
	if err := i.editLabels(ctx, proj.Dir, number, last.removed, []string{last.added}); err != nil {
		return fmt.Errorf("undoing move of issue #%d: %w", number, err)
	}
	i.mu.Lock()
	delete(i.moved, key)
	i.mu.Unlock()
	i.Reload(ctx, proj)
	return nil
}

// editLabels adds and removes labels of the issue on GitHub; an empty label is skipped.
func (i *Issues) editLabels(ctx context.Context, dir string, number int, add, remove []string) error {
	args := []string{"issue", "edit", strconv.Itoa(number)}
	if add = slices.DeleteFunc(slices.Clone(add), func(l string) bool { return l == "" }); len(add) > 0 {
		args = append(args, "--add-label", strings.Join(add, ","))
	}
	if remove = slices.DeleteFunc(slices.Clone(remove), func(l string) bool { return l == "" }); len(remove) > 0 {
		args = append(args, "--remove-label", strings.Join(remove, ","))
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	_, err := i.run(ctx, dir, "gh", args...)
	return err
}

// pickAction is the action of the section with the name given, or its default for "".
func pickAction(s project.Section, name string) (project.Action, bool) {
	actions := s.ActionList()
	if name == "" {
		return actions[0], true
	}
	i := slices.IndexFunc(actions, func(a project.Action) bool { return a.Name == name })
	if i < 0 {
		return project.Action{}, false
	}
	return actions[i], true
}
