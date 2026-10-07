package notes_test

import (
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nednella/agentos/desktop/internal/app"
	"github.com/nednella/agentos/desktop/internal/apptest"
	"github.com/nednella/agentos/desktop/notes"
)

func TestNotes(t *testing.T) {
	h := newHarness(t)
	blank, err := h.AddNote("  ") // a note that waits for its pictures
	if err != nil || blank.Text != "" {
		t.Fatalf("AddNote with no text = %+v, %v", blank, err)
	}
	if _, err := h.UpdateNote(blank.ID, "  "); err == nil {
		t.Error("a note was emptied by an edit")
	}
	if err := h.DeleteNote(blank.ID); err != nil {
		t.Fatal(err)
	}
	first, err := h.AddNote("first title\nbody line")
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Millisecond)
	second, err := h.AddNote("second")
	if err != nil {
		t.Fatal(err)
	}
	ids := func(ns []notes.Note) string {
		var out []string
		for _, n := range ns {
			out = append(out, n.Text)
		}
		return strings.Join(out, "|")
	}
	if got := ids(h.Snapshot().Notes); got != "second|first title\nbody line" {
		t.Errorf("order = %q", got)
	}
	if _, err := h.SetNoteArchived(second.ID, true); err != nil {
		t.Fatal(err)
	}
	if n, err := h.UpdateNote(first.ID, "edited"); err != nil || n.Text != "edited" || n.UpdatedAt < first.UpdatedAt {
		t.Errorf("UpdateNote = %+v, %v", n, err)
	}
	if got := ids(h.Snapshot().Notes); got != "edited|second" {
		t.Errorf("done notes sort last: %q", got)
	}
	if got := h.Rec.ProjectOf("notes"); got != "main" {
		t.Errorf("notes event project = %q", got)
	}
	if events := h.Rec.LastNotes(); ids(events) != "edited|second" {
		t.Errorf("notes event = %q", ids(events))
	}

	fresh := h.Restart(t)
	if got := ids(fresh.Snapshot().Notes); got != "edited|second" || !fresh.Snapshot().Notes[1].Archived {
		t.Errorf("after reload: %q", got)
	}

	if err := h.DeleteNote(second.ID); err != nil {
		t.Fatal(err)
	}
	if err := h.DeleteNote(second.ID); err == nil {
		t.Error("deleting a deleted note succeeded")
	}
	if _, err := h.UpdateNote("nope", "x"); err == nil {
		t.Error("updating an unknown note succeeded")
	}
	if got := ids(h.Snapshot().Notes); got != "edited" {
		t.Errorf("after delete: %q", got)
	}
	if got := h.Snapshot().Version; got == "" {
		t.Error("no version in the snapshot")
	}
}

func TestNoteToIssue(t *testing.T) {
	h := newHarness(t)
	h.GH.Create = "Creating issue in acme/widgets\n\nhttps://github.com/acme/widgets/issues/55\n"
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	n, err := h.AddNote("Fix login\n\nIt fails on Safari.\nSecond line.")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.NoteToIssue(n.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Notes().Get("main", n.ID); err == nil {
		t.Error("the note is still there after filing")
	}
	var create string
	for _, c := range h.GH.CallLog() {
		if strings.HasPrefix(c, "issue create") {
			create = c
		}
	}
	if want := "issue create --title Fix login --body ## Description\n\nFix login\n\nIt fails on Safari.\nSecond line. --assignee @me"; create != want {
		t.Errorf("gh call = %q, want %q", create, want)
	}
	if h.GH.IssueCalls() != 2 {
		t.Errorf("issue list ran %d times, want a refresh", h.GH.IssueCalls())
	}
	if h.Rec.Last("issues") == nil {
		t.Error("no issues event")
	}
	if got := h.Rec.LastNotes(); len(got) != 0 {
		t.Errorf("notes event = %+v", got)
	}

	h.GH.Create = "something odd\n"
	odd, _ := h.AddNote("odd")
	if err := h.NoteToIssue(odd.ID); err == nil {
		t.Error("output without an address was accepted")
	}
	if _, err := h.Notes().Get("main", odd.ID); err != nil {
		t.Error("a failed filing deleted the note")
	}
}

func TestNoteToIssueWithoutRepo(t *testing.T) {
	h := newHarness(t)
	h.GH.Repo = errors.New("not a repo")
	n, _ := h.AddNote("x")
	if err := h.NoteToIssue(n.ID); err == nil {
		t.Error("filed an issue without a repo")
	}
}

func TestNoteToSession(t *testing.T) {
	h := newHarness(t)
	n, _ := h.AddNote("remember this")
	s, err := h.NoteToSession(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	if s.Title != "remember this" {
		t.Errorf("title = %q", s.Title)
	}
	eventually(t, "note text typed", func() bool { return strings.Contains(h.Pane(t, s.ID), "remember this") })
}

func TestNoteToSessionKeepsLineBreaksUnsent(t *testing.T) {
	h := apptest.NewWith(t, apptest.Options{ProjectExtra: "    session_prompt_send: manual\n"})
	// Pasted before bash's prompt, the line breaks would run: type only once the session reports ready.
	h.App.Sessions().SetPrefillWait(time.Minute)
	n, _ := h.AddNote("first line\nsecond line")
	s, err := h.NoteToSession(n.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventually(t, "the shell's prompt", func() bool { return strings.HasSuffix(strings.TrimSpace(h.Pane(t, s.ID)), "$") })
	h.Hook(t, s.ID, "SessionStart", `{}`)
	eventually(t, "both lines typed", func() bool {
		pane := h.Pane(t, s.ID)
		return strings.Contains(pane, "first line") && strings.Contains(pane, "second line")
	})
	time.Sleep(300 * time.Millisecond)
	if pane := h.Pane(t, s.ID); strings.Contains(pane, "not found") {
		t.Errorf("the note was submitted:\n%s", pane)
	}
}

func TestNoteOrderAndFlags(t *testing.T) {
	h := newHarness(t)
	old, _ := h.AddNote("old")
	time.Sleep(3 * time.Millisecond)
	archivedLater, _ := h.AddNote("archived later")
	time.Sleep(3 * time.Millisecond)
	fresh, _ := h.AddNote("fresh")
	if n, err := h.SetNotePinned(old.ID, true); err != nil || !n.Pinned {
		t.Fatalf("SetNotePinned = %+v, %v", n, err)
	}
	order := func() string {
		var ids []string
		for _, n := range h.Snapshot().Notes {
			ids = append(ids, n.ID)
		}
		return strings.Join(ids, ",")
	}
	if got, want := order(), old.ID+","+fresh.ID+","+archivedLater.ID; got != want {
		t.Errorf("pinned first, then newest: %s, want %s", got, want)
	}
	if _, err := h.SetNoteArchived(archivedLater.ID, true); err != nil {
		t.Fatal(err)
	}
	if got, want := order(), old.ID+","+fresh.ID+","+archivedLater.ID; got != want {
		t.Errorf("archived last: %s, want %s", got, want)
	}
	if _, err := h.SetNoteArchived(old.ID, true); err != nil {
		t.Fatal(err)
	}
	if got := order(); !strings.HasPrefix(got, fresh.ID) {
		t.Errorf("an archived pinned note stays below the open ones: %s", got)
	}
	if _, err := h.SetNoteArchived(old.ID, false); err != nil {
		t.Fatal(err)
	}
	if got := order(); !strings.HasPrefix(got, old.ID) {
		t.Errorf("an unarchived pinned note goes back to the top: %s", got)
	}
}

var onePixelPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")

func TestNoteImages(t *testing.T) {
	h := newHarness(t)
	n, _ := h.AddNote("with picture")
	b64 := base64.StdEncoding.EncodeToString(onePixelPNG)

	n, err := h.AddNoteImage(n.ID, b64, "image/png")
	if err != nil || len(n.Images) != 1 || !strings.HasPrefix(n.Images[0], "/media/main/notes-media/") || !strings.HasSuffix(n.Images[0], ".png") {
		t.Fatalf("AddNoteImage = %+v, %v", n, err)
	}
	url := n.Images[0]
	bad := []struct{ name, data, mime string }{
		{"svg", b64, "image/svg+xml"},
		{"html as png", base64.StdEncoding.EncodeToString([]byte("<html></html>")), "image/png"},
		{"not base64", "%%%", "image/png"},
		{"over 10 MB", base64.StdEncoding.EncodeToString(append(slices.Clone(onePixelPNG), make([]byte, 10<<20)...)), "image/png"},
	}
	for _, tt := range bad {
		if _, err := h.AddNoteImage(n.ID, tt.data, tt.mime); err == nil {
			t.Errorf("%s was accepted", tt.name)
		}
	}
	if _, err := h.AddNoteImage("nope", b64, "image/png"); err == nil {
		t.Error("an image for an unknown note was accepted")
	}
	if files, _ := os.ReadDir(filepath.Join(h.State, "data", "main", "notes-media")); len(files) != 1 {
		t.Errorf("%d files stored, want 1", len(files))
	}

	srv := httptest.NewServer(h.App.Media())
	defer srv.Close()
	get := func(path string) (int, string, []byte) {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body := make([]byte, 4096)
		k, _ := resp.Body.Read(body)
		return resp.StatusCode, resp.Header.Get("Content-Type"), body[:k]
	}
	if code, ctype, body := get(url); code != 200 || ctype != "image/png" || string(body) != string(onePixelPNG) {
		t.Errorf("GET %s = %d %s", url, code, ctype)
	}
	for _, p := range []string{
		"/media/main/notes.json",
		"/media/main/notes-media/../notes.json",
		"/media/main/notes-media/../../main/notes.json",
		"/media/main/notes-media/..%2fnotes.json",
		"/media/../../../etc/passwd",
		"/media/main/notes-media/",
		"/media/main/notes-media/missing.png",
		"/other",
	} {
		if code, _, _ := get(p); code != 404 {
			t.Errorf("GET %s = %d, want 404", p, code)
		}
	}

	if _, err := h.RemoveNoteImage(n.ID, "/media/main/notes-media/other.png"); err == nil {
		t.Error("removed a picture the note does not have")
	}
	n, err = h.RemoveNoteImage(n.ID, url)
	if err != nil || len(n.Images) != 0 || exists(h.Notes().MediaFile(url)) {
		t.Errorf("RemoveNoteImage = %+v, %v", n, err)
	}
	n, _ = h.AddNoteImage(n.ID, b64, "image/png")
	file := h.Notes().MediaFile(n.Images[0])
	if err := h.DeleteNote(n.ID); err != nil || exists(file) {
		t.Errorf("deleting the note left its picture: %v", err)
	}

	h.GH.Create = "https://github.com/acme/widgets/issues/9\n"
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	n, _ = h.AddNote("filed with picture")
	n, _ = h.AddNoteImage(n.ID, b64, "image/png")
	file = h.Notes().MediaFile(n.Images[0])
	if err := h.NoteToIssue(n.ID); err != nil || exists(file) {
		t.Errorf("filing the note left its picture: %v", err)
	}
	var create string
	for _, c := range h.GH.CallLog() {
		if strings.HasPrefix(c, "issue create") {
			create = c
		}
	}
	if want := " --attach " + file + "#Picture 1"; !strings.HasSuffix(create, want) {
		t.Errorf("gh call = %q, want it to end with %q", create, want)
	}

	h.GH.CreateErr = errors.New("gh: exit status 1: uploading big.png: too large")
	n, _ = h.AddNote("filed without its picture")
	n, _ = h.AddNoteImage(n.ID, b64, "image/png")
	file = h.Notes().MediaFile(n.Images[0])
	err = h.NoteToIssue(n.ID)
	if err == nil || !strings.Contains(err.Error(), "filed https://github.com/acme/widgets/issues/9 but") {
		t.Errorf("NoteToIssue = %v, want the filed issue named", err)
	}
	if _, err := h.Notes().Get("main", n.ID); err != nil || !exists(file) {
		t.Error("a half-filed note or its picture was deleted")
	}
}

func TestMediaServedInHTTPMode(t *testing.T) {
	h := newHarness(t)
	n, _ := h.AddNote("x")
	n, err := h.AddNoteImage(n.ID, base64.StdEncoding.EncodeToString(onePixelPNG), "image/png")
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/media/", h.App.Media())
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest("GET", n.Images[0], nil))
	if rec.Code != 200 || rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Errorf("status %d, headers %v", rec.Code, rec.Header())
	}
}

func TestNotesLiveInTheDataDir(t *testing.T) {
	root := t.TempDir()
	synced := filepath.Join(root, "cloud", "agentos")
	t.Setenv("HOME", filepath.Join(root, "home"))
	t.Setenv("AGENTOS_DEV_DATA_DIR", "")
	conf := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(conf, []byte("data_dir: "+synced+"\nprojects:\n  - {name: p, directory: "+root+"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AGENTOS_CONFIG", conf)
	t.Setenv("AGENTOS_STATE_DIR", filepath.Join(root, "state"))
	t.Setenv("AGENTOS_DIR", root)
	t.Setenv("AGENTOS_TMUX_SOCKET", "aostest-unused")

	cfg, err := app.Load()
	if err != nil || cfg.DataDir != synced {
		t.Fatalf("dataDir = %q, %v", cfg.DataDir, err)
	}
	a := app.New(cfg, app.Host{Emit: func(string, any) {}, Clipboard: func(string) {}}, apptest.NoGH, apptest.NoStream, apptest.NoClaude)
	if _, err := addNote(a, "kept in the synced folder"); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(synced, "p", "notes.json")) {
		t.Error("the note was not written to data_dir")
	}

	t.Setenv("AGENTOS_DEV_DATA_DIR", filepath.Join(root, "override"))
	if cfg, _ = app.Load(); cfg.DataDir != filepath.Join(root, "override") {
		t.Errorf("the override gave %q", cfg.DataDir)
	}

	t.Setenv("AGENTOS_DATA_DIR", filepath.Join(root, "old"))
	if _, err := app.Load(); err == nil || !strings.Contains(err.Error(), "AGENTOS_DEV_DATA_DIR") {
		t.Errorf("the old name gave %v", err)
	}
}

// addNote calls the notes service of an app the way the front end does.
func addNote(a *app.App, text string) (notes.Note, error) {
	for _, svc := range a.Services() {
		if s, ok := svc.(*notes.Service); ok {
			return s.AddNote(text)
		}
	}
	return notes.Note{}, errors.New("no notes service")
}

func TestNoteToIssueFilesOneIssueUnderConcurrentCalls(t *testing.T) {
	h := newHarness(t)
	h.GH.Create = "https://github.com/acme/widgets/issues/61\n"
	h.GH.Delay = 300 * time.Millisecond
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	n, err := h.AddNote("Double click")
	if err != nil {
		t.Fatal(err)
	}
	errs := make([]error, 3)
	var wg sync.WaitGroup
	for i := range errs {
		wg.Go(func() { errs[i] = h.NoteToIssue(n.ID) })
	}
	wg.Wait()

	created, failed := 0, 0
	for _, c := range h.GH.CallLog() {
		if strings.HasPrefix(c, "issue create") {
			created++
		}
	}
	for _, err := range errs {
		if err != nil {
			failed++
		}
	}
	if created != 1 || failed != 2 {
		t.Errorf("%d issues created, %d calls failed; want 1 and 2 (%v)", created, failed, errs)
	}
	if _, err := h.Notes().Get("main", n.ID); err == nil {
		t.Error("the note is still there after filing")
	}
}

func TestNoteWithoutTextCannotBecomeAnIssue(t *testing.T) {
	h := newHarness(t)
	if _, err := h.Issues(false); err != nil {
		t.Fatal(err)
	}
	n, _ := h.AddNote("")
	if err := h.NoteToIssue(n.ID); err == nil {
		t.Error("a note with no text was filed")
	}
	for _, c := range h.GH.CallLog() {
		if strings.HasPrefix(c, "issue create") {
			t.Errorf("gh was asked to file it: %s", c)
		}
	}
}
