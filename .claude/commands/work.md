---
description: Work issue <n> end to end: branch in a worktree, build, verify, draft PR
argument-hint: <issue number>
---

Work issue #$ARGUMENTS on `nednella/agentos`, start to finish, without waiting for
the owner unless a decision is genuinely theirs. Read `AGENTS.md` first; its hard rules apply.

## 1. Understand

- `gh issue view $ARGUMENTS` and read every comment. The issue body is the owner's;
  never edit it.
- Read the code the issue touches before planning. `AGENTS.md` says how the parts fit
  together; `docs/testing.md` says how to verify.
- If the issue allows more than one reasonable reading, pick the simplest and say so in
  your Agent Review (step 5); do not stop to ask.
- If the task is clearly beyond your model, say so in your Agent Review and stop rather than grind.

## 2. Branch

```
git fetch origin
git worktree add trees/issue-$ARGUMENTS -b issue-$ARGUMENTS origin/main
cd trees/issue-$ARGUMENTS
```

Branch from `origin/main`, never from local `main`, so the PR holds only your change.
Work only inside that worktree. Never touch the tree the owner is sitting in.

## 3. Build

- Smallest change that closes the issue. Match the code around it. No new
  dependencies without a reason in the PR.
- Change `docs/config.md` in the same change when a config key or command moves.
- Tests beside the code they cover.

## 4. Verify

- `go vet ./...` and `go test -race ./...` for Go; `cd desktop/frontend && npm run build`
  for the front end.
- For anything visible, run the real app in browser mode and look at it, as
  `docs/testing.md` describes. Always set `AGENTOS_TMUX_SOCKET`, `AGENTOS_STATE_DIR`,
  `AGENTOS_DEV_DATA_DIR` and `AGENTOS_CONFIG` to throwaway values: the defaults hold the owner's
  live sessions.
- Never claim something works that you did not run.

## 5. Deliver

- Commits: conventional, scoped by layer (`internal`, `cli`, `desktop`, `ui`, `docs`,
  `build`), imperative subject with no "the" or "a", one logical change each, a short
  body saying why.
- Post a comment on the issue headed `## Agent Review`, with
  `gh issue comment $ARGUMENTS --body-file -`: what you found, what you changed and why,
  what you verified and how, anything you chose between. Short, plain sentences.
- `git push -u origin issue-$ARGUMENTS`, then
  `gh pr create --draft --assignee @me --title "<subject>" --body "<Description heading, then the summary; Closes #$ARGUMENTS>"`.
- Never `gh pr merge`, never `gh pr ready`, never request reviewers. The owner merges.
- Report in a few lines: the PR link, what you verified, what is left to the owner.
