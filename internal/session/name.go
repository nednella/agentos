package session

import (
	"fmt"
	"strconv"
	"strings"
)

// Name is the tmux session name: "<project key>/<n>". The title shown to the
// user lives in a tmux option, so renaming never changes the name the agent's
// hooks report under.
type Name struct {
	Project string
	N       int
}

func (n Name) String() string { return fmt.Sprintf("%s/%d", n.Project, n.N) }

// ParseName decodes a tmux session name. It fails for sessions agentos did not create.
func ParseName(s string) (Name, error) {
	i := strings.LastIndexByte(s, '/')
	if i <= 0 {
		return Name{}, fmt.Errorf("session name %q is not <project>/<n>", s)
	}
	n, err := strconv.Atoi(s[i+1:])
	if err != nil || n < 1 {
		return Name{}, fmt.Errorf("session name %q has no positive number", s)
	}
	return Name{Project: s[:i], N: n}, nil
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
