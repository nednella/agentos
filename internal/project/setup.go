package project

import (
	"cmp"

	"github.com/nednella/agentos/internal/prompts"
)

// SetupCleanupCommand is the clean-up command that SetUp writes. {worktree} is '' for a branch with no worktree.
const SetupCleanupCommand = "{ [ -z {worktree} ] || git worktree remove {force} {worktree}; } && git branch -D {branch}"

// NeedsSetup says whether the project lacks the queue sections that setting it up writes.
func (p Project) NeedsSetup() bool { return len(p.QueueSections) == 0 }

// SetUp fills the keys of a basic agentos block that the project leaves empty: an issue's branch, the clean-up
// command, the commands for PR reviews, failing checks and conflicts, and one Inbox section of every issue whose action is /work.
func (p *Project) SetUp() {
	p.Branch = cmp.Or(p.Branch, "issue-{n}")
	p.CleanupCommand = cmp.Or(p.CleanupCommand, SetupCleanupCommand)
	p.OnReview = cmp.Or(p.OnReview, prompts.SetupReview())
	p.OnChecks = cmp.Or(p.OnChecks, prompts.SetupChecks())
	p.OnConflict = cmp.Or(p.OnConflict, prompts.SetupConflict())
	if p.NeedsSetup() {
		p.QueueSections = []Section{{Name: "Inbox", Labels: []string{"*"}, Actions: []Action{{Name: "Work", Command: "/work {n}"}}}}
	}
}
