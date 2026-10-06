package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var repoName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

const readmeTemplate = "<div align=\"center\">\n  <h3><b>%s</b></h3>\n</div>\n"

// NewProject asks where to put a new folder called name, starts a git repository in it with a
// README, creates the private GitHub repo as its origin, and makes it the current project.
func (s *Service) NewProject(name string) (Snapshot, error) {
	name = strings.TrimSpace(name)
	if !repoName.MatchString(name) || strings.Trim(name, ".") == "" {
		return Snapshot{}, fmt.Errorf("%q is not a project name: use letters, digits, '.', '-' and '_'", name)
	}
	parent, err := s.pickDir()
	if err != nil {
		return Snapshot{}, fmt.Errorf("choosing a folder: %w", err)
	}
	if parent == "" {
		return s.Snapshot(), nil
	}
	dir := filepath.Join(parent, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		return Snapshot{}, fmt.Errorf("creating the folder: %w", err)
	}
	if err := s.initRepo(dir, name); err != nil {
		_ = os.RemoveAll(dir)
		return Snapshot{}, err
	}
	if _, err := s.run(s.ctx(), dir, "gh", "repo", "create", name, "--private", "--source", ".", "--push"); err != nil {
		return Snapshot{}, fmt.Errorf("%s is ready but its GitHub repo is not: %w", dir, err)
	}
	return s.AddProjectDir(dir)
}

func (s *Service) initRepo(dir, name string) error {
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(fmt.Sprintf(readmeTemplate, name)), 0o644); err != nil {
		return err
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"add", "README.md"},
		{"commit", "-m", "chore: initial commit"},
	} {
		if _, err := s.run(s.ctx(), dir, "git", args...); err != nil {
			return err
		}
	}
	return nil
}
