package browser

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/util"
)

const (
	pageProbeLimit = 3 * time.Second
	newPageLimit   = 10 * time.Second
)

// page is one browser page a session owns: its first, the tabs and popups it opens, and any tab the owner opens in
// the session's window. Its fields are guarded by the mutex of its session's tab.
type page struct {
	t        *tab
	targetID string

	sessionID  string // the protocol session; empty until attached
	windowID   int    // 0 until asked for
	mainFrame  string
	url        string
	title      string // without the session's label
	loading    bool
	loadSeq    int
	loadedAt   int64
	canBack    bool
	canForward bool
	shown      bool // document.visibilityState was "visible" at the last look: the selected tab of its window
	requests   map[string]string
	dialog     *time.Timer // set while a JavaScript dialog is open
}

func (t *tab) newPage(targetID string) *page {
	return &page{t: t, targetID: targetID, url: "about:blank", requests: map[string]string{}}
}

func (pg *page) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	pg.t.mu.Lock()
	sessionID := pg.sessionID
	pg.t.mu.Unlock()
	return pg.t.proc.cdp.call(ctx, sessionID, method, params)
}

// stopDialog forgets the open dialog's timer. The tab's mutex must be held.
func (pg *page) stopDialog() {
	if pg.dialog != nil {
		pg.dialog.Stop()
		pg.dialog = nil
	}
}

// activeLocked is the page the session shows: the newest whose document is visible, else the newest. The tab's
// mutex must be held.
func (t *tab) activeLocked() *page {
	for _, pg := range slices.Backward(t.pages) {
		if pg.shown {
			return pg
		}
	}
	if n := len(t.pages); n > 0 {
		return t.pages[n-1]
	}
	return nil
}

var errNoPage = errors.New("this session has no open page")

func (t *tab) activePage() (*page, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if pg := t.activeLocked(); pg != nil {
		return pg, nil
	}
	return nil, errNoPage
}

// newest is the session's most recently opened page.
func (t *tab) newest() (*page, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if n := len(t.pages); n > 0 {
		return t.pages[n-1], nil
	}
	return nil, errNoPage
}

// call runs a protocol method in the active page.
func (t *tab) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	pg, err := t.activePage()
	if err != nil {
		return nil, err
	}
	return pg.call(ctx, method, params)
}

// eval runs JavaScript in the active page.
func (t *tab) eval(ctx context.Context, expression string) (json.RawMessage, error) {
	pg, err := t.activePage()
	if err != nil {
		return nil, err
	}
	return pg.eval(ctx, expression)
}

// change runs fn, then brings the session's state in line with its active page and publishes it if that changed it.
// It reports whether the state changed. fn runs with the tab's mutex held.
func (t *tab) change(fn func()) bool {
	t.mu.Lock()
	before := t.state
	fn()
	t.state.Pages = len(t.pages)
	if pg := t.activeLocked(); pg != nil {
		t.state.URL, t.state.Title, t.state.Loading, t.state.LoadedAt = pg.url, pg.title, pg.loading, pg.loadedAt
		t.state.CanGoBack, t.state.CanGoForward = pg.canBack, pg.canForward
	}
	changed := !reflect.DeepEqual(before, t.state)
	t.mu.Unlock()
	if changed {
		t.publish()
	}
	return changed
}

// attach subscribes to a page's events and gives its documents the title label.
func (t *tab) attach(ctx context.Context, pg *page) error {
	raw, err := t.proc.cdp.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": pg.targetID, "flatten": true})
	if err != nil {
		return err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &attached)
	t.mu.Lock()
	pg.sessionID = attached.SessionID
	t.mu.Unlock()

	for _, method := range []string{"Page.enable", "Runtime.enable", "Network.enable"} {
		if _, err := pg.call(ctx, method, nil); err != nil {
			return err
		}
	}
	if t.b.headless {
		if _, err := pg.call(ctx, "Emulation.setDeviceMetricsOverride", t.metrics()); err != nil {
			return err
		}
	} else {
		if _, err := pg.call(ctx, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": titleScript(t.prefix)}); err != nil {
			return err
		}
		// The page may have loaded already.
		if _, err := pg.eval(ctx, titleScript(t.prefix)); err != nil {
			return err
		}
	}
	if raw, err := pg.call(ctx, "Page.getFrameTree", nil); err == nil {
		var tree struct {
			FrameTree struct {
				Frame struct {
					ID string `json:"id"`
				} `json:"frame"`
			} `json:"frameTree"`
		}
		if json.Unmarshal(raw, &tree) == nil {
			t.mu.Lock()
			pg.mainFrame = tree.FrameTree.Frame.ID
			t.mu.Unlock()
		}
	}
	return nil
}

// refreshMeta reads the title, address and visibility from the page: the browser reports a title only as the
// address until the page has loaded, and not when a script changes it. A page that loaded before agentos listened
// (a tab a script opened) gets its load time from here.
func (pg *page) refreshMeta(ctx context.Context) {
	raw, err := pg.eval(ctx, `document.title + "\n" + location.href + "\n" + document.visibilityState + "\n" + document.readyState`)
	var all string
	if err != nil || json.Unmarshal(raw, &all) != nil {
		return
	}
	title, rest, _ := strings.Cut(all, "\n")
	addr, rest, _ := strings.Cut(rest, "\n")
	visibility, readyState, _ := strings.Cut(rest, "\n")
	t := pg.t
	title = t.pageTitle(title)
	t.change(func() {
		pg.title = title
		if addr != "" {
			pg.url = addr
		}
		pg.shown = visibility == "visible"
		if readyState == "complete" && pg.loadedAt == 0 {
			pg.loadedAt = time.Now().UnixMilli()
		}
	})
}

// refreshAll looks at every page of the session, so the active one is current.
func (t *tab) refreshAll(ctx context.Context) {
	t.mu.Lock()
	pages := slices.Clone(t.pages)
	t.mu.Unlock()
	for _, pg := range pages {
		probe, cancel := context.WithTimeout(ctx, pageProbeLimit)
		pg.refreshMeta(probe)
		cancel()
	}
}

func (t *tab) refreshActive(ctx context.Context) (*page, error) {
	t.refreshAll(ctx)
	return t.activePage()
}

// watch keeps the pages' titles and the active page current while the session has a browser.
func (t *tab) watch() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if t.b.tab(t.id) != t {
			return
		}
		t.refreshAll(context.Background())
	}
}

// refreshHistory updates the back and forward flags.
func (pg *page) refreshHistory() {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	h, err := pg.history(ctx)
	if err != nil {
		return
	}
	pg.t.change(func() {
		pg.canBack, pg.canForward = h.Current > 0, h.Current < len(h.Entries)-1
	})
}

type history struct {
	Current int `json:"currentIndex"`
	Entries []struct {
		ID int `json:"id"`
	} `json:"entries"`
}

func (pg *page) history(ctx context.Context) (history, error) {
	raw, err := pg.call(ctx, "Page.getNavigationHistory", nil)
	var h history
	if err == nil {
		err = json.Unmarshal(raw, &h)
	}
	return h, err
}

func (pg *page) step(ctx context.Context, delta int) error {
	h, err := pg.history(ctx)
	if err != nil {
		return err
	}
	i := h.Current + delta
	if i < 0 || i >= len(h.Entries) {
		return errors.New("no page to go to")
	}
	_, err = pg.call(ctx, "Page.navigateToHistoryEntry", map[string]any{"entryId": h.Entries[i].ID})
	return err
}

// goTo starts loading an address in the page.
func (pg *page) goTo(ctx context.Context, address string) error {
	address, err := webAddress(address)
	if err != nil {
		return err
	}
	raw, err := pg.call(ctx, "Page.navigate", map[string]any{"url": address})
	if err != nil {
		return err
	}
	var res struct {
		ErrorText string `json:"errorText"`
	}
	_ = json.Unmarshal(raw, &res)
	if res.ErrorText != "" {
		return fmt.Errorf("could not open %s: %s", address, res.ErrorText)
	}
	return nil
}

// activate selects the page in its window and raises the window.
func (pg *page) activate(ctx context.Context) error {
	_, err := pg.t.proc.cdp.call(ctx, "", "Target.activateTarget", map[string]any{"targetId": pg.targetID})
	return err
}

// window is the id of the browser window the page is in.
func (pg *page) window(ctx context.Context) (int, error) {
	pg.t.mu.Lock()
	id := pg.windowID
	pg.t.mu.Unlock()
	if id != 0 {
		return id, nil
	}
	id, err := windowOf(ctx, pg.t.proc, pg.targetID)
	if err == nil {
		pg.t.mu.Lock()
		pg.windowID = id
		pg.t.mu.Unlock()
	}
	return id, err
}

func windowOf(ctx context.Context, p *browserProc, targetID string) (int, error) {
	raw, err := p.cdp.call(ctx, "", "Browser.getWindowForTarget", map[string]any{"targetId": targetID})
	if err != nil {
		return 0, err
	}
	var res struct {
		WindowID int `json:"windowId"`
	}
	if err := json.Unmarshal(raw, &res); err != nil || res.WindowID == 0 {
		return 0, errors.New("no window for the page")
	}
	return res.WindowID, nil
}

// eval runs JavaScript in the page and returns its JSON result.
func (pg *page) eval(ctx context.Context, expression string) (json.RawMessage, error) {
	return pg.evaluate(ctx, expression, false)
}

// evaluate is eval; userGesture runs the script as if the user had just clicked, so the browser allows a popup.
func (pg *page) evaluate(ctx context.Context, expression string, userGesture bool) (json.RawMessage, error) {
	raw, err := pg.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true, "userGesture": userGesture})
	if err != nil {
		return nil, err
	}
	var res struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		Exception *struct {
			Text      string `json:"text"`
			Exception struct {
				Description string `json:"description"`
			} `json:"exception"`
		} `json:"exceptionDetails"`
	}
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("reading the result: %w", err)
	}
	if res.Exception != nil {
		msg := util.FirstLine(res.Exception.Exception.Description)
		if msg == "" {
			msg = res.Exception.Text
		}
		return nil, errors.New(msg)
	}
	return res.Result.Value, nil
}

// line is the page's title and address, read from the page itself.
func (pg *page) line(ctx context.Context) string {
	raw, err := pg.eval(ctx, `document.title + "\n" + location.href`)
	var both string
	if err != nil || json.Unmarshal(raw, &both) != nil {
		pg.t.mu.Lock()
		defer pg.t.mu.Unlock()
		return pg.title + " - " + pg.url
	}
	title, addr, _ := strings.Cut(both, "\n")
	return pg.t.pageTitle(title) + " - " + addr
}

// targetInfo is what the browser says of a target.
type targetInfo struct {
	TargetID string `json:"targetId"`
	Type     string `json:"type"`
	OpenerID string `json:"openerId"`
	URL      string `json:"url"`
	Title    string `json:"title"`
}

// pageBy finds the page, of any session, that match accepts. match runs with its tab's mutex held.
func (b *Browsers) pageBy(match func(*page) bool) *page {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, t := range b.tabs {
		t.mu.Lock()
		i := slices.IndexFunc(t.pages, match)
		var pg *page
		if i >= 0 {
			pg = t.pages[i]
		}
		t.mu.Unlock()
		if pg != nil {
			return pg
		}
	}
	return nil
}

func (b *Browsers) pageOfTarget(targetID string) *page {
	return b.pageBy(func(pg *page) bool { return pg.targetID == targetID })
}

// ownerOf is the session a new page belongs to: the one that owns the page which opened it, else the one with a page
// in the same window (a tab the owner opened by hand); nil when the page is not the app's. Headless has no windows
// to tell apart.
func (b *Browsers) ownerOf(ctx context.Context, p *browserProc, info targetInfo) *tab {
	if info.OpenerID != "" {
		if opener := b.pageOfTarget(info.OpenerID); opener != nil && opener.t.proc == p {
			return opener.t
		}
	}
	if b.headless {
		return nil
	}
	window, err := windowOf(ctx, p, info.TargetID)
	if err != nil {
		return nil
	}
	b.mu.Lock()
	var tabs []*tab
	for _, t := range b.tabs {
		if t.proc == p {
			tabs = append(tabs, t)
		}
	}
	b.mu.Unlock()
	for _, t := range tabs {
		t.mu.Lock()
		pages := slices.Clone(t.pages)
		t.mu.Unlock()
		for _, pg := range pages {
			if id, err := pg.window(ctx); err == nil && id == window {
				return t
			}
		}
	}
	return nil
}

// claim adds the page to the session, unless some session has it already or the session has closed.
func (b *Browsers) claim(t *tab, pg *page) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tabs[t.id] != t {
		return false
	}
	for _, o := range b.tabs {
		o.mu.Lock()
		known := slices.ContainsFunc(o.pages, func(x *page) bool { return x.targetID == pg.targetID })
		o.mu.Unlock()
		if known {
			return false
		}
	}
	t.mu.Lock()
	t.pages = append(t.pages, pg)
	t.mu.Unlock()
	return true
}

// adopt takes a page that appeared in the browser into the session it belongs to, if any.
func (b *Browsers) adopt(p *browserProc, info targetInfo) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*callLimit)
	defer cancel()
	t := b.ownerOf(ctx, p, info)
	if t == nil {
		return
	}
	pg := t.newPage(info.TargetID)
	if !b.claim(t, pg) {
		return
	}
	if err := t.attach(ctx, pg); err != nil {
		t.drop(pg)
		return
	}
	t.mu.Lock()
	casting := t.visible
	t.mu.Unlock()
	if b.headless && casting {
		_ = t.startScreencast(ctx)
	}
	t.refreshAll(ctx)
	t.change(func() {})
}

// drop removes a page from the session and reports how many are left.
func (t *tab) drop(pg *page) int {
	t.mu.Lock()
	t.pages = slices.DeleteFunc(t.pages, func(x *page) bool { return x == pg })
	pg.stopDialog()
	left := len(t.pages)
	t.mu.Unlock()
	return left
}

// pageGone forgets a page that closed. The session's browser counts as closed when its last page is gone.
func (b *Browsers) pageGone(targetID string) {
	pg := b.pageOfTarget(targetID)
	if pg == nil {
		return
	}
	t := pg.t
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	if t.drop(pg) == 0 {
		b.Close(ctx, t.id)
		return
	}
	t.refreshAll(ctx)
	t.change(func() {})
}

// event takes a protocol event. It runs in the connection's read loop, so it never waits for a reply.
func (b *Browsers) event(p *browserProc, sessionID, method string, params json.RawMessage) {
	if sessionID != "" {
		pg := b.pageBy(func(pg *page) bool { return pg.sessionID == sessionID })
		if pg != nil {
			pg.event(method, params)
		}
		return
	}
	switch method {
	case "Target.targetCreated":
		var ev struct {
			TargetInfo targetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &ev) == nil && ev.TargetInfo.Type == "page" {
			go b.adopt(p, ev.TargetInfo)
		}
	case "Target.targetDestroyed":
		var ev struct {
			TargetID string `json:"targetId"`
		}
		if json.Unmarshal(params, &ev) == nil {
			go b.pageGone(ev.TargetID)
		}
	case "Target.targetInfoChanged":
		var ev struct {
			TargetInfo targetInfo `json:"targetInfo"`
		}
		if json.Unmarshal(params, &ev) != nil {
			return
		}
		pg := b.pageOfTarget(ev.TargetInfo.TargetID)
		if pg == nil {
			return
		}
		title := pg.t.pageTitle(ev.TargetInfo.Title)
		changed := pg.t.change(func() { pg.url, pg.title = ev.TargetInfo.URL, title })
		if changed {
			go pg.refreshHistory()
		}
	}
}

func (pg *page) event(method string, params json.RawMessage) {
	t := pg.t
	switch method {
	case "Page.frameNavigated":
		var ev struct {
			Frame struct {
				ID       string `json:"id"`
				ParentID string `json:"parentId"`
			} `json:"frame"`
		}
		if json.Unmarshal(params, &ev) == nil && ev.Frame.ParentID == "" {
			t.mu.Lock()
			pg.mainFrame = ev.Frame.ID
			t.mu.Unlock()
		}
	case "Page.frameStartedLoading", "Page.frameStoppedLoading":
		var ev struct {
			FrameID string `json:"frameId"`
		}
		if json.Unmarshal(params, &ev) != nil {
			return
		}
		t.mu.Lock()
		other := ev.FrameID != pg.mainFrame && pg.mainFrame != ""
		t.mu.Unlock()
		if other {
			return
		}
		loading := method == "Page.frameStartedLoading"
		t.change(func() { pg.loading = loading })
		if !loading {
			go pg.refreshHistory()
			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), callLimit)
				defer cancel()
				pg.refreshMeta(ctx)
			}()
		}
	case "Page.loadEventFired":
		t.change(func() {
			pg.loadSeq++
			pg.loadedAt = time.Now().UnixMilli()
		})
	case "Page.screencastFrame":
		pg.frame(params)
	case "Page.javascriptDialogOpening":
		var ev struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		}
		if json.Unmarshal(params, &ev) == nil {
			pg.dialogOpened(ev.Type, ev.Message)
		}
	case "Page.javascriptDialogClosed":
		t.mu.Lock()
		pg.stopDialog()
		t.mu.Unlock()
	case "Runtime.consoleAPICalled":
		var ev struct {
			Type string `json:"type"`
			Args []struct {
				Value       json.RawMessage `json:"value"`
				Description string          `json:"description"`
			} `json:"args"`
		}
		if json.Unmarshal(params, &ev) == nil && ev.Type == "error" {
			var parts []string
			for _, a := range ev.Args {
				var s string
				if json.Unmarshal(a.Value, &s) == nil && s != "" {
					parts = append(parts, s)
				} else if a.Description != "" {
					parts = append(parts, a.Description)
				} else if len(a.Value) > 0 {
					parts = append(parts, string(a.Value))
				}
			}
			t.logConsole("console.error: " + strings.Join(parts, " "))
		}
	case "Runtime.exceptionThrown":
		var ev struct {
			ExceptionDetails struct {
				Text      string `json:"text"`
				Exception struct {
					Description string `json:"description"`
				} `json:"exception"`
			} `json:"exceptionDetails"`
		}
		if json.Unmarshal(params, &ev) == nil {
			msg := ev.ExceptionDetails.Exception.Description
			if msg == "" {
				msg = ev.ExceptionDetails.Text
			}
			t.logConsole("uncaught: " + util.FirstLine(msg))
		}
	case "Network.requestWillBeSent":
		var ev struct {
			RequestID string `json:"requestId"`
			Request   struct {
				URL    string `json:"url"`
				Method string `json:"method"`
			} `json:"request"`
		}
		if json.Unmarshal(params, &ev) == nil {
			t.mu.Lock()
			pg.requests[ev.RequestID] = ev.Request.Method + " " + ev.Request.URL
			t.mu.Unlock()
		}
	case "Network.responseReceived":
		var ev struct {
			RequestID string `json:"requestId"`
			Response  struct {
				URL    string `json:"url"`
				Status int    `json:"status"`
			} `json:"response"`
		}
		if json.Unmarshal(params, &ev) == nil && ev.Response.Status >= 400 {
			t.mu.Lock()
			req := pg.requests[ev.RequestID]
			t.mu.Unlock()
			if req == "" {
				req = ev.Response.URL
			}
			t.logConsole(fmt.Sprintf("request failed: %s -> %d", req, ev.Response.Status))
		}
	case "Network.loadingFailed":
		var ev struct {
			RequestID string `json:"requestId"`
			ErrorText string `json:"errorText"`
			Canceled  bool   `json:"canceled"`
		}
		if json.Unmarshal(params, &ev) == nil && !ev.Canceled {
			t.mu.Lock()
			req := pg.requests[ev.RequestID]
			t.mu.Unlock()
			t.logConsole(fmt.Sprintf("request failed: %s -> %s", req, ev.ErrorText))
		}
	}
}

// frame passes on a live picture from the page when it is the active one, and always acknowledges it.
func (pg *page) frame(params json.RawMessage) {
	var ev struct {
		Data      string `json:"data"`
		SessionID int    `json:"sessionId"`
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	pg.t.mu.Lock()
	active := pg.t.activeLocked() == pg
	pg.t.mu.Unlock()
	if active {
		pg.t.sendFrame(ev.Data)
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), callLimit)
		defer cancel()
		_, _ = pg.call(ctx, "Page.screencastFrameAck", map[string]any{"sessionId": ev.SessionID})
	}()
}

// dialogOpened dismisses the dialog if nobody answers it within dialogWait.
func (pg *page) dialogOpened(kind, message string) {
	t := pg.t
	t.mu.Lock()
	defer t.mu.Unlock()
	pg.stopDialog()
	wait := t.b.dialogWait
	pg.dialog = time.AfterFunc(wait, func() {
		ctx, cancel := context.WithTimeout(context.Background(), callLimit)
		defer cancel()
		if _, err := pg.call(ctx, "Page.handleJavaScriptDialog", map[string]any{"accept": false}); err != nil {
			return
		}
		line := util.FirstLine(message)
		if r := []rune(line); len(r) > dialogShown {
			line = string(r[:dialogShown]) + "..."
		}
		t.logConsole(fmt.Sprintf("dialog dismissed after %s: %s %q", wait, kind, line))
	})
}

func (t *tab) pagesCopy() []*page {
	t.mu.Lock()
	defer t.mu.Unlock()
	return slices.Clone(t.pages)
}
