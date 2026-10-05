package evidence

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var onePixelPNG, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNkYPhfDwAChwGA60e6kgAAAABJRU5ErkJggg==")

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestEvidenceStore(t *testing.T) {
	e := New(t.TempDir())
	id := "main/3"
	img, err := e.AddImage(id, onePixelPNG, "c", "user")
	if err != nil || !strings.HasPrefix(img.URL, "/media/main/evidence/3/") {
		t.Fatalf("AddImage = %+v, %v", img, err)
	}
	if _, err := e.AddImage(id, []byte("<svg/>"), "", "user"); err == nil {
		t.Error("a non-image was filed as an image")
	}
	if _, err := e.AddText(id, "words", "", "agent"); err != nil {
		t.Fatal(err)
	}
	again := New(e.dir)
	if got := again.List(id); len(got) != 2 || got[0].ID != img.ID || got[1].Text != "words" {
		t.Errorf("after reload: %+v", got)
	}
	if got := again.List("main/4"); len(got) != 0 {
		t.Errorf("another session's evidence: %+v", got)
	}
	if err := again.Delete(id, img.ID); err != nil || again.Count(id) != 1 || exists(e.File(img.URL)) {
		t.Errorf("Delete: %v", err)
	}
	if n := again.Purge(id); n != 1 || again.Count(id) != 0 || exists(filepath.Join(e.dir, "main", "evidence", "3")) {
		t.Errorf("Purge removed %d", n)
	}
	if got := again.List("not a session"); len(got) != 0 {
		t.Errorf("a bad id gave %+v", got)
	}
}
