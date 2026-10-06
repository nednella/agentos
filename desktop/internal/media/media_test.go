package media

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerServesBothKinds(t *testing.T) {
	dir := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, "main", "notes-media", "a.png"), "note picture")
	write(filepath.Join(dir, "main", "evidence", "3", "b.JPG"), "evidence picture")
	write(filepath.Join(dir, "main", "notes.json"), "secret")
	write(filepath.Join(dir, "main", "notes-media", "page.html"), "<script>")
	write(filepath.Join(dir, "main", "evidence", "3", "index.json"), "secret")

	h := Handler(dir)
	tests := []struct {
		url  string
		want int
		body string
	}{
		{"/media/main/notes-media/a.png", 200, "note picture"},
		{"/media/main/evidence/3/b.JPG", 200, "evidence picture"},
		{"/media/main/notes.json", 404, ""},
		{"/media/main/notes-media/page.html", 404, ""},
		{"/media/main/evidence/3/index.json", 404, ""},
		{"/media/main/notes-media/../notes.json", 404, ""},
		{"/media/main/notes-media/..%2fnotes.json", 404, ""},
		{"/media/../../../etc/passwd", 404, ""},
		{"/media/main/notes-media/", 404, ""},
		{"/media/main/notes-media/missing.png", 404, ""},
		{"/other", 404, ""},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest("GET", tt.url, nil))
			if rec.Code != tt.want || (tt.want == 200 && rec.Body.String() != tt.body) {
				t.Errorf("status %d body %q, want %d %q", rec.Code, rec.Body.String(), tt.want, tt.body)
			}
			if tt.want == 200 && rec.Header().Get("X-Content-Type-Options") != "nosniff" {
				t.Errorf("headers %v", rec.Header())
			}
		})
	}
}

func TestExt(t *testing.T) {
	for mime, want := range map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/webp": ".webp", "image/svg+xml": "", "": ""} {
		if got, ok := Ext(mime); got != want || ok != (want != "") {
			t.Errorf("Ext(%q) = %q, %v", mime, got, ok)
		}
	}
}
