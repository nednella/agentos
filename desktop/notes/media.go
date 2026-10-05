package notes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/nednella/agentos/internal/atomicfile"
)

const (
	mediaPrefix  = "/media/"
	mediaFolder  = "notes-media"
	maxImageSize = 10 << 20
)

var imageTypes = map[string]string{
	"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp",
}

// mediaFile maps a /media/<project key>/<folder>/... URL to a picture in the folder that holds that kind
// of picture (roots maps the URL's second folder to its dir), or "" for anything else.
func mediaFile(roots map[string]string, url string) string {
	rel, ok := strings.CutPrefix(path.Clean(url), mediaPrefix)
	if !ok {
		return ""
	}
	parts := strings.SplitN(rel, "/", 3)
	if len(parts) < 3 {
		return ""
	}
	key, folder := parts[0], parts[1]
	dir, ok := roots[folder]
	if !ok {
		return ""
	}
	file := filepath.Join(dir, filepath.FromSlash(rel))
	if !strings.HasPrefix(file, filepath.Join(dir, key, folder)+string(filepath.Separator)) || !slices.Contains(mapValues(imageTypes), strings.ToLower(filepath.Ext(file))) {
		return ""
	}
	return file
}

func (s *Notes) mediaPath(url string) string {
	return mediaFile(map[string]string{mediaFolder: s.dir}, url)
}

func mapValues(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// MediaHandler serves the pictures of the notes under /media/.
func (s *Notes) MediaHandler() http.Handler {
	return mediaHandler(map[string]string{mediaFolder: s.dir})
}

// mediaHandler serves the notes' and evidence pictures from the data dir, and nothing else in it.
func mediaHandler(roots map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		file := mediaFile(roots, r.URL.Path)
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

// removeFile deletes the image behind a /media/ URL; anything else is left alone.
func (s *Notes) removeFile(url string) {
	if file := s.mediaPath(url); file != "" {
		_ = os.Remove(file)
	}
}

// AddImage stores a picture beside the notes and adds its URL to the note.
func (s *Notes) AddImage(key, id, b64, mime string) (Note, error) {
	ext, ok := imageTypes[mime]
	if !ok {
		return Note{}, fmt.Errorf("%q is not a png, jpeg, gif or webp picture", mime)
	}
	if base64.StdEncoding.DecodedLen(len(b64)) > maxImageSize+3 {
		return Note{}, errors.New("the picture is over 10 MB")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return Note{}, fmt.Errorf("reading the picture: %w", err)
	}
	if len(data) > maxImageSize {
		return Note{}, errors.New("the picture is over 10 MB")
	}
	if http.DetectContentType(data) != mime {
		return Note{}, fmt.Errorf("the data is not a %s picture", mime)
	}
	name := make([]byte, 12)
	if _, err := rand.Read(name); err != nil {
		return Note{}, fmt.Errorf("naming the picture: %w", err)
	}
	url := path.Join(mediaPrefix, key, mediaFolder, hex.EncodeToString(name)+ext)
	file := s.mediaPath(url)
	if file == "" {
		return Note{}, errors.New("the picture has no place to go")
	}
	if err := atomicfile.Write(file, data, 0o600); err != nil {
		return Note{}, err
	}
	n, err := s.edit(key, id, func(n *Note) error { n.Images = append(n.Images, url); return nil })
	if err != nil {
		_ = os.Remove(file)
	}
	return n, err
}

func (s *Notes) RemoveImage(key, id, url string) (Note, error) {
	n, err := s.edit(key, id, func(n *Note) error {
		i := slices.Index(n.Images, url)
		if i < 0 {
			return errors.New("the note has no such picture")
		}
		n.Images = slices.Delete(n.Images, i, i+1)
		return nil
	})
	if err == nil {
		s.removeFile(url)
	}
	return n, err
}

// MediaFile is the file behind a /media/ URL of a note picture, or "".
func (s *Notes) MediaFile(url string) string { return s.mediaPath(url) }
