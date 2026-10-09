package projects

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/nednella/agentos/desktop/sessions"
)

var repoName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

const readmeTemplate = "<div align=\"center\">\n  <h3><b>%s</b></h3>\n</div>\n"

// NewProject asks where to put a new folder called name, starts a git repository in it with a
// README, and makes it the current project. CreateRepo then makes its GitHub repo.
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
	return s.AddProjectDir(dir)
}

// CreateRepo creates the public GitHub repo of the project called name as its origin and pushes it.
func (s *Service) CreateRepo(name string) (Snapshot, error) {
	projects := s.sessions.Projects()
	i := slices.IndexFunc(projects, func(p sessions.Project) bool { return p.Name == name })
	if i < 0 {
		return Snapshot{}, fmt.Errorf("no project called %s", name)
	}
	dir := projects[i].Dir
	if _, err := s.run(s.ctx(), dir, "gh", "repo", "create", filepath.Base(dir), "--public", "--source", ".", "--push"); err != nil {
		return Snapshot{}, fmt.Errorf("%s has no GitHub repo: %w", name, err)
	}
	s.repos.FreshRepo(s.ctx(), dir)
	return s.Snapshot(), nil
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
