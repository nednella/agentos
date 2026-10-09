package sessions_test

import (
	"testing"
	"time"
)

func TestActivityIsCounted(t *testing.T) {
	h := newHarness(t)
	s, err := h.Sessions().Create("#4 work", "", false, 4)
	if err != nil {
		t.Fatal(err)
	}
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"a"}`)
	time.Sleep(50 * time.Millisecond)
	h.Hook(t, s.ID, "Stop", `{}`)
	h.Hook(t, s.ID, "UserPromptSubmit", `{"prompt":"b"}`)
	time.Sleep(50 * time.Millisecond)
	h.Hook(t, s.ID, "Stop", `{}`)

	eventually(t, "two prompts and the work between them", func() bool {
		led, err := h.Ledger(30)
		return err == nil && led.Totals.Prompts == 2 && led.Totals.WorkMs >= 100
	})
	led, err := h.Ledger(30)
	if err != nil {
		t.Fatal(err)
	}
	if led.Totals.Sessions != 1 || led.Totals.IssueSessions != 1 || led.Projects[0].Project != "main" || led.Heat[363].Count != 2 {
		t.Errorf("totals = %+v, heat today = %+v", led.Totals, led.Heat[363])
	}

	second := h.Restart(t)
	time.Sleep(200 * time.Millisecond)
	again, err := second.Ledger(30)
	if err != nil {
		t.Fatal(err)
	}
	if again.Totals.Prompts != 2 || again.Totals.Sessions != 1 {
		t.Errorf("after a restart, totals = %+v: a replayed record or a known session was counted", again.Totals)
	}

	if err := second.Sessions().Kill(s.ID); err != nil {
		t.Fatal(err)
	}
	eventually(t, "the session's length", func() bool {
		led, err := second.Ledger(30)
		return err == nil && led.Totals.AvgSessionMs >= 100
	})
}
