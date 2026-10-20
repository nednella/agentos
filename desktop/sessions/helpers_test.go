package sessions_test

import (
	"os"
	"testing"

	"github.com/nednella/agentos/desktop/internal/apptest"
)

func TestMain(m *testing.M) { os.Exit(apptest.Main(m)) }

var (
	eventually = apptest.Eventually
	exists     = apptest.Exists
	prJSON     = apptest.PRJSON
)

// removeWorktreeAndBranch is the clean-up command most tests run: it leaves the project folder alone.
const removeWorktreeAndBranch = "git worktree remove {force} {worktree} && git branch -D {branch}"

func newHarness(t *testing.T) *apptest.Harness {
	return apptest.NewWith(t, apptest.Options{CleanupCommand: removeWorktreeAndBranch})
}

// newClaudeHarness runs the sessions on the Claude stand-in: for a test that fakes Claude's hooks with a transcript,
// since only Claude's provider reads one.
func newClaudeHarness(t *testing.T) *apptest.Harness {
	apptest.ClaudeOnPath(t)
	return apptest.NewWith(t, apptest.Options{Agent: "claude", CleanupCommand: removeWorktreeAndBranch})
}
