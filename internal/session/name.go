package session

import (
	"fmt"
	"strconv"
	"strings"
)

// Name is the tmux session name: "<project key>/<n>". The title shown to the
// user lives in a tmux option, so renaming never changes the name the agent's
// hooks report under. A name with Shell set is one of the project's shells,
// "<project key>/shell" for the first and "<project key>/shell-<k>" for the
// rest; a shell is not an agent and never reports.
type Name struct {
	Project string
	N       int
	Shell   int
}

const shellSuffix = "shell"

func (n Name) String() string {
	switch {
	case n.Shell == 1:
		return n.Project + "/" + shellSuffix
	case n.Shell > 1:
		return fmt.Sprintf("%s/%s-%d", n.Project, shellSuffix, n.Shell)
	}
	return fmt.Sprintf("%s/%d", n.Project, n.N)
}

// IsShell says whether the name is one of a project's shells rather than an agent.
func (n Name) IsShell() bool { return n.Shell > 0 }

// ParseName decodes a tmux session name. It fails for sessions agentos did not create.
func ParseName(s string) (Name, error) {
	i := strings.IndexByte(s, '/')
	if i <= 0 || i != strings.LastIndexByte(s, '/') || strings.Contains(s[:i], "..") {
		return Name{}, fmt.Errorf("session name %q is not <project>/<n>", s)
	}
	if s[i+1:] == shellSuffix {
		return Name{Project: s[:i], Shell: 1}, nil
	}
	if rest, ok := strings.CutPrefix(s[i+1:], shellSuffix+"-"); ok {
		k, err := strconv.Atoi(rest)
		if err != nil || k < 2 || strconv.Itoa(k) != rest {
			return Name{}, fmt.Errorf("session name %q has no valid shell number", s)
		}
		return Name{Project: s[:i], Shell: k}, nil
	}
	n, err := strconv.Atoi(s[i+1:])
	if err != nil || n < 1 {
		return Name{}, fmt.Errorf("session name %q has no positive number", s)
	}
	return Name{Project: s[:i], N: n}, nil
}

// ProjectKey is the project key in a session id, or "project" when the id does not parse.
func ProjectKey(id string) string {
	name, err := ParseName(id)
	if err != nil {
		return "project"
	}
	return name.Project
}

// NextN returns the lowest number not in use.
func NextN(used []int) int {
	taken := make(map[int]bool, len(used))
	for _, n := range used {
		taken[n] = true
	}
	n := 1
	for taken[n] {
		n++
	}
	return n
}
