package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const releaseJSON = `{"tag_name":"v1.2.3","assets":[
	{"name":"agentos-darwin-arm64.zip","browser_download_url":"https://x/agentos-darwin-arm64.zip"},
	{"name":"agentos-darwin-arm64.zip.sha256","browser_download_url":"https://x/agentos-darwin-arm64.zip.sha256"}]}`

func TestLatestReadsTheReleaseAndItsAssets(t *testing.T) {
	run := func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "curl" || args[len(args)-1] != latestURL {
			return nil, fmt.Errorf("unexpected %s %v", name, args)
		}
		return []byte(releaseJSON), nil
	}
	rel, err := Latest(context.Background(), run)
	if err != nil {
		t.Fatal(err)
	}
	want := Release{Version: "1.2.3", Zip: "https://x/agentos-darwin-arm64.zip", Sum: "https://x/agentos-darwin-arm64.zip.sha256"}
	if rel != want {
		t.Errorf("got %+v, want %+v", rel, want)
	}

	run = func(context.Context, string, ...string) ([]byte, error) {
		return []byte(`{"tag_name":"v2.0.0","assets":[]}`), nil
	}
	if _, err := Latest(context.Background(), run); err == nil || !strings.Contains(err.Error(), "no agentos-darwin-arm64.zip") {
		t.Errorf("a release without the bundle: %v", err)
	}
}

func TestCheckAsksAgainOnlyWhenTheAnswerIsOld(t *testing.T) {
	state := t.TempDir()
	calls := 0
	run := func(context.Context, string, ...string) ([]byte, error) {
		calls++
		return []byte(releaseJSON), nil
	}
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 2; i++ {
		rel, err := Check(context.Background(), run, state, 24*time.Hour, now.Add(time.Duration(i)*time.Hour))
		if err != nil || rel.Version != "1.2.3" {
			t.Fatalf("check %d: %+v, %v", i, rel, err)
		}
	}
	if calls != 1 {
		t.Errorf("GitHub was asked %d times within a day", calls)
	}
	if _, err := Check(context.Background(), run, state, 24*time.Hour, now.Add(25*time.Hour)); err != nil || calls != 2 {
		t.Errorf("after a day: asked %d times, %v", calls, err)
	}
	if _, err := Check(context.Background(), run, state, 0, now); err != nil || calls != 3 {
		t.Errorf("with no age allowed: asked %d times, %v", calls, err)
	}
}

func TestNewer(t *testing.T) {
	for _, tt := range []struct {
		current, latest string
		want            bool
	}{
		{"1.2.3", "1.2.4", true},
		{"1.2.3", "1.10.0", true},
		{"1.2.3", "2.0.0", true},
		{"1.2.3", "1.2.3", false},
		{"1.2.3", "1.2.2", false},
		{"v1.2.3", "v1.3.0", true},
		{"dev", "1.2.3", false},
		{"1.2.3", "", false},
	} {
		if got := Newer(tt.current, tt.latest); got != tt.want {
			t.Errorf("Newer(%q, %q) = %v", tt.current, tt.latest, got)
		}
	}
}

// fakeDownloads answers curl from files on disk and runs ditto for real.
func fakeDownloads(t *testing.T, files map[string]string) Runner {
	t.Helper()
	return func(ctx context.Context, name string, args ...string) ([]byte, error) {
		if name == "ditto" {
			return exec.CommandContext(ctx, name, args...).Output()
		}
		if name != "curl" {
			return nil, fmt.Errorf("unexpected %s", name)
		}
		url := args[len(args)-1]
		src, ok := files[url]
		if !ok {
			return nil, fmt.Errorf("curl: (22) 404 for %s", url)
		}
		data, err := os.ReadFile(src)
		if err != nil {
			return nil, err
		}
		for i, a := range args {
			if a == "-o" {
				return nil, os.WriteFile(args[i+1], data, 0o600)
			}
		}
		return data, nil
	}
}

// releaseZip makes a zip of a bundle whose binary prints version, and its sha256 file.
func releaseZip(t *testing.T, dir, version string) (zip, sum string) {
	t.Helper()
	app := filepath.Join(dir, bundle)
	if err := os.MkdirAll(filepath.Join(app, "Contents", "MacOS"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", "MacOS", "agentos"), []byte("#!/bin/sh\necho "+version+"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	zip = filepath.Join(dir, Asset)
	if out, err := exec.Command("ditto", "-c", "-k", "--keepParent", app, zip).CombinedOutput(); err != nil {
		t.Fatalf("ditto: %v: %s", err, out)
	}
	data, err := os.ReadFile(zip)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(data)
	sum = zip + ".sha256"
	if err := os.WriteFile(sum, []byte(hex.EncodeToString(h[:])+"  "+Asset+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return zip, sum
}

func TestInstallSwapsTheBundleAfterCheckingTheSum(t *testing.T) {
	if _, err := exec.LookPath("ditto"); err != nil {
		t.Skip("ditto not installed")
	}
	zip, sum := releaseZip(t, t.TempDir(), "2.0.0")
	home := t.TempDir()
	path := filepath.Join(home, "Applications", bundle)
	if err := os.MkdirAll(filepath.Join(path, "Contents", "MacOS"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "Contents", "MacOS", "agentos"), []byte("old"), 0o700); err != nil {
		t.Fatal(err)
	}
	rel := Release{Version: "2.0.0", Zip: "https://x/zip", Sum: "https://x/sum"}
	run := fakeDownloads(t, map[string]string{rel.Zip: zip, rel.Sum: sum})
	if err := Install(context.Background(), run, rel, path); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(filepath.Join(path, "Contents", "MacOS", "agentos")).Output()
	if err != nil || strings.TrimSpace(string(out)) != "2.0.0" {
		t.Errorf("the installed binary printed %q, %v", out, err)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("leftovers beside the bundle: %v", entries)
	}

	bad := filepath.Join(t.TempDir(), "bad.sha256")
	if err := os.WriteFile(bad, []byte(strings.Repeat("0", 64)+"  "+Asset+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run = fakeDownloads(t, map[string]string{rel.Zip: zip, rel.Sum: bad})
	err = Install(context.Background(), run, rel, path)
	if err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Errorf("a wrong sum installed anyway: %v", err)
	}
	if out, _ := exec.Command(filepath.Join(path, "Contents", "MacOS", "agentos")).Output(); strings.TrimSpace(string(out)) != "2.0.0" {
		t.Errorf("the bundle changed after a failed install: %q", out)
	}
}

func TestInstallReportsAFailedDownload(t *testing.T) {
	rel := Release{Version: "2.0.0", Zip: "https://x/zip", Sum: "https://x/sum"}
	path := filepath.Join(t.TempDir(), bundle)
	err := Install(context.Background(), fakeDownloads(t, nil), rel, path)
	var pathErr *os.PathError
	if err == nil || errors.As(err, &pathErr) || !strings.Contains(err.Error(), "downloading https://x/zip") {
		t.Errorf("err = %v", err)
	}
}
