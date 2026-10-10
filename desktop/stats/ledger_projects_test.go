package stats_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
)

// A config shared between two Macs lists projects that only one of them has. The data folder can be
// shared too, so this Mac holds the other's records; its ledger leaves them out.
func TestLedgerLeavesOutProjectsNotOnThisMachine(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "  - name: elsewhere\n    directory: /nonexistent/elsewhere\n"})
	now := time.Now()
	for _, key := range []string{"main", "elsewhere"} {
		dir := filepath.Join(h.State, "data", key, "events")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
		line := fmt.Sprintf(`{"at":%d,"kind":"prompt"}`+"\n", now.UnixMilli())
		if err := os.WriteFile(filepath.Join(dir, now.Format(time.DateOnly)+".jsonl"), []byte(line), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	led, err := h.Restart(t).Ledger(30)
	if err != nil {
		t.Fatal(err)
	}
	if len(led.Projects) != 1 || led.Projects[0].Project != "main" || led.Totals.Prompts != 1 || led.Heat[len(led.Heat)-1].Count != 1 {
		t.Errorf("projects = %+v, totals = %+v: only main is on this machine", led.Projects, led.Totals)
	}
}
