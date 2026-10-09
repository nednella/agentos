package settings

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// dataNames are what each project keeps in the data folder. The rest of a project's folder there may be
// the local folder's, which never moves.
var dataNames = []string{"notes.json", "notes-media", "evidence", "stats.jsonl", "activity.json", "digest.json"}

var errDataFixed = errors.New("AGENTOS_DEV_DATA_DIR sets the data folder, so it cannot change here")

// DataDirChoice is a folder picked for the data, and whether it is empty, so the data can be copied in.
type DataDirChoice struct {
	Dir   string `json:"dir"` // "" when the user cancelled
	Empty bool   `json:"empty"`
}

// PickDataDir asks for a folder to keep the data in. It changes nothing.
func (s *Service) PickDataDir() (DataDirChoice, error) {
	if s.dataFixed {
		return DataDirChoice{}, errDataFixed
	}
	dir, err := s.pickDir()
	if err != nil {
		return DataDirChoice{}, fmt.Errorf("choosing a folder: %w", err)
	}
	if dir == "" {
		return DataDirChoice{}, nil
	}
	empty, err := isEmpty(dir)
	return DataDirChoice{Dir: dir, Empty: empty}, err
}

// SetDataDir makes dir the data folder and relaunches the app to use it. With withData, the current data is
// copied in first, which only an empty folder takes; the old folder stays as it was.
func (s *Service) SetDataDir(dir string, withData bool) error {
	if s.dataFixed {
		return errDataFixed
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if dir == filepath.Clean(s.dataDir) {
		return fmt.Errorf("%s is the data folder already", dir)
	}
	if withData {
		if err := copyData(s.dataDir, dir); err != nil {
			return fmt.Errorf("copying the data to %s: %w", dir, err)
		}
	}
	if err := s.store.SetDataDir(dir); err != nil {
		return err
	}
	if err := s.relaunch(); err != nil {
		return fmt.Errorf("saved the data folder; quit and open agentos to use it: %w", err)
	}
	return nil
}

// isEmpty says whether dir holds nothing but hidden files, such as the .DS_Store Finder leaves.
func isEmpty(dir string) (bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if !strings.HasPrefix(e.Name(), ".") {
			return false, nil
		}
	}
	return true, nil
}

// copyData copies each project's data from one data folder to an empty one. On failure it removes
// what it copied, so the folder is empty again.
func copyData(from, to string) (err error) {
	empty, err := isEmpty(to)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("%s is not empty", to)
	}
	projects, err := os.ReadDir(from)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var copied []string
	defer func() {
		if err != nil {
			for _, dir := range copied {
				os.RemoveAll(dir)
			}
		}
	}()
	for _, p := range projects {
		if !p.IsDir() || strings.HasPrefix(p.Name(), ".") {
			continue
		}
		dst := filepath.Join(to, p.Name())
		copied = append(copied, dst)
		for _, name := range dataNames {
			src := filepath.Join(from, p.Name(), name)
			if _, err := os.Lstat(src); errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err := copyTree(src, filepath.Join(dst, name)); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyTree copies a file or a folder, keeping each file's permissions.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dst, strings.TrimPrefix(path, src))
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if !d.Type().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
