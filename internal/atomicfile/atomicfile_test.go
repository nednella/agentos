package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b.txt")
	for _, body := range []string{"one", "two"} {
		if err := Write(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(path); string(got) != body {
			t.Errorf("file = %q, want %q", got, body)
		}
	}
	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Errorf("%d files left, want 1", len(entries))
	}
}

func TestWriteKeepsALink(t *testing.T) {
	target := filepath.Join(t.TempDir(), "dotfiles", "config.yaml")
	if err := Write(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := Write(link, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("the link was replaced by a file: %v %v", info.Mode(), err)
	}
	if got, _ := os.ReadFile(target); string(got) != "new" {
		t.Errorf("target = %q, want %q", got, "new")
	}
	if entries, _ := os.ReadDir(filepath.Dir(link)); len(entries) != 1 {
		t.Errorf("%d files left beside the link, want 1", len(entries))
	}
}
