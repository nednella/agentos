<!-- placeholders: none -->

agentos setup: what a project's repository needs so that `/work <n>` runs with no help.
The app writes the project's block in the agentos config when the owner chooses Set up; this covers the repository.
Ask the owner before you write each file.

1. Issue template, .github/ISSUE_TEMPLATE/issue.md
   One template, with only a `## Description` heading. A note filed as an issue gets the same body: `## Description`,
   then the note. The Description is the owner's; a session never edits it and appends its findings below it.

     ---
     name: Issue
     about: Describe what you want or what is wrong; a session appends its findings below
     ---

     ## Description

2. Command, .claude/commands/work.md
   For `/work <n>`: read issue n and its comments, make a worktree on branch `issue-<n>` from the default branch on
   origin, build the smallest change, run the tests and check the change, append `## Agent Review` below the
   Description (what it found, changed and verified), push, and open a draft PR that closes the issue. It never merges,
   marks the PR ready or requests reviewers.
   It starts with front matter: `description:` and `argument-hint: <issue number>`. $ARGUMENTS is the number.

3. The project's CLAUDE.md
   `/work` runs alone only when CLAUDE.md says:
   - how to make a worktree for an issue: where it goes, from which branch, and any set-up it needs (dependencies,
     env files, a build)
   - how to run the tests, the linters and the build, as exact commands
   - how to check a change by hand, when tests are not enough
   - the commit message style, and any hard rules, such as what a session must never touch
