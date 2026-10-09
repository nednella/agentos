<!-- placeholders: none -->

The owner chose Set up for this project in agentos. The app has filled in the project's agentos config; your job is
the repository side, so that `/work <n>` runs with no help. Check what the repository already has, then build what
is missing with the owner. Ask before you write each file. `/work` and the app's pull request tracking need the
repository on GitHub as `origin`; when it has none, tell the owner before anything else.

1. Issue template, .github/ISSUE_TEMPLATE/issue.md
   One template, with only a `## Description` heading. A note filed as an issue gets the same body: `## Description`,
   then the note. The issue body is the owner's; a session never edits it and posts its plans and reviews as comments.

     ---
     name: Issue
     about: Describe what you want or what is wrong; sessions post their plans and reviews as comments
     ---

     ## Description

2. Command, .claude/commands/work.md
   For `/work <n>`: read issue n and its comments, make a worktree on branch `issue-<n>` from the default branch on
   origin, build the smallest change, run the tests and check the change, post a comment on the issue headed
   `## Agent Review` (what it found, changed and verified), push, and open a draft PR that closes the issue. It never merges,
   marks the PR ready or requests reviewers.
   It starts with front matter: `description:` and `argument-hint: <issue number>`. $ARGUMENTS is the number.

3. AGENTS.md at the repository root
   `/work` runs alone only when AGENTS.md says:
   - where a worktree goes and the set-up it needs (dependencies, env files, a build); a worktree folder inside the
     repository goes in .gitignore
   - how to run the tests, the linters and the build, as exact commands
   - how to check a change by hand, when tests are not enough
   - the commit message style, and any hard rules, such as what a session must never touch
   - how a branch takes the base branch's changes, rebase or merge, and how it pushes after (for example
     `--force-with-lease` after a rebase); a session whose pull request conflicts with its base follows this
   Claude Code reads AGENTS.md only when the project has no CLAUDE.md. When it has one, ask the owner whether to move
   it into AGENTS.md.
