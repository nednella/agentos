// Package warn tells the front end about failures a service would otherwise swallow.
package warn

import "sync"

// Warning is the payload of the warnings event. An empty Message means the source works again.
type Warning struct {
	Source  string `json:"source"`
	Message string `json:"message"`
}

// Warnings emits a warning once per distinct message of each source, and again when the source recovers.
type Warnings struct {
	emit func(event string, payload any)
	mu   sync.Mutex
	last map[string]string
}

func New(emit func(event string, payload any)) *Warnings {
	return &Warnings{emit: emit, last: map[string]string{}}
}

// Report emits the warning unless the source already reported this message.
func (w *Warnings) Report(source, message string) {
	w.mu.Lock()
	same := w.last[source] == message
	w.last[source] = message
	w.mu.Unlock()
	if !same {
		w.emit("warnings", Warning{Source: source, Message: message})
	}
}

// Clear tells the front end the source works again, if it had reported a failure.
func (w *Warnings) Clear(source string) {
	w.mu.Lock()
	_, failed := w.last[source]
	delete(w.last, source)
	w.mu.Unlock()
	if failed {
		w.emit("warnings", Warning{Source: source})
	}
}
