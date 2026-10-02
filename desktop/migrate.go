package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/nednella/agentos/internal/atomicfile"
)

// migrateLayout moves files saved by older versions, grouped by kind, into the
// folder of their project. Nothing is copied and left, and an existing file in
// the new place is never overwritten: the old one stays, and the log says so.
// Folders the moves empty are removed.
func migrateLayout(syncDir, localDir string, logf func(format string, args ...any)) {
	m := mover{logf: logf}

	// Notes, stats and digests live in the data dir.
	for _, f := range m.files(filepath.Join(syncDir, "notes"), ".json") {
		key := strings.TrimSuffix(filepath.Base(f), ".json")
		m.file(f, filepath.Join(syncDir, key, "notes.json"), func(b []byte) []byte {
			return bytes.ReplaceAll(b, []byte("/media/notes-media/"+key+"/"), []byte("/media/"+key+"/notes-media/"))
		})
	}
	for _, d := range m.dirs(filepath.Join(syncDir, "notes-media")) {
		m.tree(d, filepath.Join(syncDir, filepath.Base(d), "notes-media"))
	}
	for _, f := range m.files(filepath.Join(syncDir, "stats"), ".jsonl") {
		m.file(f, filepath.Join(syncDir, strings.TrimSuffix(filepath.Base(f), ".jsonl"), "stats.jsonl"), nil)
	}
	for _, f := range m.files(filepath.Join(syncDir, "digest"), ".json") {
		m.file(f, filepath.Join(syncDir, strings.TrimSuffix(filepath.Base(f), ".json"), "digest.json"), nil)
	}
	m.tidy(syncDir, "notes", "notes-media", "stats", "digest")

	// Evidence, PR tracking, the clean-up log and the browser profile are local.
	for _, d := range m.dirs(filepath.Join(localDir, "evidence")) {
		key := filepath.Base(d)
		m.tree(d, filepath.Join(localDir, key, "evidence"))
		for _, index := range m.glob(filepath.Join(localDir, key, "evidence", "*", "index.json")) {
			m.rewrite(index, "/media/evidence/"+key+"/", "/media/"+key+"/evidence/")
		}
	}
	for _, f := range m.files(filepath.Join(localDir, "prs"), ".json") {
		if strings.HasSuffix(f, ".live.json") {
			continue
		}
		key := strings.TrimSuffix(filepath.Base(f), ".json")
		m.prs(localDir, key)
	}
	for _, f := range m.files(filepath.Join(localDir, "cleanups"), ".jsonl") {
		m.cleanups(f, filepath.Join(localDir, strings.TrimSuffix(filepath.Base(f), ".jsonl"), "cleanups.json"))
	}
	for _, d := range m.dirs(filepath.Join(localDir, "browser")) {
		if browserRunning(d) {
			logf("not moving the browser profile %s: a browser is running on it", d)
			continue
		}
		m.tree(d, filepath.Join(localDir, filepath.Base(d), "browser"))
	}
	m.tidy(localDir, "evidence", "prs", "cleanups", "browser")
}

type mover struct {
	logf func(format string, args ...any)
}

func (m mover) files(dir, ext string) []string {
	var out []string
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func (m mover) dirs(dir string) []string {
	var out []string
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.IsDir() {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out
}

func (m mover) glob(pattern string) []string {
	out, _ := filepath.Glob(pattern)
	return out
}

func present(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// file moves src to dst, passing the content through change if given.
func (m mover) file(src, dst string, change func([]byte) []byte) {
	if src == dst {
		return
	}
	if present(dst) {
		m.logf("kept %s: %s already exists", src, dst)
		return
	}
	var err error
	if change == nil {
		if err = os.MkdirAll(filepath.Dir(dst), 0o700); err == nil {
			err = os.Rename(src, dst)
		}
	} else {
		var data []byte
		if data, err = os.ReadFile(src); err == nil {
			if err = atomicfile.Write(dst, change(data), 0o600); err == nil {
				err = os.Remove(src)
			}
		}
	}
	if err != nil {
		m.logf("could not move %s to %s: %v", src, dst, err)
		return
	}
	m.logf("moved %s to %s", src, dst)
}

// tree moves a folder; where the destination exists, file by file.
func (m mover) tree(src, dst string) {
	if src == dst {
		return
	}
	if !present(dst) {
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err == nil {
			err = os.Rename(src, dst)
			if err == nil {
				m.logf("moved %s to %s", src, dst)
				return
			}
			m.logf("could not move %s to %s: %v", src, dst, err)
		}
		return
	}
	entries, _ := os.ReadDir(src)
	for _, e := range entries {
		if e.IsDir() {
			m.tree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
		} else {
			m.file(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()), nil)
		}
	}
	_ = os.Remove(src)
}

// rewrite replaces text in a file, if it holds it.
func (m mover) rewrite(path, old, new string) {
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Contains(data, []byte(old)) {
		return
	}
	if err := atomicfile.Write(path, bytes.ReplaceAll(data, []byte(old), []byte(new)), 0o600); err != nil {
		m.logf("could not update %s: %v", path, err)
	}
}

// prs merges the old acknowledgement and live files of a project into prs.json.
func (m mover) prs(localDir, key string) {
	acks, live := filepath.Join(localDir, "prs", key+".json"), filepath.Join(localDir, "prs", key+".live.json")
	dst := filepath.Join(localDir, key, "prs.json")
	if present(dst) {
		m.logf("kept %s: %s already exists", acks, dst)
		return
	}
	f := prsFile{Acks: map[string]ack{}, Live: map[string]bool{}}
	if data, err := os.ReadFile(acks); err == nil {
		_ = json.Unmarshal(data, &f.Acks)
	}
	if data, err := os.ReadFile(live); err == nil {
		_ = json.Unmarshal(data, &f.Live)
	}
	data, err := json.Marshal(f)
	if err == nil {
		err = atomicfile.Write(dst, data, 0o600)
	}
	if err != nil {
		m.logf("could not write %s: %v", dst, err)
		return
	}
	_ = os.Remove(acks)
	_ = os.Remove(live)
	m.logf("moved %s and %s to %s", acks, live, dst)
}

// cleanups turns the old line-per-entry log into the JSON list.
func (m mover) cleanups(src, dst string) {
	if present(dst) {
		m.logf("kept %s: %s already exists", src, dst)
		return
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return
	}
	log := []Cleanup{}
	for _, line := range bytes.Split(data, []byte("\n")) {
		var c Cleanup
		if json.Unmarshal(line, &c) == nil {
			log = append(log, c)
		}
	}
	out, err := json.Marshal(log)
	if err == nil {
		err = atomicfile.Write(dst, out, 0o600)
	}
	if err != nil {
		m.logf("could not write %s: %v", dst, err)
		return
	}
	_ = os.Remove(src)
	m.logf("moved %s to %s", src, dst)
}

// tidy removes the old type folders that the moves emptied.
func (m mover) tidy(root string, names ...string) {
	for _, name := range names {
		if err := os.Remove(filepath.Join(root, name)); err == nil {
			m.logf("removed the empty folder %s", filepath.Join(root, name))
		}
	}
}

// browserRunning says whether a browser runs on the profile, by its lock.
func browserRunning(profile string) bool {
	_, alive := lockedPID(profile)
	return alive
}
