// Package devhttp serves the app to a browser instead of a window: the way to check
// the real front end against the real core where no window can be seen.
package devhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"path"
	"reflect"
	"strings"
	"sync"
)

// devShim gives a plain browser the same window.go / window.runtime the Wails
// window provides, so the real front end can run against the real core.
const devShim = `
const listeners = {}
const proxy = (get) => new Proxy({}, { get: (_, name) => get(name) })
window.go = proxy((pkg) => proxy((type) => proxy((method) => async (...args) => {
  const res = await fetch('/__call/' + pkg + '.' + type + '/' + method, { method: 'POST', body: JSON.stringify(args) })
  const text = await res.text()
  if (!res.ok) throw text
  return text ? JSON.parse(text) : undefined
})))
window.runtime = { EventsOn: (name, handler) => {
  (listeners[name] ??= []).push(handler)
  return () => { listeners[name] = listeners[name].filter((h) => h !== handler) }
} }
new EventSource('/__events').onmessage = (e) => {
  const { event, payload } = JSON.parse(e.data)
  for (const handler of listeners[event] ?? []) handler(payload)
}
`

// Hub fans backend events out to every connected browser tab.
type Hub struct {
	mu   sync.Mutex
	subs map[chan []byte]struct{}
}

// Emit sends an event to every tab.
func (h *Hub) Emit(event string, payload any) {
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

func (h *Hub) serve(w http.ResponseWriter, r *http.Request) {
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

// NewHub makes a hub with no tabs.
func NewHub() *Hub { return &Hub{subs: map[chan []byte]struct{}{}} }

// service finds the bound service by "package.Type".
func service(services []any, name string) (any, bool) {
	for _, svc := range services {
		t := reflect.TypeOf(svc).Elem()
		if path.Base(t.PkgPath())+"."+t.Name() == name {
			return svc, true
		}
	}
	return nil, false
}

// call runs one method of a bound service with JSON arguments, as the Wails bridge would.
func call(services []any, svcName, name string, body []byte) (any, error) {
	svc, ok := service(services, svcName)
	if !ok {
		return nil, fmt.Errorf("no service %s", svcName)
	}
	method := reflect.ValueOf(svc).MethodByName(name)
	if !method.IsValid() {
		return nil, fmt.Errorf("no method %s.%s", svcName, name)
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

// Options is what Serve needs.
type Options struct {
	Services []any // the bound services
	Start    func(ctx context.Context) error
	Stop     func()
	Hub      *Hub
	Assets   fs.FS
	Media    http.Handler // serves /media/
	Addr     string
}

// MediaPrefix is where the pictures are served.
const MediaPrefix = "/media/"

// Serve runs the app in a browser.
func Serve(o Options) error {
	if err := o.Start(context.Background()); err != nil {
		return err
	}
	defer o.Stop()

	index, err := fs.ReadFile(o.Assets, "index.html")
	if err != nil {
		return err
	}
	index = bytes.Replace(index, []byte("<head>"), []byte(`<head><script src="/__shim.js"></script>`), 1)
	files := http.FileServerFS(o.Assets)

	mux := http.NewServeMux()
	mux.HandleFunc("/__shim.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript")
		io.WriteString(w, devShim)
	})
	mux.HandleFunc("/__events", o.Hub.serve)
	mux.Handle(MediaPrefix, o.Media)
	mux.HandleFunc("/__call/", func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		svcName, method, _ := strings.Cut(strings.TrimPrefix(r.URL.Path, "/__call/"), "/")
		result, err := call(o.Services, svcName, method, body)
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
	fmt.Println("agentos: serving on http://" + o.Addr)
	return http.ListenAndServe(o.Addr, mux)
}
