package evidence

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	evidenceFolder = "evidence"
	// MediaPrefix is where the pictures are served.
	MediaPrefix  = "/media/"
	maxImageSize = 10 << 20
)

var imageTypes = map[string]string{
	"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp",
}

// fileOf maps a /media/<key>/evidence/<n>/<file> URL to a picture of the store, or "".
func (e *Store) fileOf(url string) string {
	rel, ok := strings.CutPrefix(path.Clean(url), MediaPrefix)
	if !ok {
		return ""
	}
	parts := strings.SplitN(rel, "/", 4)
	if len(parts) < 4 || parts[1] != evidenceFolder {
		return ""
	}
	file := filepath.Join(e.dir, filepath.FromSlash(rel))
	if !strings.HasPrefix(file, filepath.Join(e.dir, parts[0], evidenceFolder)+string(filepath.Separator)) {
		return ""
	}
	for _, ext := range imageTypes {
		if strings.EqualFold(filepath.Ext(file), ext) {
			return file
		}
	}
	return ""
}

// File is the file behind the URL of a picture, or "".
func (e *Store) File(url string) string { return e.fileOf(url) }

// MediaHandler serves the pictures of the store, and nothing else in it.
func (e *Store) MediaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := e.fileOf(r.URL.Path)
		if file == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(file)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil || !info.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		http.ServeContent(w, r, file, info.ModTime(), f)
	})
}
