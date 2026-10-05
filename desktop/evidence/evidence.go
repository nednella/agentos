// Package evidence keeps what a session files for the owner to see: pictures and text cards.
package evidence

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/nednella/agentos/desktop/internal/media"
	"github.com/nednella/agentos/internal/atomicfile"
	"github.com/nednella/agentos/internal/session"
)

// Evidence is one thing a session shows the owner: a picture or a text card.
type Evidence struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	URL     string `json:"url"`
	Text    string `json:"text"`
	Caption string `json:"caption"`
	Source  string `json:"source"`
	At      int64  `json:"at"`
}

// Store keeps each session's evidence in its own folder of the data dir.
type Store struct {
	dir string

	mu    sync.Mutex
	items map[string][]Evidence // by session id
}

// New keeps the evidence under dataDir.
func New(dataDir string) *Store {
	return &Store{dir: dataDir, items: map[string][]Evidence{}}
}

// File is the file behind the URL of a picture, or "".
func (e *Store) File(url string) string { return media.File(e.dir, media.EvidenceFolder, url) }

// folder is where the session's evidence lives. The session's number is reused
// after it ends, so the folder is emptied when a session ends.
func (e *Store) folder(id string) (string, error) {
	name, err := session.ParseName(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(e.dir, name.Project, media.EvidenceFolder, strconv.Itoa(name.N)), nil
}

// load needs mu.
func (e *Store) load(id string) ([]Evidence, error) {
	if items, ok := e.items[id]; ok {
		return items, nil
	}
	dir, err := e.folder(id)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(dir, "index.json"))
	var items []Evidence
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("reading evidence: %w", err)
	default:
		if err := json.Unmarshal(data, &items); err != nil {
			return nil, fmt.Errorf("decoding evidence: %w", err)
		}
	}
	e.items[id] = items
	return items, nil
}

// save needs mu.
func (e *Store) save(id string, items []Evidence) error {
	dir, err := e.folder(id)
	if err != nil {
		return err
	}
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("encoding evidence: %w", err)
	}
	if err := atomicfile.Write(filepath.Join(dir, "index.json"), data, 0o600); err != nil {
		return err
	}
	e.items[id] = items
	return nil
}

// List is the session's evidence, oldest first.
func (e *Store) List(id string) []Evidence {
	if e == nil {
		return []Evidence{}
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	items, err := e.load(id)
	if err != nil {
		return []Evidence{}
	}
	return append([]Evidence{}, items...)
}

func (e *Store) Count(id string) int {
	if e == nil {
		return 0
	}
	return len(e.List(id))
}

func newEvidenceID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("naming evidence: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// AddImage files a picture. Its type comes from the bytes, not from the name.
func (e *Store) AddImage(id string, data []byte, caption, source string) (Evidence, error) {
	ext, ok := media.Ext(http.DetectContentType(data))
	if !ok {
		return Evidence{}, errors.New("the file is not a png, jpeg, gif or webp picture")
	}
	if len(data) > media.MaxImageSize {
		return Evidence{}, errors.New("the picture is over 10 MB")
	}
	name, err := newEvidenceID()
	if err != nil {
		return Evidence{}, err
	}
	dir, err := e.folder(id)
	if err != nil {
		return Evidence{}, err
	}
	if err := atomicfile.Write(filepath.Join(dir, name+ext), data, 0o600); err != nil {
		return Evidence{}, err
	}
	parsed, _ := session.ParseName(id)
	item := Evidence{ID: name, Kind: "image", Caption: caption, Source: source, At: time.Now().UnixMilli(),
		URL: path.Join(media.Prefix, parsed.Project, media.EvidenceFolder, strconv.Itoa(parsed.N), name+ext)}
	return e.add(id, item, filepath.Join(dir, name+ext))
}

// AddText files a text card.
func (e *Store) AddText(id, text, caption, source string) (Evidence, error) {
	name, err := newEvidenceID()
	if err != nil {
		return Evidence{}, err
	}
	return e.add(id, Evidence{ID: name, Kind: "text", Text: text, Caption: caption, Source: source, At: time.Now().UnixMilli()}, "")
}

// add records the item; file is the picture to remove again if that fails.
func (e *Store) add(id string, item Evidence, file string) (Evidence, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	items, err := e.load(id)
	if err == nil {
		err = e.save(id, append(slices.Clone(items), item))
	}
	if err != nil && file != "" {
		_ = os.Remove(file)
	}
	return item, err
}

func (e *Store) Delete(id, evidenceID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	items, err := e.load(id)
	if err != nil {
		return err
	}
	i := slices.IndexFunc(items, func(it Evidence) bool { return it.ID == evidenceID })
	if i < 0 {
		return errors.New("no such evidence")
	}
	if file := e.File(items[i].URL); file != "" {
		_ = os.Remove(file)
	}
	return e.save(id, slices.Delete(slices.Clone(items), i, i+1))
}

// Purge removes everything the session filed, and reports how many items there were.
func (e *Store) Purge(id string) int {
	if e == nil {
		return 0
	}
	dir, err := e.folder(id)
	if err != nil {
		return 0
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	items, _ := e.load(id)
	delete(e.items, id)
	if err := os.RemoveAll(dir); err != nil {
		fmt.Fprintf(os.Stderr, "agentos: removing evidence: %v\n", err)
	}
	return len(items)
}
