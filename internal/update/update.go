// Package update finds, downloads and installs a newer release of agentos.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nednella/agentos/internal/atomicfile"
)

const (
	// Repo is where the releases are.
	Repo = "nednella/agentos"
	// Asset is the zip of the app bundle attached to each release; its sha256 sits beside it as Asset + ".sha256".
	Asset = "agentos-darwin-arm64.zip"

	latestURL = "https://api.github.com/repos/" + Repo + "/releases/latest"
	cacheFile = "update.json"
	bundle    = "agentos.app"
)

// Runner runs a command and returns its stdout: curl and ditto. Tests inject one that never
// reaches the network.
type Runner func(ctx context.Context, name string, args ...string) ([]byte, error)

// Release is a published version and where its bundle is.
type Release struct {
	Version string `json:"version"` // "1.2.3", without the v of the tag
	Zip     string `json:"zip"`     // where to download the bundle
	Sum     string `json:"sum"`     // where its sha256 is
}

// Latest asks GitHub for the newest release.
func Latest(ctx context.Context, run Runner) (Release, error) {
	out, err := run(ctx, "curl", "-fsSL", "--max-time", "10", "-H", "Accept: application/vnd.github+json", latestURL)
	if err != nil {
		return Release{}, fmt.Errorf("asking GitHub for the latest release: %w", err)
	}
	var body struct {
		Tag    string `json:"tag_name"`
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(out, &body); err != nil {
		return Release{}, fmt.Errorf("reading the latest release: %w", err)
	}
	rel := Release{Version: strings.TrimPrefix(body.Tag, "v")}
	for _, a := range body.Assets {
		switch a.Name {
		case Asset:
			rel.Zip = a.URL
		case Asset + ".sha256":
			rel.Sum = a.URL
		}
	}
	if rel.Version == "" || rel.Zip == "" || rel.Sum == "" {
		return Release{}, fmt.Errorf("release %q has no %s with its sha256", body.Tag, Asset)
	}
	return rel, nil
}

type cache struct {
	CheckedAt time.Time `json:"checkedAt"`
	Release   Release   `json:"release"`
}

// Check is Latest with a memory: the answer is kept in stateDir, and one younger than maxAge
// is returned without asking again.
func Check(ctx context.Context, run Runner, stateDir string, maxAge time.Duration, now time.Time) (Release, error) {
	path := filepath.Join(stateDir, cacheFile)
	var c cache
	if data, err := os.ReadFile(path); err == nil && json.Unmarshal(data, &c) == nil && maxAge > 0 && now.Sub(c.CheckedAt) < maxAge {
		return c.Release, nil
	}
	rel, err := Latest(ctx, run)
	if err != nil {
		return Release{}, err
	}
	data, err := json.Marshal(cache{CheckedAt: now, Release: rel})
	if err != nil {
		return Release{}, fmt.Errorf("encoding the latest release: %w", err)
	}
	if err := atomicfile.Write(path, data, 0o600); err != nil {
		return Release{}, err
	}
	return rel, nil
}

// Newer says whether latest is a higher version than current. A dev build is never behind.
func Newer(current, latest string) bool {
	a, okA := parse(current)
	b, okB := parse(latest)
	if !okA || !okB {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return b[i] > a[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var n [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return n, false
	}
	for i, p := range parts {
		x, err := strconv.Atoi(p)
		if err != nil || x < 0 {
			return n, false
		}
		n[i] = x
	}
	return n, true
}

// Install downloads rel, checks its sha256 and puts its bundle in place of the one at path.
// The download goes through curl, which sets no quarantine flag on what it fetches.
func Install(ctx context.Context, run Runner, rel Release, path string) error {
	work, err := os.MkdirTemp(filepath.Dir(path), ".agentos-update-") // beside the bundle: the swap is then a rename
	if err != nil {
		return fmt.Errorf("making room for the download: %w", err)
	}
	defer os.RemoveAll(work)
	zip := filepath.Join(work, Asset)
	if _, err := run(ctx, "curl", "-fsSL", "--max-time", "600", "-o", zip, rel.Zip); err != nil {
		return fmt.Errorf("downloading %s: %w", rel.Zip, err)
	}
	sum, err := run(ctx, "curl", "-fsSL", "--max-time", "60", rel.Sum)
	if err != nil {
		return fmt.Errorf("downloading %s: %w", rel.Sum, err)
	}
	if err := verify(zip, string(sum)); err != nil {
		return err
	}
	if _, err := run(ctx, "ditto", "-x", "-k", zip, work); err != nil {
		return fmt.Errorf("unpacking %s: %w", Asset, err)
	}
	fresh := filepath.Join(work, bundle)
	if _, err := os.Stat(filepath.Join(fresh, "Contents", "MacOS", "agentos")); err != nil {
		return fmt.Errorf("%s holds no %s", Asset, bundle)
	}
	return swap(fresh, path)
}

// verify checks the file against a sha256 line of the form "<hex>  <name>".
func verify(path, sum string) error {
	want := strings.Fields(sum)
	if len(want) == 0 {
		return errors.New("the release's sha256 file is empty")
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want[0] {
		return fmt.Errorf("the download's sha256 is %s, the release says %s", got, want[0])
	}
	return nil
}

// swap puts fresh at path and removes what was there. The old bundle is moved aside first, so a
// running app keeps its files until the move succeeds.
func swap(fresh, path string) error {
	old := path + ".old"
	_ = os.RemoveAll(old)
	if err := os.Rename(path, old); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("moving the old app aside: %w", err)
	}
	if err := os.Rename(fresh, path); err != nil {
		_ = os.Rename(old, path)
		return fmt.Errorf("putting the new app in place: %w", err)
	}
	return os.RemoveAll(old)
}

// Relaunch opens the bundle at path once this process has ended; the caller quits right after.
func Relaunch(path string) error {
	script := "while kill -0 $1 2>/dev/null; do sleep 0.2; done; open \"$2\""
	cmd := exec.Command("sh", "-c", script, "sh", strconv.Itoa(os.Getpid()), path)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("arranging the relaunch: %w", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
