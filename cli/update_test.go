package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/nednella/agentos/internal/update"
)

const releaseJSON = `{"tag_name":"v9.0.0","assets":[
	{"name":"agentos-darwin-arm64.zip","browser_download_url":"https://x/zip"},
	{"name":"agentos-darwin-arm64.zip.sha256","browser_download_url":"https://x/sum"}]}`

func releasesRunner(calls *int) update.Runner {
	return func(_ context.Context, name string, args ...string) ([]byte, error) {
		*calls++
		if name != "curl" || !strings.HasSuffix(args[len(args)-1], "/releases/latest") {
			return nil, fmt.Errorf("unexpected %s %v", name, args)
		}
		return []byte(releaseJSON), nil
	}
}

func TestNoticeNamesTheNewerReleaseOnceADay(t *testing.T) {
	state := t.TempDir()
	calls := 0
	now := time.Now()
	var out bytes.Buffer
	printNotice(context.Background(), &out, releasesRunner(&calls), state, "1.0.0", now)
	if out.String() != "agentos v9.0.0 is out: run agentos update\n" {
		t.Errorf("printed %q", out.String())
	}
	out.Reset()
	printNotice(context.Background(), &out, releasesRunner(&calls), state, "9.0.0", now.Add(time.Hour))
	if out.Len() != 0 || calls != 1 {
		t.Errorf("a current build printed %q after %d calls", out.String(), calls)
	}
}

func TestUpdateStopsWhenNothingIsNewer(t *testing.T) {
	app := filepath.Join(t.TempDir(), appBundle)
	if err := os.MkdirAll(app, 0o700); err != nil {
		t.Fatal(err)
	}
	calls := 0
	for current, want := range map[string]string{"9.0.0": "agentos v9.0.0 is the latest release\n", "dev": "this is a dev build; the latest release is v9.0.0\n"} {
		var out bytes.Buffer
		cmd := newRootCmdWith(launcher{places: func() []string { return []string{app} }, attached: func() bool { return true }})
		cmd.SetOut(&out)
		cmd.SetContext(context.Background())
		if err := runUpdate(cmd, []string{app}, releasesRunner(&calls), current); err != nil {
			t.Fatal(err)
		}
		if out.String() != want {
			t.Errorf("%s: printed %q", current, out.String())
		}
	}
}

func TestUpdateNeedsTheApp(t *testing.T) {
	cmd := newRootCmdWith(launcher{places: func() []string { return nil }, attached: func() bool { return true }})
	cmd.SetContext(context.Background())
	calls := 0
	err := runUpdate(cmd, []string{filepath.Join(t.TempDir(), appBundle)}, releasesRunner(&calls), "1.0.0")
	if err == nil || !strings.Contains(err.Error(), "agentos.app not found") || calls != 0 {
		t.Errorf("err = %v after %d calls", err, calls)
	}
}
