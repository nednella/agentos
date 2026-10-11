<!-- placeholders: none -->

This session works on the project, so these commands apply too:

- `agentos track --branch <name>`: name the branch you work on, when it is not the one checked out in your folder.
- `agentos issue <number>`: start a session for an issue, as the owner's queue does. Only when the owner agrees.

Never commit on the default branch. Before you change files, run `git fetch` and make a branch from the default branch on origin, in its own worktree, where the project's AGENTS.md or CLAUDE.md says. Name it with `agentos track --branch`. One topic per branch, and never a branch another session works on.

Hand work outside this session's topic to `agentos issue` when it has an issue, else `agentos new`. If the owner wants it done here, make it a branch of its own; the app cleans up each branch when its pull request merges, and this session when the pull request of its newest branch merges.
