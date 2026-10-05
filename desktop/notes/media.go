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
	"slices"

	"github.com/nednella/agentos/desktop/internal/media"
	"github.com/nednella/agentos/internal/atomicfile"
)

func (s *Notes) mediaPath(url string) string { return media.File(s.dir, media.NotesFolder, url) }

// removeFile deletes the image behind a /media/ URL; anything else is left alone.
func (s *Notes) removeFile(url string) {
	if file := s.mediaPath(url); file != "" {
		_ = os.Remove(file)
	}
}

// AddImage stores a picture beside the notes and adds its URL to the note.
func (s *Notes) AddImage(key, id, b64, mime string) (Note, error) {
	ext, ok := media.Ext(mime)
	if !ok {
		return Note{}, fmt.Errorf("%q is not a png, jpeg, gif or webp picture", mime)
	}
	if base64.StdEncoding.DecodedLen(len(b64)) > media.MaxImageSize+3 {
		return Note{}, errors.New("the picture is over 10 MB")
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return Note{}, fmt.Errorf("reading the picture: %w", err)
	}
	if len(data) > media.MaxImageSize {
		return Note{}, errors.New("the picture is over 10 MB")
	}
	if http.DetectContentType(data) != mime {
		return Note{}, fmt.Errorf("the data is not a %s picture", mime)
	}
	name := make([]byte, 12)
	if _, err := rand.Read(name); err != nil {
		return Note{}, fmt.Errorf("naming the picture: %w", err)
	}
	url := path.Join(media.Prefix, key, media.NotesFolder, hex.EncodeToString(name)+ext)
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
