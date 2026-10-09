<!-- placeholders: none -->

The owner chose Set up for this project in agentos. The app has filled in the project's agentos config; your job is
the repository side, so that `/work <n>` runs with no help. Check what the repository already has, then build what
is missing with the owner. Ask before you write each file. `/work` and the app's pull request tracking need the
repository on GitHub as `origin`; when it has none, tell the owner before anything else.

Every project agentos sets up gets the same files. Write the issue template and the command exactly as given below,
and AGENTS.md from its list. Each file you write carries the marker shown with it, so a reader knows agentos made it
and where agentos lives. When the repository already has one of these files, show the owner how it differs from what
is given here and ask which to keep; a file the owner keeps as it is gets no marker.

## 1. Issue template, `.github/ISSUE_TEMPLATE/issue.md`

One template, with only a `## Description` heading. A note filed as an issue gets the same body: `## Description`,
then the note. The issue body is the owner's; a session never edits it and posts its plans and reviews as comments.

```markdown
---
# Set up by agentos: https://github.com/nednella/agentos
name: Issue
about: Describe what you want or what is wrong; sessions post their plans and reviews as comments
---

## Description
```

## 2. Command, `.claude/commands/work.md`

For `/work <n>`. `$ARGUMENTS` is the issue number.

```markdown
---
# Set up by agentos: https://github.com/nednella/agentos
description: "Work issue <n> end to end: branch in a worktree, build, verify, draft PR"
argument-hint: <issue number>
---

Work issue #$ARGUMENTS start to finish, without waiting for the owner unless a decision is
genuinely theirs. Read `AGENTS.md` first; its hard rules apply.

## 1. Understand

- `gh issue view $ARGUMENTS --comments` and read every comment. The issue body is the owner's;
  never edit it.
- Read the code the issue touches before planning.
- If the issue allows more than one reasonable reading, pick the simplest and say so in your
  Agent Review (step 5); do not stop to ask.

## 2. Branch

Make a worktree on branch `issue-$ARGUMENTS` from the default branch on origin, where and with
the set-up `AGENTS.md` names. Work only inside it.

## 3. Build

- Smallest change that closes the issue. Match the code around it. No new dependencies without
  a reason in the PR.
- Tests beside the code they cover.

## 4. Verify

- Run the tests, linters and build that `AGENTS.md` names, and check the change by hand as it
  says.
- Never claim something works that you did not run.

## 5. Deliver

- Commit in the style `AGENTS.md` names, one logical change each.
- Post a comment on the issue headed `## Agent Review`, with
  `gh issue comment $ARGUMENTS --body-file -`: what you found, what you changed and why, what
  you verified and how, anything you chose between. Short, plain sentences.
- `git push -u origin issue-$ARGUMENTS`, then
  `gh pr create --draft --assignee @me --title "<subject>" --body "<summary; Closes #$ARGUMENTS>"`.
- Never `gh pr merge`, never `gh pr ready`, never request reviewers. The owner merges.
- Report in a few lines: the PR link, what you verified, what is left to the owner.
```

## 3. `AGENTS.md` at the repository root

It starts with the marker, then the repository's own text:

```markdown
<!-- Set up by agentos: https://github.com/nednella/agentos -->
```

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
