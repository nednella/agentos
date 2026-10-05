package digest

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

const (
	dependenciesMax = 150
	scanDepth       = 3
)

// dependencyName is what a package or module name may look like. The names go into a
// prompt, so a file in the repository cannot smuggle sentences in through one.
var dependencyName = regexp.MustCompile(`^[A-Za-z0-9@][A-Za-z0-9@/._~-]{0,99}$`)

// dependencies lists, sorted and without repeats, the names the project's package.json
// and go.mod files depend on. Folders that hold other people's code are skipped.
func dependencies(dir string) []string {
	seen := map[string]bool{}
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			depth := strings.Count(strings.TrimPrefix(path, dir), string(filepath.Separator))
			if depth >= scanDepth || (path != dir && slices.Contains([]string{"node_modules", "vendor", ".git", "trees"}, d.Name())) {
				return fs.SkipDir
			}
			return nil
		}
		switch d.Name() {
		case "package.json":
			for _, name := range packageNames(path) {
				seen[name] = true
			}
		case "go.mod":
			for _, name := range moduleNames(path) {
				seen[name] = true
			}
		}
		return nil
	})
	var out []string
	for name := range seen {
		if dependencyName.MatchString(name) {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out[:min(len(out), dependenciesMax)]
}

func packageNames(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var pkg struct {
		Dependencies    map[string]json.RawMessage `json:"dependencies"`
		DevDependencies map[string]json.RawMessage `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}
	var names []string
	for name := range pkg.Dependencies {
		names = append(names, name)
	}
	for name := range pkg.DevDependencies {
		names = append(names, name)
	}
	return names
}

func moduleNames(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var names []string
	inBlock := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "//")
		fields := strings.Fields(line)
		switch {
		case len(fields) == 2 && fields[0] == "require" && fields[1] == "(":
			inBlock = true
		case inBlock && len(fields) == 1 && fields[0] == ")":
			inBlock = false
		case inBlock && len(fields) >= 1:
			names = append(names, fields[0])
		case len(fields) >= 2 && fields[0] == "require":
			names = append(names, fields[1])
		}
	}
	return names
}
