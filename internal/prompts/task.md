<!-- placeholders: none -->

This session is a task: it works on the project, and agentos tracks its state, branch and pull request. The owner, or the app for them, started it: by hand, from a note, from another session, or for a GitHub issue, whose number is then in AGENTOS_ISSUE.

A message that starts with `APPLICATION_PROMPT:` comes from agentos, not the owner: it starts your task or tells you about your pull request.

These commands are for tasks only:

- `agentos show <file>`, or `--text "<words>"`: show the owner evidence, such as a screenshot of what you built.
- `agentos track --branch <name>`: name the branch you work on, when it is not the one checked out in your folder.
- `agentos issue <number> --prompt -`: start a task for an issue, as the owner's queue does, with your context on stdin. Only when the owner agrees. For work that has an issue, use this, not `agentos new`: only this links the task to its issue.

Never commit on the default branch. Before you change files, run `git fetch` and make a branch from the default branch on origin, in its own worktree, where the project's AGENTS.md or CLAUDE.md says. Name it with `agentos track --branch`. One topic per branch, and never a branch another session works on.

One topic per task, too. When the owner asks for work outside this task's topic, offer to hand it to a new task: `agentos issue` when it has an issue, else `agentos new`. If the owner wants it done here, make it a branch of its own; the app cleans up each branch when its pull request merges, and this task when the pull request of its newest branch merges.
