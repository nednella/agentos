// Package media serves the pictures of notes and evidence under /media/<project key>/<folder>/...
package media

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	// Prefix is where the pictures are served.
	Prefix = "/media/"
	// NotesFolder holds a project's note pictures; EvidenceFolder holds the pictures of its sessions' evidence.
	NotesFolder    = "notes-media"
	EvidenceFolder = "evidence"
	// MaxImageSize is the largest picture agentos stores.
	MaxImageSize = 10 << 20
)

var imageTypes = map[string]string{
	"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp",
}

// Ext is the file extension for a picture type, and whether the type is one agentos stores.
func Ext(mime string) (string, bool) {
	ext, ok := imageTypes[mime]
	return ext, ok
}

// File maps a /media/<project key>/<folder>/... URL to a picture below dir, or "" for anything else.
func File(dir, folder, url string) string {
	rel, ok := strings.CutPrefix(path.Clean(url), Prefix)
	if !ok {
		return ""
	}
	parts := strings.SplitN(rel, "/", 3)
	if len(parts) < 3 || parts[1] != folder {
		return ""
	}
	file := filepath.Join(dir, filepath.FromSlash(rel))
	if !strings.HasPrefix(file, filepath.Join(dir, parts[0], folder)+string(filepath.Separator)) {
		return ""
	}
	for _, ext := range imageTypes {
		if strings.EqualFold(filepath.Ext(file), ext) {
			return file
		}
	}
	return ""
}

// Handler serves the pictures of notes and evidence kept under dir, and nothing else in it.
func Handler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := File(dir, NotesFolder, r.URL.Path)
		if file == "" {
			file = File(dir, EvidenceFolder, r.URL.Path)
		}
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
