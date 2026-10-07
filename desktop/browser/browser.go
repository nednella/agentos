// Package browser runs one browser per project and one window per session.
package browser

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	consoleShown      = 20
	consolePublishGap = 500 * time.Millisecond
	browserStartLimit = 15 * time.Second
	callLimit         = 15 * time.Second
	stopLimit         = 5 * time.Second
	// dialogWait is how long a JavaScript dialog may stay open before agentos dismisses it: a dialog blocks every
	// call to its page, including the ones that would answer it.
	dialogWait  = 5 * time.Second
	dialogShown = 80
)

// BrowserState is what the front end shows of a session's browser: the state of its active page.
type BrowserState struct {
	ID       string `json:"id"`
	Open     bool   `json:"open"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Loading  bool   `json:"loading"`
	Error    string `json:"error"`
	LoadedAt int64  `json:"loadedAt"` // unix ms the main frame last finished loading, 0 if it never has
	Pages    int    `json:"pages"`    // how many pages the session has open; the rest of the state is of the active one
	// Console is the page's last few console errors, failed requests and dismissed dialogs, oldest first.
	Console []string `json:"console"`
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

// Browsers runs one browser per project and one window per session. AGENTOS_BROWSER_HEADLESS=1 starts the
// browser without windows, for tests.
type Browsers struct {
	dataDir    string
	binary     string
	headless   bool
	emit       func(event string, payload any)
	dialogWait time.Duration
	urlFor     func(sessionID string) string // the page a new tab opens first
	labelFor   func(sessionID string) string // names the session in its window's title
	onTabs     func()                        // a tab opened or closed
	launchMu   sync.Mutex
	portMu     sync.Mutex

	mu      sync.Mutex
	procs   map[string]*browserProc // by project key
	tabs    map[string]*tab         // by session id
	opening map[string]*sync.Mutex  // by session id: one Open at a time per session
}

// New runs the browsers with their profiles under dataDir.
func New(dataDir string, emit func(string, any)) *Browsers {
	return &Browsers{
		dataDir: dataDir, binary: findBrowser(), headless: os.Getenv("AGENTOS_BROWSER_HEADLESS") == "1", emit: emit, dialogWait: dialogWait,
		urlFor: func(string) string { return "" }, labelFor: func(string) string { return "" }, onTabs: func() {},
		procs: map[string]*browserProc{}, tabs: map[string]*tab{}, opening: map[string]*sync.Mutex{},
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

	tabs int
}

// tab is one session's browser: the pages it owns, and the state the front end shows.
type tab struct {
	b      *Browsers
	proc   *browserProc
	id     string
	prefix string // what the title script puts before a page's title

	mu        sync.Mutex
	state     BrowserState
	pages     []*page     // oldest first
	recent    []string    // the last consoleShown lines, from every page
	recentPub *time.Timer // set while a coalesced publish of recent is pending
}

func (t *tab) snapshotState() BrowserState {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.state
	st.Console = slices.Clone(t.recent)
	if st.Console == nil {
		st.Console = []string{}
	}
	return st
}

// publish sends the tab's state to the front end.
func (t *tab) publish() { t.b.emit("browser:state", t.snapshotState()) }

func (t *tab) logConsole(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.recent = append(t.recent, line)
	if len(t.recent) > consoleShown {
		t.recent = t.recent[len(t.recent)-consoleShown:]
	}
	// A noisy page would otherwise publish a state event per line.
	if t.recentPub == nil {
		t.recentPub = time.AfterFunc(consolePublishGap, func() {
			t.mu.Lock()
			t.recentPub = nil
			t.mu.Unlock()
			t.publish()
		})
	}
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
	stopStrayBrowser(profile)
	// A browser that ended with pages open restores them at its next start, into the first session's window, which
	// then adopts them as its own.
	if err := os.RemoveAll(filepath.Join(profile, "Default", "Sessions")); err != nil {
		return nil, fmt.Errorf("clearing the browser's last session: %w", err)
	}
	port, err := b.launchPort(key)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(b.binary, b.launchArgs(profile, port)...)
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

	wsURL, err := waitForEndpoint(ctx, port, p.gone)
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
	// A browser that drops the connection cannot be driven any more, though it may still run: forget it, and the next
	// start ends it as a stray.
	go func() {
		<-c.done
		b.exited(p)
	}()
	if _, err := c.call(ctx, "", "Target.setDiscoverTargets", map[string]any{"discover": true}); err != nil {
		p.kill()
		return nil, err
	}
	b.mu.Lock()
	b.procs[key] = p
	b.mu.Unlock()
	return p, nil
}

func (b *Browsers) launchArgs(profile string, port int) []string {
	args := []string{"--remote-debugging-port=" + strconv.Itoa(port), "--user-data-dir=" + profile, "--no-first-run", "--no-default-browser-check"}
	if b.headless {
		return append(args, "--headless=new", "--hide-scrollbars", "about:blank")
	}
	// Without these a window hidden behind others stops painting and running timers, and agent screenshots hang.
	return append(args, "--disable-backgrounding-occluded-windows", "--disable-renderer-backgrounding",
		"--disable-background-timer-throttling", "--no-startup-window", "--hide-crash-restore-bubble")
}

// Port is the project's remote debugging port: the one saved for it, else a free one it saves now, 0 when that fails.
// It starts no browser, so a session can name the port in its agent's tool config before the first page opens.
func (b *Browsers) Port(key string) int {
	b.portMu.Lock()
	defer b.portMu.Unlock()
	if port := b.savedPort(key); port != 0 {
		return port
	}
	port, err := freePort()
	if err != nil || b.savePort(key, port) != nil {
		return 0
	}
	return port
}

// launchPort is the port the project's browser listens on at this start: the saved one, unless something else holds
// it now, then a new one that is saved instead.
func (b *Browsers) launchPort(key string) (int, error) {
	b.portMu.Lock()
	defer b.portMu.Unlock()
	if port := b.savedPort(key); port != 0 && portFree(port) {
		return port, nil
	}
	port, err := freePort()
	if err != nil {
		return 0, fmt.Errorf("finding a port for the browser: %w", err)
	}
	if err := b.savePort(key, port); err != nil {
		return 0, err
	}
	return port, nil
}

func (b *Browsers) portFile(key string) string { return filepath.Join(b.dataDir, key, "browser-port") }

func (b *Browsers) savedPort(key string) int {
	data, err := os.ReadFile(b.portFile(key))
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || port < portMin || port > portMax {
		return 0
	}
	return port
}

func (b *Browsers) savePort(key string, port int) error {
	if err := os.MkdirAll(filepath.Dir(b.portFile(key)), 0o700); err != nil {
		return fmt.Errorf("saving the browser port: %w", err)
	}
	return os.WriteFile(b.portFile(key), []byte(strconv.Itoa(port)+"\n"), 0o600)
}

// The ports sit below the range the system gives to outgoing connections, so one of them stays free between runs.
const (
	portMin = 20000
	portMax = 29999
)

func portFree(port int) bool {
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}

func freePort() (int, error) {
	for range 100 {
		if port := portMin + rand.IntN(portMax-portMin+1); portFree(port) {
			return port, nil
		}
	}
	return 0, errors.New("no free port")
}

// waitForEndpoint polls the browser on port until it names the websocket to its protocol.
func waitForEndpoint(ctx context.Context, port int, gone <-chan struct{}) (string, error) {
	deadline := time.After(browserStartLimit)
	url := "http://" + net.JoinHostPort("127.0.0.1", strconv.Itoa(port)) + "/json/version"
	client := &http.Client{Timeout: time.Second}
	for {
		if resp, err := client.Get(url); err == nil {
			var info struct {
				WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
			}
			err = json.NewDecoder(resp.Body).Decode(&info)
			resp.Body.Close()
			if err == nil && info.WebSocketDebuggerURL != "" {
				return info.WebSocketDebuggerURL, nil
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

// stop asks the browser to quit, so it writes its profile out, and kills it only if it has not gone within stopLimit.
func (p *browserProc) stop() {
	if p.cdp != nil {
		ctx, cancel := context.WithTimeout(context.Background(), stopLimit)
		defer cancel()
		_, _ = p.cdp.call(ctx, "", "Browser.close", nil)
		select {
		case <-p.gone:
		case <-ctx.Done():
		}
	}
	p.kill()
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
		for _, pg := range t.pages {
			pg.stopDialog()
		}
		t.mu.Unlock()
		t.publish()
	}
	if len(lost) > 0 {
		b.onTabs()
	}
}

// openLock is the lock that keeps two Opens of one session from each making a tab.
func (b *Browsers) openLock(id string) *sync.Mutex {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.opening[id] == nil {
		b.opening[id] = &sync.Mutex{}
	}
	return b.opening[id]
}

// Open gives the session a browser with one page, starting the project's browser if needed, and goes to url when it
// is not empty. When the first page cannot be opened the page stays, and Open returns the state with the error.
func (b *Browsers) Open(ctx context.Context, id, url string) (BrowserState, error) {
	lock := b.openLock(id)
	lock.Lock()
	defer lock.Unlock()
	if t := b.tab(id); t != nil {
		if url != "" {
			if err := b.Goto(ctx, id, url); err != nil {
				return BrowserState{}, err
			}
		}
		return t.snapshotState(), nil
	}
	key := session.ProjectKey(id)
	p, err := b.start(ctx, key)
	if err != nil {
		return BrowserState{ID: id, Error: err.Error()}, err
	}
	target := map[string]any{"url": "about:blank", "newWindow": true, "background": true, "width": defaultViewWidth, "height": defaultViewHeight}
	raw, err := p.cdp.call(ctx, "", "Target.createTarget", target)
	if err != nil {
		return BrowserState{}, err
	}
	var created struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &created)
	t := &tab{b: b, proc: p, id: id, prefix: windowLabel(b.labelFor(id)),
		state: BrowserState{ID: id, Open: true, URL: "about:blank", Pages: 1}}
	first := t.newPage(created.TargetID)
	t.pages = []*page{first}
	b.mu.Lock()
	b.tabs[id] = t
	p.tabs++
	b.mu.Unlock()
	b.onTabs()

	if err := t.attach(ctx, first); err != nil {
		b.Close(ctx, id)
		return BrowserState{}, err
	}
	go t.watch()
	if url == "" {
		url = b.urlFor(id)
	}
	if url != "" {
		if err := b.Goto(ctx, id, url); err != nil {
			t.publish()
			return t.snapshotState(), err
		}
	}
	t.publish()
	return t.snapshotState(), nil
}

// windowLabel is the bracketed label before a window's title. Whitespace is collapsed the way document.title
// collapses it, so the label read back from the page still matches and pageTitle can strip it.
func windowLabel(label string) string {
	return "[" + strings.Join(strings.Fields(label), " ") + "]"
}

// titleScript keeps prefix before the page's document title, and is the whole title when the page has none.
func titleScript(prefix string) string {
	quoted, _ := json.Marshal(prefix)
	return `(() => {
  if (window !== top) return
  const prefix = ` + string(quoted) + `
  const watch = () => observer.observe(document, { subtree: true, childList: true, characterData: true })
  const apply = () => {
    const own = document.title === prefix ? '' : document.title.startsWith(prefix + ' ') ? document.title.slice(prefix.length + 1) : document.title
    const want = own ? prefix + ' ' + own : prefix
    if (document.title === want) return
    // Unwatched while writing, so a title that reads back different from what was set cannot retrigger apply forever.
    observer.disconnect()
    document.title = want
    watch()
  }
  const observer = new MutationObserver(apply)
  watch()
  apply()
})()`
}

// pageTitle is the title the page set, without the label the window shows before it.
func (t *tab) pageTitle(title string) string {
	if t.prefix == "" {
		return title
	}
	if title == t.prefix {
		return ""
	}
	return strings.TrimPrefix(title, t.prefix+" ")
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
	return BrowserState{ID: id, Console: []string{}}
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

// Goto starts loading an address in the session's active page.
func (b *Browsers) Goto(ctx context.Context, id, address string) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	pg, err := t.activePage()
	if err != nil {
		return err
	}
	return pg.goTo(ctx, address)
}

// Show brings the window of the session's active page to the front.
func (b *Browsers) Show(ctx context.Context, id string) error {
	t, err := b.need(id)
	if err != nil {
		return err
	}
	pg, err := t.refreshActive(ctx)
	if err != nil {
		return err
	}
	return pg.activate(ctx)
}

// Close closes all the session's pages, and stops the project's browser when no other session has a window in it.
func (b *Browsers) Close(ctx context.Context, id string) {
	b.mu.Lock()
	t := b.tabs[id]
	delete(b.tabs, id)
	b.mu.Unlock()
	if t == nil {
		return
	}
	t.mu.Lock()
	pages := slices.Clone(t.pages)
	t.mu.Unlock()
	for _, pg := range pages {
		_, _ = t.proc.cdp.call(ctx, "", "Target.closeTarget", map[string]any{"targetId": pg.targetID})
	}
	t.mu.Lock()
	t.state.Open = false
	for _, pg := range pages {
		pg.stopDialog()
	}
	t.mu.Unlock()
	t.publish()
	b.onTabs()

	p := t.proc
	b.mu.Lock()
	p.tabs--
	// Forgotten before it stops, so an Open meanwhile starts a new browser rather than reuse this one.
	idle := p.tabs <= 0 && b.procs[p.key] == p
	if idle {
		delete(b.procs, p.key)
	}
	b.mu.Unlock()
	if idle {
		p.stop()
	}
}

// CloseAll stops every browser: the app is quitting.
func (b *Browsers) CloseAll() {
	b.mu.Lock()
	procs := slices.Collect(maps.Values(b.procs))
	b.mu.Unlock()
	for _, p := range procs {
		p.stop()
		<-p.gone
	}
}

// Screenshot returns a PNG of the viewport or the whole page of the session's active page.
func (b *Browsers) Screenshot(ctx context.Context, id string, full bool) ([]byte, error) {
	t, err := b.need(id)
	if err != nil {
		return nil, err
	}
	pg, err := t.refreshActive(ctx)
	if err != nil {
		return nil, err
	}
	params := map[string]any{"format": "png"}
	if full {
		raw, err := pg.call(ctx, "Page.getLayoutMetrics", nil)
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
	raw, err := pg.call(ctx, "Page.captureScreenshot", params)
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

// Hook sets what the browsers ask of the app: the page a new tab opens first, the label that names a session in
// its window's title, and a nudge when a tab opens or closes.
func (b *Browsers) Hook(urlFor, labelFor func(sessionID string) string, onTabs func()) {
	b.urlFor, b.labelFor, b.onTabs = urlFor, labelFor, onTabs
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

// SetDialogWait sets how long a JavaScript dialog may stay open before it is dismissed.
func (b *Browsers) SetDialogWait(d time.Duration) { b.dialogWait = d }
