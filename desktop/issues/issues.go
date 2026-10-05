// Package issues reads a project's GitHub issues into lanes and starts sessions for them.
package issues

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/run"
	"github.com/nednella/agentos/desktop/sessions"
	"github.com/nednella/agentos/internal/project"
)

// Sessions is what the issues need of the session list.
type Sessions interface {
	Current() project.Project
	IssueSessions() map[int]string
	List() []sessions.Session
	Create(title, prefill string, issue int) (sessions.Session, error)
}

// Issue is a GitHub issue placed in a lane of the board.
type Issue struct {
	Number    int      `json:"number"`
	Title     string   `json:"title"`
	Type      string   `json:"type"`
	Lane      string   `json:"lane"`
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

	mu    sync.Mutex
	repos map[string]string
	lists map[string][]Issue
}

// New reads issues through run.
func New(runner run.Runner, sessions Sessions, emit func(string, any)) *Issues {
	return &Issues{run: runner, sessions: sessions, emit: emit, repos: map[string]string{}, lists: map[string][]Issue{}}
}

// Repo is "owner/name" of the folder's GitHub repository, or "" when gh cannot say.
func (i *Issues) Repo(ctx context.Context, dir string) string {
	i.mu.Lock()
	repo, ok := i.repos[dir]
	i.mu.Unlock()
	if ok {
		return repo
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	if out, err := i.run(ctx, dir, "gh", "repo", "view", "--json", "nameWithOwner"); err == nil {
		var v struct {
			NameWithOwner string `json:"nameWithOwner"`
		}
		if json.Unmarshal(out, &v) == nil {
			repo = v.NameWithOwner
		}
	}
	i.mu.Lock()
	i.repos[dir] = repo
	i.mu.Unlock()
	return repo
}

// Cached returns the folder's issues if they were read before; it never runs gh.
func (i *Issues) Cached(dir string) ([]Issue, bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	list, ok := i.lists[dir]
	return slices.Clone(list), ok
}

// List returns the project's open issues; without a repo that is none. Refreshing drops the cache first.
func (i *Issues) List(ctx context.Context, proj project.Project, refresh bool) ([]Issue, error) {
	dir := proj.Dir
	if refresh {
		i.mu.Lock()
		delete(i.repos, dir)
		delete(i.lists, dir)
		i.mu.Unlock()
	}
	if cached, ok := i.Cached(dir); ok {
		return cached, nil
	}
	if i.Repo(ctx, dir) == "" {
		return []Issue{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, run.GHTimeout)
	defer cancel()
	out, err := i.run(ctx, dir, "gh", "issue", "list", "--state", "open", "--limit", "200", "--json", "number,title,labels,url,author,assignees,createdAt,updatedAt")
	if err != nil {
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
	out := make([]Issue, 0, len(raw))
	for _, r := range raw {
		labels := make([]string, len(r.Labels))
		for k, l := range r.Labels {
			labels[k] = l.Name
		}
		assignees := make([]string, len(r.Assignees))
		for k, a := range r.Assignees {
			assignees[k] = a.Login
		}
		out = append(out, Issue{
			Number: r.Number, Title: r.Title, URL: r.URL, Lane: proj.Lane(labels), Type: typeOf(labels),
			Author: r.Author.Login, Assignees: assignees, Labels: labels,
			CreatedAt: millis(r.CreatedAt), UpdatedAt: millis(r.UpdatedAt),
		})
	}
	return out, nil
}

func typeOf(labels []string) string {
	for _, t := range []string{"bug", "feature", "refactor", "chore"} {
		if slices.Contains(labels, "type:"+t) {
			return t
		}
	}
	return ""
}

func (i Issue) sessionTitle() string {
	title := []rune(i.Title)
	if len(title) > 40 {
		title = append(title[:39], '…')
	}
	return fmt.Sprintf("#%d %s", i.Number, string(title))
}

func millis(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UnixMilli()
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
	if list, ok := i.Cached(i.sessions.Current().Dir); ok {
		i.emit("issues", i.WithSessions(list))
	}
}

// Reload reads the project's issues afresh and sends them.
func (i *Issues) Reload(ctx context.Context, proj project.Project) {
	if _, err := i.List(ctx, proj, true); err == nil {
		i.Emit()
	}
}

// Start opens a session for the issue. An issue that already has a live
// session gets that session back instead of a second agent.
func (i *Issues) Start(ctx context.Context, number int) (sessions.Session, error) {
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
		return i.sessions.Create(is.sessionTitle(), i.sessions.Current().IssueCommand(is.Lane, number), number)
	}
	return sessions.Session{}, fmt.Errorf("issue #%d is not open in this project", number)
}
