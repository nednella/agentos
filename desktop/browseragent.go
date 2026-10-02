package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nednella/agentos/internal/control"
)

//go:embed snapshot.js
var snapshotJS string

const (
	loadLimit    = 30 * time.Second
	waitForLimit = 10 * time.Second
	maxTextOut   = 20000
)

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

func (t *tab) mouse(ctx context.Context, typ string, x, y float64, button string, clicks int) error {
	_, err := t.call(ctx, "Input.dispatchMouseEvent", map[string]any{"type": typ, "x": x, "y": y, "button": button, "clickCount": clicks})
	return err
}

// waitLoaded waits for the page to finish loading after a navigation that began when loadSeq was before.
func (t *tab) waitLoaded(ctx context.Context, before int) error {
	ctx, cancel := context.WithTimeout(ctx, loadLimit)
	defer cancel()
	start := time.Now()
	for {
		t.mu.Lock()
		loaded := t.loadSeq > before && !t.state.Loading
		loading := t.state.Loading
		t.mu.Unlock()
		if loaded {
			return nil
		}
		// A navigation inside the page fires no load event.
		if !loading && time.Since(start) > 500*time.Millisecond {
			if raw, err := t.eval(ctx, "document.readyState"); err == nil && string(raw) == `"complete"` {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return errors.New("the page did not finish loading")
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (t *tab) seq() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.loadSeq
}

// tabFor returns the session's tab, opening one if there is none.
func (b *Browsers) tabFor(ctx context.Context, id string) (*tab, error) {
	if t := b.tab(id); t != nil {
		return t, nil
	}
	if _, err := b.Open(ctx, id, ""); err != nil {
		return nil, err
	}
	return b.need(id)
}

// pageLine is the page's title and address, read from the page itself.
func (t *tab) pageLine(ctx context.Context) string {
	raw, err := t.eval(ctx, `document.title + "\n" + location.href`)
	var both string
	if err != nil || json.Unmarshal(raw, &both) != nil {
		s := t.snapshotState()
		return s.Title + " - " + s.URL
	}
	title, addr, _ := strings.Cut(both, "\n")
	return title + " - " + addr
}

// agent runs one command of the agentos browser CLI on the session's tab.
func (b *Browsers) agent(ctx context.Context, id, cmd string, args []string, opts map[string]string) (string, error) {
	need := func(n int, usage string) error {
		if len(args) < n {
			return fmt.Errorf("usage: agentos browser %s", usage)
		}
		return nil
	}
	if cmd == "help" {
		return control.BrowserHelp, nil
	}
	if cmd == "open" {
		if err := need(1, "open <url>"); err != nil {
			return "", err
		}
	}
	t, err := b.tabFor(ctx, id)
	if err != nil {
		return "", err
	}
	switch cmd {
	case "open":
		before := t.seq()
		if err := b.Goto(ctx, id, args[0]); err != nil {
			return "", err
		}
		if err := t.waitLoaded(ctx, before); err != nil {
			return "", err
		}
		return "opened " + t.pageLine(ctx), nil
	case "back", "reload":
		before := t.seq()
		if err := b.Nav(ctx, id, cmd); err != nil {
			return "", err
		}
		if err := t.waitLoaded(ctx, before); err != nil {
			return "", err
		}
		return t.pageLine(ctx), nil
	case "url":
		raw, err := t.eval(ctx, `location.href + "\n" + document.title`)
		if err != nil {
			return "", err
		}
		var out string
		_ = json.Unmarshal(raw, &out)
		return out, nil
	case "snapshot":
		raw, err := t.eval(ctx, snapshotJS)
		if err != nil {
			return "", err
		}
		var out string
		_ = json.Unmarshal(raw, &out)
		return out, nil
	case "click", "hover":
		if err := need(1, cmd+" <ref>"); err != nil {
			return "", err
		}
		r, err := t.elementBox(ctx, args[0])
		if err != nil {
			return "", err
		}
		if err := t.mouse(ctx, "mouseMoved", r.X, r.Y, "none", 0); err != nil {
			return "", err
		}
		if cmd == "click" {
			if err := t.mouse(ctx, "mousePressed", r.X, r.Y, "left", 1); err != nil {
				return "", err
			}
			if err := t.mouse(ctx, "mouseReleased", r.X, r.Y, "left", 1); err != nil {
				return "", err
			}
		}
		out := map[string]string{"click": "clicked ", "hover": "hovering "}[cmd] + args[0]
		if r.Covered {
			out += " (another element may cover it)"
		}
		return out, nil
	case "type":
		if err := need(2, "type <ref> <text> [--append]"); err != nil {
			return "", err
		}
		return "", t.typeInto(ctx, args[0], strings.Join(args[1:], " "), opts["append"] != "")
	case "press":
		if err := need(1, "press <key>"); err != nil {
			return "", err
		}
		return "", t.press(ctx, args[0])
	case "select":
		if err := need(2, "select <ref> <option text or value>"); err != nil {
			return "", err
		}
		return t.choose(ctx, args[0], strings.Join(args[1:], " "))
	case "scroll":
		if err := need(1, "scroll <up|down|ref> [pixels]"); err != nil {
			return "", err
		}
		return t.scroll(ctx, args)
	case "wait":
		if err := need(1, "wait <ms>"); err != nil {
			return "", err
		}
		ms, err := strconv.Atoi(args[0])
		if err != nil || ms < 0 {
			return "", fmt.Errorf("%q is not a number of milliseconds", args[0])
		}
		select {
		case <-time.After(time.Duration(min(ms, 60000)) * time.Millisecond):
		case <-ctx.Done():
		}
		return "", nil
	case "wait-for":
		if err := need(1, "wait-for <text> [--timeout <ms>]"); err != nil {
			return "", err
		}
		return t.waitFor(ctx, strings.Join(args, " "), opts["timeout"])
	case "text":
		expr := "document.body ? document.body.innerText : ''"
		if len(args) > 0 {
			if !refPattern.MatchString(args[0]) {
				return "", fmt.Errorf("%q is not an element ref like e12", args[0])
			}
			expr = `(() => { const el = document.querySelector('[data-agentos-ref="` + args[0] + `"]'); return el ? el.innerText : null })()`
		}
		raw, err := t.eval(ctx, expr)
		if err != nil {
			return "", err
		}
		if string(raw) == "null" {
			return "", fmt.Errorf("no element %s on the page: run snapshot again", args[0])
		}
		var out string
		_ = json.Unmarshal(raw, &out)
		if r := []rune(out); len(r) > maxTextOut {
			out = string(r[:maxTextOut]) + "\n... cut at 20000 characters"
		}
		return out, nil
	case "eval":
		if err := need(1, "eval <js>"); err != nil {
			return "", err
		}
		raw, err := t.eval(ctx, strings.Join(args, " "))
		if err != nil {
			return "", err
		}
		if len(raw) == 0 {
			return "undefined", nil
		}
		return string(raw), nil
	case "console":
		lines := t.drainConsole()
		if len(lines) == 0 {
			return "no console errors or failed requests", nil
		}
		return strings.Join(lines, "\n"), nil
	}
	return "", fmt.Errorf("unknown browser command %q: run agentos browser help", cmd)
}

func (t *tab) typeInto(ctx context.Context, ref, text string, appendText bool) error {
	if !refPattern.MatchString(ref) {
		return fmt.Errorf("%q is not an element ref like e12", ref)
	}
	raw, err := t.apply(ctx, `(a) => {
  const el = document.querySelector('[data-agentos-ref="' + a.ref + '"]')
  if (!el) return 'unknown'
  el.scrollIntoView({ block: 'center' })
  el.focus()
  if (a.append) {
    if (el.setSelectionRange && el.value !== undefined) { try { el.setSelectionRange(el.value.length, el.value.length) } catch (e) {} }
    else { const s = getSelection(); s.selectAllChildren(el); s.collapseToEnd() }
  } else if (el.select) { el.select() } else { document.execCommand('selectAll') }
  return 'ok'
}`, map[string]any{"ref": ref, "append": appendText})
	if err != nil {
		return err
	}
	if string(raw) == `"unknown"` {
		return fmt.Errorf("no element %s on the page: run snapshot again", ref)
	}
	if !appendText {
		if err := t.press(ctx, "Backspace"); err != nil {
			return err
		}
	}
	if text == "" {
		return nil
	}
	_, err = t.call(ctx, "Input.insertText", map[string]any{"text": text})
	return err
}

// press sends a key or chord such as Enter or Control+a.
func (t *tab) press(ctx context.Context, spec string) error {
	parts := strings.Split(spec, "+")
	key := parts[len(parts)-1]
	if spec == "+" || strings.HasSuffix(spec, "++") {
		key = "+"
	}
	mods := 0
	for _, m := range parts[:len(parts)-1] {
		switch strings.ToLower(m) {
		case "alt", "option":
			mods |= modAlt
		case "control", "ctrl":
			mods |= modCtrl
		case "meta", "cmd", "command":
			mods |= modMeta
		case "shift":
			mods |= modShift
		}
	}
	if key == "Space" {
		key = " "
	}
	code := key
	if r := []rune(key); len(r) == 1 {
		switch {
		case r[0] >= 'a' && r[0] <= 'z', r[0] >= 'A' && r[0] <= 'Z':
			code = "Key" + strings.ToUpper(key)
		case r[0] >= '0' && r[0] <= '9':
			code = "Digit" + key
		}
	}
	text := ""
	if len([]rune(key)) == 1 {
		text = key
	}
	for _, action := range []string{"down", "up"} {
		if err := t.key(ctx, BrowserInput{Type: "key", Action: action, Key: key, Code: code, Text: text, Modifiers: mods}); err != nil {
			return err
		}
	}
	return nil
}

func (t *tab) choose(ctx context.Context, ref, option string) (string, error) {
	if !refPattern.MatchString(ref) {
		return "", fmt.Errorf("%q is not an element ref like e12", ref)
	}
	raw, err := t.apply(ctx, `(a) => {
  const el = document.querySelector('[data-agentos-ref="' + a.ref + '"]')
  if (!el) return { error: 'unknown' }
  if (el.tagName !== 'SELECT') return { error: 'not a select' }
  const want = a.option.trim().toLowerCase()
  const opts = Array.from(el.options)
  const hit = opts.find((o) => o.text.trim().toLowerCase() === want) || opts.find((o) => o.value.toLowerCase() === want)
  if (!hit) return { error: 'no option', options: opts.map((o) => o.text.trim()) }
  el.value = hit.value
  el.dispatchEvent(new Event('input', { bubbles: true }))
  el.dispatchEvent(new Event('change', { bubbles: true }))
  return { chosen: hit.text.trim() }
}`, map[string]string{"ref": ref, "option": option})
	if err != nil {
		return "", err
	}
	var r struct {
		Error   string
		Chosen  string
		Options []string
	}
	_ = json.Unmarshal(raw, &r)
	switch r.Error {
	case "":
		return "selected " + r.Chosen, nil
	case "unknown":
		return "", fmt.Errorf("no element %s on the page: run snapshot again", ref)
	case "not a select":
		return "", fmt.Errorf("%s is not a select", ref)
	}
	return "", fmt.Errorf("no option %q in %s; options: %s", option, ref, strings.Join(r.Options, " | "))
}

func (t *tab) scroll(ctx context.Context, args []string) (string, error) {
	dir := args[0]
	if dir == "up" || dir == "down" {
		t.mu.Lock()
		w, h := t.width, t.height
		t.mu.Unlock()
		pixels := float64(h) * 0.8
		if len(args) > 1 {
			n, err := strconv.Atoi(args[1])
			if err != nil {
				return "", fmt.Errorf("%q is not a number of pixels", args[1])
			}
			pixels = float64(n)
		}
		if dir == "up" {
			pixels = -pixels
		}
		if _, err := t.call(ctx, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": w / 2, "y": h / 2, "deltaX": 0, "deltaY": pixels}); err != nil {
			return "", err
		}
		time.Sleep(150 * time.Millisecond)
	} else if _, err := t.elementBox(ctx, dir); err != nil {
		return "", err
	}
	raw, err := t.eval(ctx, `'scrolled to ' + Math.round(scrollY) + ' of ' + Math.max(0, Math.round(document.documentElement.scrollHeight - innerHeight)) + ' px'`)
	if err != nil {
		return "", err
	}
	var out string
	_ = json.Unmarshal(raw, &out)
	return out, nil
}

func (t *tab) waitFor(ctx context.Context, text, timeoutMs string) (string, error) {
	limit := waitForLimit
	if timeoutMs != "" {
		ms, err := strconv.Atoi(timeoutMs)
		if err != nil || ms < 0 {
			return "", fmt.Errorf("%q is not a number of milliseconds", timeoutMs)
		}
		limit = time.Duration(ms) * time.Millisecond
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	for {
		raw, err := t.apply(ctx, `(text) => !!document.body && document.body.innerText.includes(text)`, text)
		if err == nil && string(raw) == "true" {
			return "found " + strconv.Quote(text), nil
		}
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("%q did not appear within %s", text, limit.Round(time.Millisecond))
		case <-time.After(200 * time.Millisecond):
		}
	}
}
