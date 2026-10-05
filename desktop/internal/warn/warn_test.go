package warn

import (
	"reflect"
	"testing"
)

func TestWarnings(t *testing.T) {
	steps := []struct {
		name string
		do   func(w *Warnings)
		want []Warning
	}{
		{"first failure", func(w *Warnings) { w.Report("tmux", "down") }, []Warning{{"tmux", "down"}}},
		{"same message again", func(w *Warnings) { w.Report("tmux", "down") }, nil},
		{"another message", func(w *Warnings) { w.Report("tmux", "worse") }, []Warning{{"tmux", "worse"}}},
		{"another source", func(w *Warnings) { w.Report("gh", "worse") }, []Warning{{"gh", "worse"}}},
		{"recovery", func(w *Warnings) { w.Clear("tmux") }, []Warning{{Source: "tmux"}}},
		{"recovery twice", func(w *Warnings) { w.Clear("tmux") }, nil},
		{"the old message after recovery", func(w *Warnings) { w.Report("tmux", "worse") }, []Warning{{"tmux", "worse"}}},
		{"clear of a source that never failed", func(w *Warnings) { w.Clear("issues") }, nil},
	}
	var got []Warning
	w := New(func(event string, payload any) {
		if event != "warnings" {
			t.Fatalf("event %q", event)
		}
		got = append(got, payload.(Warning))
	})
	for _, s := range steps {
		got = nil
		s.do(w)
		if !reflect.DeepEqual(got, s.want) {
			t.Errorf("%s: emitted %v, want %v", s.name, got, s.want)
		}
	}
}
