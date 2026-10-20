package sessions_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/bus"
	"github.com/nednella/agentos/internal/session"
)

func TestTimelineAndWorkOutliveTheApp(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("#5 work", "", false, 5, "")
	if err != nil {
		t.Fatal(err)
	}
	// The hooks of a session that worked twice while the app was closed.
	t0 := time.Now().Add(-10 * time.Second).UnixMilli()
	changes := []session.Change{{State: session.Idle, At: t0}, {State: session.Working, At: t0 + 1000}, {State: session.Waiting, At: t0 + 3000}, {State: session.Working, At: t0 + 4000}, {State: session.Idle, At: t0 + 6000}}
	rec := session.Record{Session: s.ID, State: session.Idle, Event: "Stop", At: time.UnixMilli(t0 + 6000), Changes: changes}
	if err := bus.WriteState(h.State, rec); err != nil {
		t.Fatal(err)
	}

	second := h.Restart(t)
	row, _ := second.Session(s.ID)
	if !reflect.DeepEqual(row.History, changes) {
		t.Errorf("history = %+v, want the one the hooks kept: %+v", row.History, changes)
	}
	eventually(t, "the work done while the app was closed", func() bool {
		led, err := second.Ledger(30, "")
		return err == nil && led.Totals.WorkMs == 4000
	})

	third := second.Restart(t)
	time.Sleep(300 * time.Millisecond)
	led, err := third.Ledger(30, "")
	if err != nil {
		t.Fatal(err)
	}
	if led.Totals.WorkMs != 4000 {
		t.Errorf("worked %dms after a second restart, want 4000: a stretch was counted twice", led.Totals.WorkMs)
	}
}
