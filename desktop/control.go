package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/session"
)

const maxTextCard = 64 << 10

// handleControl answers a command from the agentos command line.
func (a *App) handleControl(ctx context.Context, req control.Request) control.Response {
	out, err := a.runControl(ctx, req)
	if err != nil {
		return control.Response{Error: err.Error()}
	}
	return control.Response{OK: true, Out: out}
}

// askerSession checks that the command comes from a session that runs.
func (a *App) askerSession(req control.Request) (string, error) {
	if _, err := session.ParseName(req.Session); err != nil {
		return "", errors.New("this command works inside an agentos session; AGENTOS_SESSION is not set")
	}
	if !a.sessions.Has(req.Session) {
		return "", fmt.Errorf("session %s is not running", req.Session)
	}
	return req.Session, nil
}

// askerProject is the project key of the asking session, else the current project.
func (a *App) askerProject(req control.Request) string {
	if req.Project != "" {
		return req.Project
	}
	if name, err := session.ParseName(req.Session); err == nil {
		return name.Project
	}
	return a.sessions.Current().Key()
}

func (a *App) runControl(ctx context.Context, req control.Request) (string, error) {
	switch req.Cmd {
	case "browser":
		return a.controlBrowser(ctx, req)
	case "show":
		id, err := a.askerSession(req)
		if err != nil {
			return "", err
		}
		return a.controlShow(id, req)
	case "note":
		key := a.askerProject(req)
		n, err := a.notes.Add(key, strings.Join(req.Args, " "))
		if err != nil {
			return "", err
		}
		if key == a.sessions.Current().Key() {
			a.emitNotes()
		}
		return "note saved: " + n.title(), nil
	case "stats":
		if req.Opts["open"] != "" {
			a.contextProject(req)
			return a.ui("stats"), nil
		}
		days := 7
		if v := req.Opts["days"]; v != "" {
			var err error
			if days, err = strconv.Atoi(v); err != nil || days < 1 {
				return "", fmt.Errorf("--days needs a positive number, got %q", v)
			}
		}
		key := a.askerProject(req)
		st, err := a.sessions.waits.Stats(key, days, time.Now())
		if err != nil {
			return "", err
		}
		if req.Opts["json"] != "" {
			return jsonLine(st)
		}
		return formatStats(key, st), nil
	case "digest-add":
		return a.controlDigestAdd(req)
	}
	if out, ok, err := a.controlApp(ctx, req); ok {
		return out, err
	}
	return "", fmt.Errorf("unknown command %q", req.Cmd)
}

func (a *App) controlBrowser(ctx context.Context, req control.Request) (string, error) {
	if len(req.Args) == 0 {
		a.contextProject(req)
		return a.ui("browser"), nil
	}
	sub, args := req.Args[0], req.Args[1:]
	if sub == "help" {
		return control.BrowserHelp, nil
	}
	id, err := a.askerSession(req)
	if err != nil {
		return "", err
	}
	if !a.browsers.Available() {
		return "", errors.New("no Brave, Chrome, Chromium or Edge browser found")
	}
	if sub != "screenshot" {
		return a.browsers.agent(ctx, id, sub, args, req.Opts)
	}
	ref := ""
	if len(args) > 0 {
		ref = args[0]
	}
	if _, err := a.browsers.tabFor(ctx, id); err != nil {
		return "", err
	}
	png, err := a.browsers.Screenshot(ctx, id, req.Opts["full"] != "", ref)
	if err != nil {
		return "", err
	}
	item, err := a.evidence.AddImage(id, png, req.Opts["caption"], "agent")
	if err != nil {
		return "", err
	}
	a.evidenceChanged(id, true)
	return mediaFile(map[string]string{evidenceFolder: a.evidence.dir}, item.URL), nil
}

func (a *App) controlShow(id string, req control.Request) (string, error) {
	caption := req.Opts["caption"]
	if text := req.Opts["text"]; text != "" {
		if _, err := a.evidence.AddText(id, text, caption, "agent"); err != nil {
			return "", err
		}
		a.evidenceChanged(id, true)
		return "filed as evidence", nil
	}
	if len(req.Args) != 1 {
		return "", errors.New("usage: agentos show <file> [--caption <text>] or agentos show --text <words>")
	}
	path := req.Args[0]
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a file", path)
	}
	if info.Size() > maxImageSize {
		return "", fmt.Errorf("%s is over 10 MB", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	if _, err = a.evidence.AddImage(id, data, caption, "agent"); err != nil {
		if len(data) > maxTextCard || !utf8.Valid(data) {
			return "", fmt.Errorf("%s is neither a picture nor a text file under 64 KB", path)
		}
		if caption == "" {
			caption = filepath.Base(path)
		}
		if _, err = a.evidence.AddText(id, string(data), caption, "agent"); err != nil {
			return "", err
		}
	}
	a.evidenceChanged(id, true)
	return "filed as evidence", nil
}

func (a *App) controlDigestAdd(req control.Request) (string, error) {
	key := req.Project
	if key == "" {
		return "", errors.New("this command is for the digest run")
	}
	added, err := a.digests.Add(key, DigestItem{Title: req.Opts["title"], Why: req.Opts["why"], URL: req.Opts["url"], Source: req.Opts["source"]})
	if err != nil {
		return "", err
	}
	a.emitDigest(key)
	if !added {
		return "already in the digest", nil
	}
	return "added", nil
}

func jsonLine(v any) (string, error) {
	out, err := json.Marshal(v)
	return string(out), err
}

func formatStats(key string, st Stats) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Waits on the owner in %s over the last %d days: %d, %s in all, median %s.\n",
		key, st.Days, st.Total, shortDuration(st.TotalWaitMs), shortDuration(st.MedianWaitMs))
	if st.Total == 0 {
		return strings.TrimRight(b.String(), "\n")
	}
	b.WriteString("Most frequent causes first (count, time waited):\n")
	for _, c := range st.ByCause[:min(len(st.ByCause), 20)] {
		fmt.Fprintf(&b, "  %3dx  %-8s %-40s %s\n", c.Count, c.Kind, c.Label, shortDuration(c.TotalWaitMs))
	}
	return strings.TrimRight(b.String(), "\n")
}

func shortDuration(ms int64) string {
	d := time.Duration(ms) * time.Millisecond
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", ms)
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}
	return fmt.Sprintf("%dh %dm", int(d.Hours()), int(d.Minutes())%60)
}
