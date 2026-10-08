<!-- placeholders: none -->

agentos setup: what a project needs so that `/work <n>` (and `/plan <n>`, when the project plans first) runs with no help.
Make each part in the project's repository, except the config block, which goes in the owner's agentos config.

1. Issue template, .github/ISSUE_TEMPLATE/issue.md
   One template, with only a `## Description` heading. A note filed as an issue gets the same body: `## Description`,
   then the note. The Description is the owner's; a session never edits it and appends its findings below it.

     ---
     name: Issue
     about: Describe what you want or what is wrong; a session appends its findings below
     ---

     ## Description

2. Commands, .claude/commands/
   work.md, for `/work <n>`: read issue n and its comments, make a worktree on branch `issue-<n>` from the default
   branch on origin, build the smallest change, run the tests and check the change, append `## Agent Review` below the
   Description (what it found, changed and verified), push, and open a draft PR that closes the issue. It never merges,
   marks the PR ready or requests reviewers.
   plan.md, only when the project plans first, for `/plan <n>`: read issue n and the code, append a `## Plan` below the
   Description, then add the `ready` label and the `model:<x>` and `effort:<y>` labels that `/work` should run with.
   Both start with front matter: `description:` and `argument-hint: <issue number>`. $ARGUMENTS is the number.

3. Labels on the GitHub repository
   ready        planned, or clear enough to work
   needs-human  waits for the owner
   idea         not yet a task
   model:<x>    the model a session for the issue runs, such as model:opus; it beats the action's model
   effort:<y>   its effort: low, medium, high, xhigh or max; it beats the action's effort
   Make the ones that are missing with `gh label create <name>`. agentos reads ready, needs-human and idea only through
   the queue_sections that name them.

4. The project block in the agentos config (~/.config/agentos/config.yaml, or the path in AGENTOS_CONFIG)
   `agentos project add` in the project folder adds the block with its name and directory. Show the owner the rest
   and let them add it; the app reads the config only when it starts:

     - name: storefront
       directory: /Users/me/code/storefront
       session_branch_fallback: "issue-{n}"
       session_cleanup_command: "git worktree remove {force} {worktree} && git branch -D {branch}"
       queue_sections:
         - name: Ready
           labels: [ready]
           actions:
             - {name: Work, command: "/work {n}"}
         - name: Needs human
           labels: [needs-human]
         - name: Ideas
           labels: [idea]
         - name: Inbox
           labels: []
           actions:
             - {name: Plan, command: "/plan {n}", model: opus, effort: high}
             - {name: Work, command: "/work {n}"}

   session_branch_fallback  the branch of an issue's work when the session reports none; {n} is the issue number
   session_cleanup_command  run in the project folder when the app cleans up a session, as after its PR merges, once
                            per branch; {branch}, {worktree}, {dir}, and {force}, which is --force when forced
   queue_sections           the groups of the queue, in order. An issue goes in the first section with one of its
                            labels; `labels: []` takes the issues with no labels; the rest go in Other. Each action
                            types its command, {n} the number and {title} the title, at its model and effort; the
                            first action is the default; a section with none gets Start, which types
                            "Work on issue #<n>: <title>". A project with no `/plan` leaves the Plan action out.

5. The project's CLAUDE.md
   `/work` runs alone only when CLAUDE.md says:
   - how to make a worktree for an issue: where it goes, from which branch, and any set-up it needs (dependencies,
     env files, a build)
   - how to run the tests, the linters and the build, as exact commands
   - how to check a change by hand, when tests are not enough
   - the commit message style, and any hard rules, such as what a session must never touch

Run `agentos project add` and `gh label create` only when the owner asks you to set up the project.
