package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

// devShim gives a plain browser the same window.go / window.runtime the Wails
// window provides, so the real front end can run against the real core.
const devShim = `
const listeners = {}
window.go = { main: { App: new Proxy({}, { get: (_, method) => async (...args) => {
  const res = await fetch('/__call/' + method, { method: 'POST', body: JSON.stringify(args) })
  const text = await res.text()
  if (!res.ok) throw text
  return text ? JSON.parse(text) : undefined
} }) } }
window.runtime = { EventsOn: (name, handler) => {
  (listeners[name] ??= []).push(handler)
  return () => { listeners[name] = listeners[name].filter((h) => h !== handler) }
} }
new EventSource('/__events').onmessage = (e) => {
  const { event, payload } = JSON.parse(e.data)
  for (const handler of listeners[event] ?? []) handler(payload)
}
`

// eventHub fans backend events out to every connected browser tab.
type eventHub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

func (h *eventHub) emit(event string, payload any) {
	line, err := json.Marshal(map[string]any{"event": event, "payload": payload})
	if err != nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for sub := range h.subs {
		select {
		case sub <- line:
		default: // a stalled tab must not block the terminal stream
		}
	}
}

func (h *eventHub) serve(w http.ResponseWriter, r *http.Request) {
	sub := make(chan []byte, 4096)
	h.mu.Lock()
	h.subs[sub] = struct{}{}
	h.mu.Unlock()
	defer func() {
		h.mu.Lock()
		delete(h.subs, sub)
		h.mu.Unlock()
	}()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	flusher := w.(http.Flusher)
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case line := <-sub:
			fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

// call runs one bound App method with JSON arguments, as the Wails bridge would.
func call(app *App, name string, body []byte) (any, error) {
	method := reflect.ValueOf(app).MethodByName(name)
	if !method.IsValid() {
		return nil, fmt.Errorf("no method %s", name)
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	if len(raw) != method.Type().NumIn() {
		return nil, fmt.Errorf("%s takes %d arguments, got %d", name, method.Type().NumIn(), len(raw))
	}
	args := make([]reflect.Value, len(raw))
	for i := range raw {
		arg := reflect.New(method.Type().In(i))
		if err := json.Unmarshal(raw[i], arg.Interface()); err != nil {
			return nil, err
		}
		args[i] = arg.Elem()
	}
	var result any
	for _, out := range method.Call(args) {
		if err, ok := out.Interface().(error); ok {
			return nil, err
		}
		if out.Type().Implements(reflect.TypeFor[error]()) {
			continue
		}
		result = out.Interface()
	}
	return result, nil
}

// serveHTTP runs the app in a browser instead of a window: the way to check the
// real front end against the real core where no window can be seen.
func serveHTTP(app *App, hub *eventHub, assets fs.FS, addr string) error {
	if err := app.start(context.Background()); err != nil {
		return err
	}
	defer app.stop()

	index, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return err
	}
	index = bytes.Replace(index, []byte("<head>"), []byte(`<head><script src="/__shim.js"></script>`), 1)
	files := http.FileServerFS(assets)

	mux := http.NewServeMux()
	mux.HandleFunc("/__shim.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		io.WriteString(w, devShim)
	})
	mux.HandleFunc("/__events", hub.serve)
	mux.HandleFunc("/__call/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result, err := call(app, strings.TrimPrefix(r.URL.Path, "/__call/"), body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if result != nil {
			json.NewEncoder(w).Encode(result)
		}
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			w.Header().Set("Content-Type", "text/html")
			w.Write(index)
			return
		}
		files.ServeHTTP(w, r)
	})
	fmt.Println("agentos: serving on http://" + addr)
	return http.ListenAndServe(addr, mux)
}
