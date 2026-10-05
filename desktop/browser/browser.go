// Package browser runs one hidden browser per project and one tab per session.
package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/nednella/agentos/internal/session"
)

const (
	defaultViewWidth  = 1280
	defaultViewHeight = 800
	consoleKept       = 200
	browserStartLimit = 15 * time.Second
	callLimit         = 15 * time.Second
	closeGrace        = time.Minute
)

// BrowserState is what the front end shows of a session's tab.
type BrowserState struct {
	ID           string `json:"id"`
	Open         bool   `json:"open"`
	URL          string `json:"url"`
	Title        string `json:"title"`
	Loading      bool   `json:"loading"`
	CanGoBack    bool   `json:"canGoBack"`
	CanGoForward bool   `json:"canGoForward"`
	Error        string `json:"error"`
}

// BrowserInput is one mouse, wheel, key or paste event from the user, in CSS pixels of the page viewport.
type BrowserInput struct {
	Type       string  `json:"type"`
	Action     string  `json:"action"`
	X          float64 `json:"x"`
	Y          float64 `json:"y"`
	Button     string  `json:"button"`
	ClickCount int     `json:"clickCount"`
	Modifiers  int     `json:"modifiers"`
	DeltaX     float64 `json:"deltaX"`
	DeltaY     float64 `json:"deltaY"`
	Key        string  `json:"key"`
	Code       string  `json:"code"`
	Text       string  `json:"text"`
}

var browserPaths = []string{
	"/Applications/Brave Browser.app/Contents/MacOS/Brave Browser",
	"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
	"/Applications/Chromium.app/Contents/MacOS/Chromium",
	"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
}

// findBrowser returns a Chromium based browser to drive: AGENTOS_BROWSER, else Brave, Chrome, Chromium or Edge.
func findBrowser() string {
	if p := os.Getenv("AGENTOS_BROWSER"); p != "" {
		return p
	}
	for _, p := range browserPaths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Browsers runs one hidden browser per project and one tab per session.
type Browsers struct {
	dataDir  string
	binary   string
	emit     func(event string, payload any)
	grace    time.Duration
	urlFor   func(sessionID string) string // the page a new tab opens first
	onTabs   func()                        // a tab opened or closed
	launchMu sync.Mutex

	mu    sync.Mutex
	procs map[string]*browserProc // by project key
	tabs  map[string]*tab         // by session id
}

// New runs the browsers with their profiles under dataDir.
func New(dataDir string, emit func(string, any)) *Browsers {
	return &Browsers{
		dataDir: dataDir, binary: findBrowser(), emit: emit, grace: closeGrace,
		urlFor: func(string) string { return "" }, onTabs: func() {},
		procs: map[string]*browserProc{}, tabs: map[string]*tab{},
	}
}

// Available says whether there is a browser to drive.
func (b *Browsers) Available() bool { return b != nil && b.binary != "" }

// Has says whether the session has a tab.
func (b *Browsers) Has(id string) bool {
	if b == nil {
		return false
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tabs[id] != nil
}

type browserProc struct {
	key  string
	cmd  *exec.Cmd
	cdp  *cdp
	gone chan struct{}

	tabs  int
	timer *time.Timer
}

// tab is one session's page.
type tab struct {
	b         *Browsers
	proc      *browserProc
	id        string
	targetID  string
	sessionID string

	mu        sync.Mutex
	state     BrowserState
	mainFrame string
	width     int
	height    int
	visible   bool
	loadSeq   int
	console   []string
	requests  map[string]string
}

func (t *tab) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	return t.proc.cdp.call(ctx, t.sessionID, method, params)
}

func (t *tab) snapshotState() BrowserState {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state
}

// publish sends the tab's state to the front end.
func (t *tab) publish() { t.b.emit("browser:state", t.snapshotState()) }

func (t *tab) logConsole(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if len(t.console) < consoleKept {
		t.console = append(t.console, line)
	}
}

func (t *tab) drainConsole() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := t.console
	t.console = nil
	return out
}

// start launches the project's browser unless it runs already.
func (b *Browsers) start(ctx context.Context, key string) (*browserProc, error) {
	b.launchMu.Lock()
	defer b.launchMu.Unlock()
	b.mu.Lock()
	p := b.procs[key]
	b.mu.Unlock()
	if p != nil {
		return p, nil
	}
	if b.binary == "" {
		return nil, errors.New("no Brave, Chrome, Chromium or Edge browser found")
	}
	profile := filepath.Join(b.dataDir, key, "browser")
	if err := os.MkdirAll(profile, 0o700); err != nil {
		return nil, fmt.Errorf("creating the browser profile: %w", err)
	}
	portFile := filepath.Join(profile, "DevToolsActivePort")
	_ = os.Remove(portFile)
	stopStrayBrowser(profile)
	cmd := exec.Command(b.binary, "--headless=new", "--remote-debugging-port=0", "--user-data-dir="+profile,
		"--no-first-run", "--no-default-browser-check", "--hide-scrollbars", "about:blank")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting the browser: %w", err)
	}
	p = &browserProc{key: key, cmd: cmd, gone: make(chan struct{})}
	go func() {
		_ = cmd.Wait()
		close(p.gone)
		b.exited(p)
	}()

	wsURL, err := waitForEndpoint(ctx, portFile, p.gone)
	if err != nil {
		p.kill()
		return nil, err
	}
	c, err := dialCDP(ctx, wsURL, func(sessionID, method string, params json.RawMessage) { b.event(p, sessionID, method, params) })
	if err != nil {
		p.kill()
		return nil, err
	}
	p.cdp = c
	if _, err := c.call(ctx, "", "Target.setDiscoverTargets", map[string]any{"discover": true}); err != nil {
		p.kill()
		return nil, err
	}
	b.mu.Lock()
	b.procs[key] = p
	b.mu.Unlock()
	return p, nil
}

func waitForEndpoint(ctx context.Context, portFile string, gone <-chan struct{}) (string, error) {
	deadline := time.After(browserStartLimit)
	for {
		if data, err := os.ReadFile(portFile); err == nil {
			if lines := strings.Split(strings.TrimSpace(string(data)), "\n"); len(lines) >= 2 {
				return "ws://" + net.JoinHostPort("127.0.0.1", strings.TrimSpace(lines[0])) + strings.TrimSpace(lines[1]), nil
			}
		}
		select {
		case <-gone:
			return "", errors.New("the browser stopped at start; is its profile in use by another copy?")
		case <-deadline:
			return "", errors.New("the browser did not start in time")
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (p *browserProc) kill() {
	if p.cmd.Process != nil {
		// The whole group: the browser's helper processes go with it.
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	}
	if p.cdp != nil {
		p.cdp.close()
	}
}

// exited forgets a browser that stopped and marks its tabs closed.
func (b *Browsers) exited(p *browserProc) {
	b.mu.Lock()
	if b.procs[p.key] == p {
		delete(b.procs, p.key)
	}
	var lost []*tab
	for id, t := range b.tabs {
		if t.proc == p {
			lost = append(lost, t)
			delete(b.tabs, id)
		}
	}
	b.mu.Unlock()
	for _, t := range lost {
		t.mu.Lock()
		t.state.Open = false
		t.mu.Unlock()
		t.publish()
	}
	if len(lost) > 0 {
		b.onTabs()
	}
}

// Open gives the session a tab, starting the project's browser if needed, and goes to url when it is not empty.
func (b *Browsers) Open(ctx context.Context, id, url string) (BrowserState, error) {
	if t := b.tab(id); t != nil {
		if url != "" {
			if err := b.Goto(ctx, id, url); err != nil {
				return BrowserState{}, err
			}
		}
		return t.snapshotState(), nil
	}
	key := keyOf(id)
	p, err := b.start(ctx, key)
	if err != nil {
		return BrowserState{ID: id, Error: err.Error()}, err
	}
	raw, err := p.cdp.call(ctx, "", "Target.createTarget", map[string]any{"url": "about:blank"})
	if err != nil {
		return BrowserState{}, err
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &created)
	raw, err = p.cdp.call(ctx, "", "Target.attachToTarget", map[string]any{"targetId": created.TargetID, "flatten": true})
	if err != nil {
		return BrowserState{}, err
	}
	var attached struct {
		SessionID string `json:"sessionId"`
	}
	_ = json.Unmarshal(raw, &attached)
	t := &tab{b: b, proc: p, id: id, targetID: created.TargetID, sessionID: attached.SessionID,
		width: defaultViewWidth, height: defaultViewHeight, requests: map[string]string{},
		state: BrowserState{ID: id, Open: true, URL: "about:blank"}}
	b.mu.Lock()
	b.tabs[id] = t
	p.tabs++
	if p.timer != nil {
		p.timer.Stop()
		p.timer = nil
	}
	b.mu.Unlock()
	b.onTabs()

	for _, setup := range []struct {
		method string
		params any
	}{
		{"Page.enable", nil},
		{"Runtime.enable", nil},
		{"Network.enable", nil},
		{"Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": keepInTab}},
		{"Emulation.setDeviceMetricsOverride", t.metrics()},
	} {
		if _, err := t.call(ctx, setup.method, setup.params); err != nil {
			b.Close(ctx, id)
			return BrowserState{}, err
		}
	}
	if raw, err := t.call(ctx, "Page.getFrameTree", nil); err == nil {
		var tree struct {
			FrameTree struct {
				Frame struct {
					ID string `json:"id"`
				} `json:"frame"`
			} `json:"frameTree"`
		}
		if json.Unmarshal(raw, &tree) == nil {
			t.mu.Lock()
			t.mainFrame = tree.FrameTree.Frame.ID
			t.mu.Unlock()
		}
	}
	go t.watch()
	if url == "" {
		url = b.urlFor(id)
	}
	if url != "" {
		if err := b.Goto(ctx, id, url); err != nil {
			t.publish()
			return t.snapshotState(), nil
		}
	}
	t.publish()
	return t.snapshotState(), nil
}

// keepInTab makes links and window.open stay in the session's one tab.
const keepInTab = `(() => {
  const own = (t) => !t || ['_self', '_top', '_parent'].includes(t)
  document.addEventListener('click', (e) => {
    const a = e.target && e.target.closest && e.target.closest('a[target]')
    if (a && !own(a.target)) a.target = '_self'
  }, true)
  document.addEventListener('submit', (e) => { if (e.target && e.target.target && !own(e.target.target)) e.target.target = '_self' }, true)
  window.open = (url) => { if (url) location.href = url; return window }
})()`

func (t *tab) metrics() map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	return map[string]any{"width": t.width, "height": t.height, "deviceScaleFactor": 1, "mobile": false}
}

func (b *Browsers) tab(id string) *tab {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.tabs[id]
}

func (b *Browsers) need(id string) (*tab, error) {
	if t := b.tab(id); t != nil {
		return t, nil
	}
	return nil, errors.New("this session has no browser tab yet")
}

// State is the session's tab state; a session without a tab reports it closed.
func (b *Browsers) State(id string) BrowserState {
	if t := b.tab(id); t != nil {
		return t.snapshotState()
	}
	return BrowserState{ID: id}
}

// webAddress turns what the user typed into an address the browser can open.
func webAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", errors.New("no address")
	}
	if strings.HasPrefix(strings.ToLower(s), "javascript:") {
		return "", errors.New("javascript: addresses are not allowed")
	}
	if strings.Contains(s, "://") || strings.HasPrefix(s, "about:") || strings.HasPrefix(s, "data:") {
		return s, nil
	}
	host := s
	if i := strings.IndexAny(host, "/?#"); i >= 0 {
		host = host[:i]
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	if host == "localhost" || strings.HasSuffix(host, ".local") || net.ParseIP(strings.Trim(host, "[]")) != nil {
		return "http://" + s, nil
	}
	return "https://" + s, nil
}

// Goto starts loading an address in the session's tab.
func (b *Browsers) Goto(ctx context.Context, id, address string) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	address, err = webAddress(address)
	if err != nil {
		return err
	}
	raw, err := t.call(ctx, "Page.navigate", map[string]any{"url": address})
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

// Nav goes back or forward, reloads, or stops loading.
func (b *Browsers) Nav(ctx context.Context, id, action string) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	switch action {
	case "reload":
		_, err = t.call(ctx, "Page.reload", nil)
	case "stop":
		_, err = t.call(ctx, "Page.stopLoading", nil)
	case "back", "forward":
		err = t.step(ctx, map[string]int{"back": -1, "forward": 1}[action])
	default:
		err = fmt.Errorf("unknown navigation %q", action)
	}
	return err
}

type history struct {
	Current int `json:"currentIndex"`
	Entries []struct {
		ID int `json:"id"`
	} `json:"entries"`
}

func (t *tab) history(ctx context.Context) (history, error) {
	raw, err := t.call(ctx, "Page.getNavigationHistory", nil)
	var h history
	if err == nil {
		err = json.Unmarshal(raw, &h)
	}
	return h, err
}

func (t *tab) step(ctx context.Context, delta int) error {
	h, err := t.history(ctx)
	if err != nil {
		return err
	}
	i := h.Current + delta
	if i < 0 || i >= len(h.Entries) {
		return errors.New("no page to go to")
	}
	_, err = t.call(ctx, "Page.navigateToHistoryEntry", map[string]any{"entryId": h.Entries[i].ID})
	return err
}

// refreshMeta reads the title and address from the page: the browser reports a
// title only as the address until the page has loaded, and not when a script changes it.
func (t *tab) refreshMeta() {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	raw, err := t.eval(ctx, `document.title + "\n" + location.href`)
	var both string
	if err != nil || json.Unmarshal(raw, &both) != nil {
		return
	}
	title, addr, _ := strings.Cut(both, "\n")
	t.mu.Lock()
	changed := t.state.Title != title || (t.state.URL != addr && addr != "")
	t.state.Title = title
	if addr != "" {
		t.state.URL = addr
	}
	t.mu.Unlock()
	if changed {
		t.publish()
	}
}

// watch keeps the title current while the tab exists.
func (t *tab) watch() {
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for range tick.C {
		if t.b.tab(t.id) != t {
			return
		}
		t.refreshMeta()
	}
}

// refreshHistory updates the back and forward flags.
func (t *tab) refreshHistory() {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	h, err := t.history(ctx)
	if err != nil {
		return
	}
	t.mu.Lock()
	changed := t.state.CanGoBack != (h.Current > 0) || t.state.CanGoForward != (h.Current < len(h.Entries)-1)
	t.state.CanGoBack, t.state.CanGoForward = h.Current > 0, h.Current < len(h.Entries)-1
	t.mu.Unlock()
	if changed {
		t.publish()
	}
}

// Close closes the session's tab, and the project's browser once its last tab has been closed for a while.
func (b *Browsers) Close(ctx context.Context, id string) {
	b.mu.Lock()
	t := b.tabs[id]
	delete(b.tabs, id)
	b.mu.Unlock()
	if t == nil {
		return
	}
	_, _ = t.proc.cdp.call(ctx, "", "Target.closeTarget", map[string]any{"targetId": t.targetID})
	t.mu.Lock()
	t.state.Open = false
	t.visible = false
	t.mu.Unlock()
	t.publish()
	b.onTabs()

	b.mu.Lock()
	defer b.mu.Unlock()
	t.proc.tabs--
	if t.proc.tabs <= 0 && t.proc.timer == nil {
		p := t.proc
		p.timer = time.AfterFunc(b.grace, func() {
			b.mu.Lock()
			idle := p.tabs <= 0
			b.mu.Unlock()
			if idle {
				p.kill()
			}
		})
	}
}

// CloseAll stops every browser: the app is quitting.
func (b *Browsers) CloseAll() {
	b.mu.Lock()
	procs := slices.Collect(func(yield func(*browserProc) bool) {
		for _, p := range b.procs {
			if !yield(p) {
				return
			}
		}
	})
	b.mu.Unlock()
	for _, p := range procs {
		if p.timer != nil {
			p.timer.Stop()
		}
		p.kill()
		<-p.gone
	}
}

// event takes a protocol event. It runs in the connection's read loop, so it never waits for a reply.
func (b *Browsers) event(p *browserProc, sessionID, method string, params json.RawMessage) {
	if sessionID == "" {
		if method == "Target.targetInfoChanged" {
			var ev struct {
				TargetInfo struct {
					TargetID string `json:"targetId"`
					URL      string `json:"url"`
					Title    string `json:"title"`
				} `json:"targetInfo"`
			}
			if json.Unmarshal(params, &ev) != nil {
				return
			}
			b.mu.Lock()
			var t *tab
			for _, c := range b.tabs {
				if c.targetID == ev.TargetInfo.TargetID {
					t = c
				}
			}
			b.mu.Unlock()
			if t != nil {
				t.mu.Lock()
				changed := t.state.URL != ev.TargetInfo.URL || t.state.Title != ev.TargetInfo.Title
				t.state.URL, t.state.Title = ev.TargetInfo.URL, ev.TargetInfo.Title
				t.mu.Unlock()
				if changed {
					t.publish()
					go t.refreshHistory()
				}
			}
		}
		return
	}
	b.mu.Lock()
	var t *tab
	for _, c := range b.tabs {
		if c.sessionID == sessionID {
			t = c
		}
	}
	b.mu.Unlock()
	if t != nil {
		t.event(method, params)
	}
}

func (t *tab) event(method string, params json.RawMessage) {
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
			t.mainFrame = ev.Frame.ID
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
		if ev.FrameID != t.mainFrame && t.mainFrame != "" {
			t.mu.Unlock()
			return
		}
		loading := method == "Page.frameStartedLoading"
		changed := t.state.Loading != loading
		t.state.Loading = loading
		t.mu.Unlock()
		if changed {
			t.publish()
		}
		if !loading {
			go t.refreshHistory()
			go t.refreshMeta()
		}
	case "Page.loadEventFired":
		t.mu.Lock()
		t.loadSeq++
		t.mu.Unlock()
	case "Page.screencastFrame":
		t.frame(params)
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
			t.logConsole("uncaught: " + firstLine(msg))
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
			t.requests[ev.RequestID] = ev.Request.Method + " " + ev.Request.URL
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
			req := t.requests[ev.RequestID]
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
			req := t.requests[ev.RequestID]
			t.mu.Unlock()
			t.logConsole(fmt.Sprintf("request failed: %s -> %s", req, ev.ErrorText))
		}
	}
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(s), "\n")
	return line
}

// View turns the live picture on or off. Frames are only produced while it is on.
func (b *Browsers) View(ctx context.Context, id string, visible bool) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.visible = visible
	t.mu.Unlock()
	if !visible {
		_, err = t.call(ctx, "Page.stopScreencast", nil)
		return err
	}
	if err := t.startScreencast(ctx); err != nil {
		return err
	}
	// A page that does not change sends no frame, so send the current picture once.
	if raw, err := t.call(ctx, "Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 70}); err == nil {
		var shot struct {
			Data string `json:"data"`
		}
		if json.Unmarshal(raw, &shot) == nil {
			t.sendFrame(shot.Data)
		}
	}
	return nil
}

func (t *tab) startScreencast(ctx context.Context) error {
	t.mu.Lock()
	w, h := t.width, t.height
	t.mu.Unlock()
	_, err := t.call(ctx, "Page.startScreencast", map[string]any{
		"format": "jpeg", "quality": 70, "maxWidth": w, "maxHeight": h, "everyNthFrame": 1,
	})
	return err
}

func (t *tab) sendFrame(data string) {
	t.mu.Lock()
	visible, w, h := t.visible, t.width, t.height
	t.mu.Unlock()
	if visible {
		t.b.emit("browser:frame", map[string]any{"id": t.id, "data": data, "width": w, "height": h})
	}
}

func (t *tab) frame(params json.RawMessage) {
	var ev struct {
		Data      string `json:"data"`
		SessionID int    `json:"sessionId"`
	}
	if json.Unmarshal(params, &ev) != nil {
		return
	}
	t.sendFrame(ev.Data)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), callLimit)
		defer cancel()
		_, _ = t.call(ctx, "Page.screencastFrameAck", map[string]any{"sessionId": ev.SessionID})
	}()
}

// Resize sets the page viewport to the panel's size, in CSS pixels.
func (b *Browsers) Resize(ctx context.Context, id string, width, height int) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.width, t.height = min(max(width, 100), 4000), min(max(height, 100), 4000)
	casting := t.visible
	t.mu.Unlock()
	if _, err := t.call(ctx, "Emulation.setDeviceMetricsOverride", t.metrics()); err != nil {
		return err
	}
	if casting {
		return t.startScreencast(ctx)
	}
	return nil
}

var keyCodes = map[string]int{
	"Backspace": 8, "Tab": 9, "Enter": 13, "Escape": 27, " ": 32, "PageUp": 33, "PageDown": 34, "End": 35, "Home": 36,
	"ArrowLeft": 37, "ArrowUp": 38, "ArrowRight": 39, "ArrowDown": 40, "Delete": 46,
}

// editCommands are the editing shortcuts the browser runs for a meta key chord; headless has no menu to do it.
var editCommands = map[string]string{"a": "selectAll", "c": "copy", "x": "cut", "z": "undo"}

const (
	modAlt   = 1
	modCtrl  = 2
	modMeta  = 4
	modShift = 8
)

// Input sends one user event to the page.
func (b *Browsers) Input(ctx context.Context, id string, in BrowserInput) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	switch in.Type {
	case "mouse":
		typ := map[string]string{"move": "mouseMoved", "down": "mousePressed", "up": "mouseReleased"}[in.Action]
		if typ == "" {
			return fmt.Errorf("unknown mouse action %q", in.Action)
		}
		button := in.Button
		if button == "" {
			button = "none"
		}
		clicks := in.ClickCount
		if in.Action != "move" && clicks < 1 {
			clicks = 1
		}
		_, err = t.call(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type": typ, "x": in.X, "y": in.Y, "button": button, "clickCount": clicks, "modifiers": in.Modifiers,
		})
	case "wheel":
		_, err = t.call(ctx, "Input.dispatchMouseEvent", map[string]any{
			"type": "mouseWheel", "x": in.X, "y": in.Y, "deltaX": in.DeltaX, "deltaY": in.DeltaY, "modifiers": in.Modifiers,
		})
	case "key":
		err = t.key(ctx, in)
	case "paste":
		if in.Text != "" {
			_, err = t.call(ctx, "Input.insertText", map[string]any{"text": in.Text})
		}
	default:
		err = fmt.Errorf("unknown input type %q", in.Type)
	}
	return err
}

func (t *tab) key(ctx context.Context, in BrowserInput) error {
	params := map[string]any{"key": in.Key, "code": in.Code, "modifiers": in.Modifiers}
	vk, known := keyCodes[in.Key]
	if !known && len([]rune(in.Key)) == 1 {
		vk = int(strings.ToUpper(in.Key)[0])
	}
	if vk != 0 {
		params["windowsVirtualKeyCode"] = vk
	}
	if in.Action == "up" {
		params["type"] = "keyUp"
		_, err := t.call(ctx, "Input.dispatchKeyEvent", params)
		return err
	}
	text := in.Text
	if in.Key == "Enter" && text == "" {
		text = "\r"
	}
	chord := in.Modifiers&(modCtrl|modMeta) != 0
	if text != "" && !chord {
		params["type"], params["text"] = "keyDown", text
	} else {
		params["type"] = "rawKeyDown"
		if in.Modifiers&modMeta != 0 {
			if cmd, ok := editCommands[strings.ToLower(in.Key)]; ok {
				params["commands"] = []string{cmd}
			}
		}
	}
	_, err := t.call(ctx, "Input.dispatchKeyEvent", params)
	return err
}

// Screenshot returns a PNG of the viewport, the whole page, or one element's box (selector is an element ref).
func (b *Browsers) Screenshot(ctx context.Context, id string, full bool, ref string) ([]byte, error) {
	t, err := b.need(id)
	if err != nil {
		return nil, err
	}
	params := map[string]any{"format": "png"}
	switch {
	case ref != "":
		rect, err := t.elementBox(ctx, ref)
		if err != nil {
			return nil, err
		}
		params["clip"] = map[string]any{"x": rect.PageX, "y": rect.PageY, "width": rect.Width, "height": rect.Height, "scale": 1}
		params["captureBeyondViewport"] = true
	case full:
		raw, err := t.call(ctx, "Page.getLayoutMetrics", nil)
		if err != nil {
			return nil, err
		}
		var m struct {
			Size struct {
				Width  float64 `json:"width"`
				Height float64 `json:"height"`
			} `json:"cssContentSize"`
		}
		if err := json.Unmarshal(raw, &m); err != nil || m.Size.Width == 0 {
			return nil, errors.New("could not measure the page")
		}
		params["clip"] = map[string]any{"x": 0, "y": 0, "width": m.Size.Width, "height": min(m.Size.Height, 16000), "scale": 1}
		params["captureBeyondViewport"] = true
	}
	raw, err := t.call(ctx, "Page.captureScreenshot", params)
	if err != nil {
		return nil, err
	}
	var shot struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(raw, &shot); err != nil {
		return nil, fmt.Errorf("reading the screenshot: %w", err)
	}
	data, err := base64.StdEncoding.DecodeString(shot.Data)
	if err != nil {
		return nil, fmt.Errorf("decoding the screenshot: %w", err)
	}
	return data, nil
}

// lockedPID is the process the profile's lock names, and whether it still runs on this profile.
func lockedPID(profile string) (pid int, running bool) {
	target, err := os.Readlink(filepath.Join(profile, "SingletonLock"))
	if err != nil {
		return 0, false
	}
	i := strings.LastIndexByte(target, '-')
	pid, err = strconv.Atoi(target[i+1:])
	if i < 0 || err != nil || pid <= 1 {
		return 0, false
	}
	cmdline, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	return pid, err == nil && strings.Contains(string(cmdline), "--user-data-dir="+profile)
}

// stopStrayBrowser ends a browser that an earlier run of the app left on the profile (the app was killed
// without stopping it), and clears its lock.
func stopStrayBrowser(profile string) {
	if pid, running := lockedPID(profile); running {
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		_ = syscall.Kill(pid, syscall.SIGKILL)
		time.Sleep(200 * time.Millisecond)
	}
	for _, f := range []string{"SingletonLock", "SingletonSocket", "SingletonCookie"} {
		_ = os.Remove(filepath.Join(profile, f))
	}
}

var refPattern = regexp.MustCompile(`^e[0-9]+$`)

// eval runs JavaScript in the page and returns its JSON result.
func (t *tab) eval(ctx context.Context, expression string) (json.RawMessage, error) {
	raw, err := t.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true, "awaitPromise": true})
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
		msg := firstLine(res.Exception.Exception.Description)
		if msg == "" {
			msg = res.Exception.Text
		}
		return nil, errors.New(msg)
	}
	return res.Result.Value, nil
}

// call runs fn in the page with one JSON argument.
func (t *tab) apply(ctx context.Context, fn string, arg any) (json.RawMessage, error) {
	a, err := json.Marshal(arg)
	if err != nil {
		return nil, err
	}
	return t.eval(ctx, "("+fn+")("+string(a)+")")
}

type box struct {
	X, Y, Width, Height float64
	PageX, PageY        float64
	Covered             bool
	Error               string
}

const boxScript = `(ref) => {
  const el = document.querySelector('[data-agentos-ref="' + ref + '"]')
  if (!el) return { error: 'unknown' }
  el.scrollIntoView({ block: 'center', inline: 'center' })
  const r = el.getBoundingClientRect()
  if (!(r.width > 0 && r.height > 0)) return { error: 'hidden' }
  const x = r.left + r.width / 2, y = r.top + r.height / 2
  const top = document.elementFromPoint(x, y)
  return { x, y, width: r.width, height: r.height, pageX: r.left + scrollX, pageY: r.top + scrollY,
           covered: !!top && top !== el && !el.contains(top) && !top.contains(el) }
}`

// elementBox scrolls the element with the ref into view and measures it.
func (t *tab) elementBox(ctx context.Context, ref string) (box, error) {
	if !refPattern.MatchString(ref) {
		return box{}, fmt.Errorf("%q is not an element ref like e12", ref)
	}
	raw, err := t.apply(ctx, boxScript, ref)
	if err != nil {
		return box{}, err
	}
	var r struct {
		X, Y, Width, Height, PageX, PageY float64
		Covered                           bool
		Error                             string
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return box{}, fmt.Errorf("reading the element's position: %w", err)
	}
	switch r.Error {
	case "unknown":
		return box{}, fmt.Errorf("no element %s on the page: run snapshot again", ref)
	case "hidden":
		return box{}, fmt.Errorf("%s is not visible", ref)
	}
	return box{X: r.X, Y: r.Y, Width: r.Width, Height: r.Height, PageX: r.PageX, PageY: r.PageY, Covered: r.Covered}, nil
}

func keyOf(id string) string {
	name, err := session.ParseName(id)
	if err != nil {
		return "project"
	}
	return name.Project
}

// Hook sets what the browsers ask of the app: the page a new tab opens first, and a nudge when a tab opens or closes.
func (b *Browsers) Hook(urlFor func(sessionID string) string, onTabs func()) {
	b.urlFor, b.onTabs = urlFor, onTabs
}

// Eval runs JavaScript in the session's page and returns its JSON result.
func (b *Browsers) Eval(ctx context.Context, id, js string) (string, error) {
	t, err := b.need(id)
	if err != nil {
		return "", err
	}
	raw, err := t.eval(ctx, js)
	return string(raw), err
}

// SetGrace sets how long a browser outlives its last tab.
func (b *Browsers) SetGrace(d time.Duration) { b.grace = d }
