package bus

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/nednella/agentos/internal/session"
)

const socketFile = "agentos.sock"

// DefaultDir is where state files and the socket live. AGENTOS_STATE_DIR overrides it.
func DefaultDir() (string, error) {
	if dir := os.Getenv("AGENTOS_STATE_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home dir: %w", err)
	}
	return filepath.Join(home, ".local", "state", "agentos"), nil
}

func SocketPath(dir string) string { return filepath.Join(dir, socketFile) }

// DirOf is the state dir that holds the socket at path: hooks learn the dir from AGENTOS_SOCKET.
func DirOf(socket string) string { return filepath.Dir(socket) }

func stateFile(dir, name string) string {
	return filepath.Join(dir, "sessions", name+".json")
}

// WriteState saves rec atomically, or removes the file when the session is gone.
func WriteState(dir string, rec session.Record) error {
	path := stateFile(dir, rec.Session)
	if rec.State == session.Gone {
		return RemoveState(dir, rec.Session)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("creating state dir: %w", err)
	}
	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("encoding state: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return fmt.Errorf("creating temp state file: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing state file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing state file: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("replacing state file: %w", err)
	}
	return nil
}

// ReadState returns the saved record, or the zero record when there is none.
func ReadState(dir, name string) (session.Record, error) {
	data, err := os.ReadFile(stateFile(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return session.Record{}, nil
	}
	if err != nil {
		return session.Record{}, fmt.Errorf("reading state of %s: %w", name, err)
	}
	var rec session.Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return session.Record{}, fmt.Errorf("decoding state of %s: %w", name, err)
	}
	return rec, nil
}

func RemoveState(dir, name string) error {
	if err := os.Remove(stateFile(dir, name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("removing state of %s: %w", name, err)
	}
	return nil
}

// ReadAll returns every saved record by session name. Unreadable files are skipped.
func ReadAll(dir string) map[string]session.Record {
	out := map[string]session.Record{}
	root := filepath.Join(dir, "sessions")
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		var rec session.Record
		if json.Unmarshal(data, &rec) == nil && rec.Session != "" {
			out[rec.Session] = rec
		}
		return nil
	})
	return out
}
