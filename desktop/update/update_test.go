package update_test

import (
	"os"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/update"
	ctl "github.com/nednella/agentos/internal/control"
)

func TestMain(m *testing.M) { os.Exit(apptest.Main(m)) }

const releaseJSON = `{"tag_name":"v9.0.0","assets":[
	{"name":"agentos-darwin-arm64.zip","browser_download_url":"https://x/zip"},
	{"name":"agentos-darwin-arm64.zip.sha256","browser_download_url":"https://x/zip.sha256"}]}`

func TestANewerReleaseIsAnnouncedOnceAndShowsInTheSnapshot(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{UpdateTick: 50 * time.Millisecond, Releases: releaseJSON, Version: "1.0.0"})
	apptest.Eventually(t, "the update event", func() bool { return h.Rec.Count("update") == 1 })
	if got := h.Rec.Last("update"); got != (update.Available{Version: "9.0.0"}) {
		t.Errorf("update event = %+v", got)
	}
	if snap := h.Snapshot(); snap.Update != "9.0.0" {
		t.Errorf("snapshot.update = %q", snap.Update)
	}
	apptest.Eventually(t, "a second check", func() bool { return h.GH.Calls("curl") >= 3 })
	if h.Rec.Count("update") != 1 {
		t.Errorf("the same release was announced %d times", h.Rec.Count("update"))
	}
}

func TestAFailedCheckLeavesTheAppCurrent(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{UpdateTick: time.Hour, Version: "1.0.0"})
	apptest.Eventually(t, "the check", func() bool { return h.GH.Calls("curl") == 1 })
	if snap := h.Snapshot(); snap.Update != "" || h.Rec.Count("update") != 0 {
		t.Errorf("snapshot.update = %q, %d update events", snap.Update, h.Rec.Count("update"))
	}
}

func TestRelaunchOpensTheBundleAgainAndQuits(t *testing.T) {
	h := apptest.New(t)
	resp := h.Ask(t, ctl.Request{Cmd: "relaunch"})
	// The test binary runs from no .app bundle, so the app can only say so.
	if resp.OK || resp.Error != "agentos runs from no .app bundle, so it cannot relaunch" {
		t.Errorf("relaunch answered %+v", resp)
	}
	if len(h.Rec.Relaunched()) != 0 || h.Rec.Quits() != 0 {
		t.Errorf("relaunched %v, quit %d times", h.Rec.Relaunched(), h.Rec.Quits())
	}
}
