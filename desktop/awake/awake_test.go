package awake

import (
	"errors"
	"testing"
)

type recorder struct {
	starts, stops int
	events        []bool
	fail          bool
}

func (r *recorder) start() (func(), error) {
	if r.fail {
		return nil, errors.New("no caffeinate")
	}
	r.starts++
	return func() { r.stops++ }, nil
}

func (r *recorder) emit(event string, payload any) {
	if event == "awake" {
		r.events = append(r.events, payload.(bool))
	}
}

func TestHoldTakesAndReleasesOnce(t *testing.T) {
	r := &recorder{}
	a := New(r.start, r.emit)
	a.Hold(true)
	a.Hold(true)
	if !a.Held() || r.starts != 1 {
		t.Fatalf("held = %v after %d starts", a.Held(), r.starts)
	}
	a.Hold(false)
	a.Hold(false)
	if a.Held() || r.stops != 1 {
		t.Fatalf("held = %v after %d stops", a.Held(), r.stops)
	}
	if len(r.events) != 2 || !r.events[0] || r.events[1] {
		t.Errorf("events = %v, want [true false]", r.events)
	}
}

func TestHoldWithoutCaffeinateStaysReleased(t *testing.T) {
	r := &recorder{fail: true}
	a := New(r.start, r.emit)
	a.Hold(true)
	if a.Held() || len(r.events) != 0 {
		t.Errorf("held = %v, events = %v", a.Held(), r.events)
	}
}
