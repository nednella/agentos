# Session row status

## What exists

A task's row and its panel header show one line of chips: the PR number and state, checks, new comments, conflict, "ready", unpushed commits, uncommitted files (after a merge: commits not in main) and commits behind the base. Hovering the line opens a card with the PR's state pill and title, `base ← branch` with an ahead/behind bar, the problem chips, the Checks, Review and Comments tiles, and the failing checks by name. A chat has no line. A task with a branch and no PR shows "no PR" and the branch chips.

## How a user reaches it

Every task row in the sessions panel; the header above the terminal of the selected task. Hover the line for the card; click the PR number to open the PR.

## How to drive it

Make a throwaway repo with a bare origin and worktrees on `issue-<n>` branches, some pushed, some not, some with extra commits or files. Put a fake `gh` first on PATH that answers `pr list --head <branch>` with canned JSON (title, `statusCheckRollup` with names, `reviews` with `author.login`). The app replaces PATH with the login shell's when any agent program is missing, so stub `agy` and the other agent programs in the same folder. Start tasks with `sessions.Service/NewSession` and give each its branch with `agentos track --branch` under `AGENTOS_SESSION` and `AGENTOS_SOCKET`. A `PostToolUse` hook (`agentos hook PostToolUse` with `{"cwd": <worktree>}` on stdin) reads the git counts again without a PR fetch.

## What goes wrong

- Escape closes the card only when the terminal does not have focus: the terminal keeps Escape for the agent.
- In browser mode a click on the PR opens a tab in the same browser window, and `agentos browser screenshot` then shoots that tab.
- Comments that a PR had at first sight are not new, so the comment chip shows only for later ones; opening the PR acknowledges them.
- "Behind" uses the local `origin/<base>`, so it is only as fresh as the last fetch in the repo.
