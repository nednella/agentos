# AGENTS.md

agentos: one desktop window for every coding-agent session. It shows which session needs the owner,
the project's GitHub issues as a queue, notes, and each agent's real terminal. Agents keep running
in a hidden tmux when the window closes.

## Hard rules

- Tests and manual checks run in an isolated setup: always set `AGENTOS_TMUX_SOCKET`,
  `AGENTOS_STATE_DIR`, `AGENTOS_DEV_DATA_DIR` and `AGENTOS_CONFIG` to throwaway values. The
  default socket holds the owner's live sessions; a test on it has killed them twice.
- Never install the app to `/Applications` or `~/Applications`, never run
  `make desktop-install`, `install.sh` or `agentos update` against the real home, unless the owner asks.
  Live sessions' hooks call the installed `agentos`, a link into the installed bundle.
- Stop only the processes you started, by their PID. Never `pkill`, `killall` or kill by a
  name pattern.
- Clean-up code deletes worktrees and branches. Any change to it ships with tests against
  throwaway git repos, never a real one.
- Never create a GitHub issue without the owner approving that specific issue.

## How it works

**One binary.** `desktop/` builds a Wails v2 app whose binary is also the `agentos` command:
`~/.local/bin/agentos` links to `agentos.app/Contents/MacOS/agentos`, so the app, the command and the
hooks are always one version. Typed at a terminal with no command, it opens the app; started with no
terminal (Finder, `open`, the dock) it is the app.

**Sessions live in tmux.** Each agent runs in a tmux server on a private socket (`agentos` by default,
`AGENTOS_TMUX_SOCKET` to change). A session's id is `<project key>/<8 random characters, a-z and 2-7>`. Closing the window
leaves the agents running; the app finds them again when it starts. The front end shows a session's
terminal by attaching to its tmux pane over `terminal.Service`.

**Hooks report state.** A Claude session starts with `--settings` holding hooks that run
`agentos hook <Event>`. The hook folds the event into the session's state file under a lock, then tells the
app over the bus socket, `AGENTOS_SOCKET` (`~/.local/state/agentos/agentos.sock`); a missed send costs only a
delay. From the events the app derives the session's
state: `waiting` (a permission prompt or question), `working`, `idle` (the user's move) or `ended` (the
process is gone; the row stays until dismissed, at most 7 days). Each hook also reports the agent's working
folder, from which the app reads the branch it works on. Any other `agent_command` runs plain, with no
hooks and no states, and so does `claude` when `agentos` is not on PATH (the app logs it).

**What a session is started with.** Environment: `AGENTOS_SESSION`, `AGENTOS_SOCKET`, `AGENTOS_ISSUE` for an
issue session, and `TMPDIR` and `CLAUDE_CODE_TMPDIR` set to its own temp folder, removed when the row is
dismissed. Claude also gets `permissions.additionalDirectories` for the app's folders, `--model` and
`--effort` when something picks them (see `docs/config.md`), and `--append-system-prompt` with the texts in
`internal/prompts`: `session.md` (where it runs and the `agentos` commands it may use), `browser-session.md`
when the project has a browser, and `setup-brief.md` for the session that sets a project up.

**The command talks to the app.** Most `agentos` commands go to
the running app over the control socket (`control.sock`, beside the bus socket); each service that the command reaches registers a `Commands`
struct on the `desktop/control` router in `desktop/internal/app`. The project is `AGENTOS_PROJECT` (the
shell session), else the asking session's, else the current one.

**The front end talks to Go through Wails.** Each `desktop/<pkg>` binds one `Service`; React calls
`window.go.<pkg>.Service.<Method>(...)`, which returns a Promise, and listens with
`window.runtime.EventsOn(name, handler)`. `desktop/frontend/src/api.ts` wraps every call and `types.ts`
mirrors the Go structs; the Go side is the reference. A failing method rejects with a short sentence for the
user; empty lists are `[]`, never `null`. Lists of one project arrive as `{project, items}` events: drop one
whose project is not the current one. `window.go` is missing under plain `vite dev`, and the front end runs
on `mock.ts`. With `AGENTOS_HTTP=127.0.0.1:PORT` the binary serves the real front end in a browser, with a
shim that turns calls into `POST /__call/<pkg>.Service/<Method>` (header `X-Agentos: 1`) and events into the
stream `/__events`; it answers only its own listen address.

**The queue.** `issues.Service` lists the current project's open issues through `gh`, grouped by the
project's `queue_sections`. Starting an issue opens a session titled `#<n> <title>`, starts the agent with the
model and effort the action and the issue's `model:`/`effort:` labels pick, then types the action's command
once the agent is ready and sends it, unless `session_prompt_send` is `manual`. An issue with a live session
gets that session back. A project with no `queue_sections` gets an offer to set it up: `SetUpProject` fills a
basic block (`project.SetUp`) and starts a session briefed to write the repository side; `DismissSetup` saves
`setup_dismissed`. The palette's *Set up project* runs `SetUpProject` for any project.

**Pull requests.** A session with an issue works on the issue's branch: the one it reported with
`agentos track`, else the one checked out in its working folder, else the project's `session_branch_fallback`.
A session without one collects every branch it was seen on (other repos, a detached head and the default
branch are ignored), remembered in `prs.json`; its PR is the one on its newest branch that has one. The app
fetches every tracked PR every 5 minutes. Between those, each project with a session on a branch has a
watcher: `gh webhook forward` by default, falling back to polling the PR list every `pr_poll_interval` with
`If-None-Match`. A session's Stop hook also looks its branch's PR up at once, so a PR opened during a turn
shows when the turn ends. A new comment or review types `pr_review_command` into the session; failing checks
on a new commit type `pr_checks_command`. A session waiting on the owner gets the command once it stops
waiting. An ended issue session gets a new session for its issue that resumes the old conversation
(`claude --resume`); an ended session with no issue gets nothing.

**Clean-up.** When a session's PR merges (`auto` by default) or closes (`manual` by default), the app cleans
up: safety checks first (no changes in the worktree, no commits missing from origin or, for a merged PR whose
branch is gone, from the merged commit), then `session_cleanup_command` once per branch through `sh -c` in the
project folder, then the session's temp files, evidence, browser tab and row. A failed check or command leaves
the row `blocked` with the reason; the user can force it. A PR first seen already merged or closed sets
`cleanup: 'ask'`. Every run lands in `cleanups.json`.

**Browser.** One Chromium browser per project, on its own profile and a fixed DevTools port saved in
`browser-port`; one window per session. A Claude session gets the `chrome-devtools-mcp` server as `browser`,
with `new_page` and `close_page` disallowed, and drives its own pages, which carry the label
`[<project> · <n> <session title>]` in their title. `agentos browser open|tab|screenshot|close` make and act on
those pages. A dialog nobody answers for 5 seconds is dismissed, because it blocks every later call.

**Everything else.** Notes (`notes`, filed as issues with `## Description`), evidence an agent files with
`agentos show` or a screenshot (`evidence`), the interruption tally (`stats`), a weekly digest of what changed
in the project's dependencies, run by a locked-down `claude` with `internal/prompts/digest.md` (`digest`),
self-update from GitHub releases (`update`), holding off idle sleep while a session works (`awake`) and the
settings panel (`settings`). The config file and the command line are in `docs/config.md`.

**Storage.** Notes, note images, evidence, stats and the digest live in `data_dir` (default
`~/.local/share/agentos`), per project. PR tracking, the clean-up log, the browser profile and session temp
folders are always under `~/.local/share/agentos/<key>/`. State files, sockets, the tmux config and the last
project are in `~/.local/state/agentos`. Every file is written to a temp file and renamed into place.

## Code layout

| Where                      | What                                                                                                                                                                                                  |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `desktop/`                 | the app and the binary; one package per service: `sessions`, `terminal`, `issues`, `projects`, `notes`, `stats`, `evidence`, `browser`, `digest`, `update`, `settings`, `awake`, `control`, `devhttp` |
| `desktop/internal/app`     | wires the services and the control router                                                                                                                                                             |
| `desktop/internal/apptest` | the test harness: builds the whole app on private tmux sockets, temp git repos and fake `gh` and `claude`                                                                                             |
| `desktop/internal/run`     | runs commands; injected so tests never call the real `gh`, `git` or `claude`                                                                                                                          |
| `desktop/frontend/`        | React 18, TypeScript strict, Tailwind, xterm.js                                                                                                                                                       |
| `cli/`                     | the `agentos` commands agents and hooks call                                                                                                                                                          |
| `internal/`                | shared packages: `session`, `project` (config), `bus`, `term`, `control`, `agent`, `update`, `prompts`                                                                                                |
| `internal/prompts/*.md`    | every text agentos gives an agent                                                                                                                                                                     |
| `install.sh`               | installs a release; release-please cuts them from conventional commits                                                                                                                                |

## Commands

| What                                     | Command                                                                                                  |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------- |
| Go tests, vet                            | `go test -race ./...` · `go vet ./...`                                                                   |
| Front end typecheck + build              | `cd desktop/frontend && npm run build`                                                                   |
| Build the app                            | `make desktop-app` → `bin/agentos-dev.app` (`VERSION=v1.2.3` stamps a version)                           |
| Build a release zip                      | `make release` → `bin/agentos-darwin-arm64.zip` and its `.sha256`                                        |
| Open the app                             | `open bin/agentos-dev.app`                                                                               |
| Run the real app in a browser for checks | `AGENTOS_HTTP=127.0.0.1:34777 bin/agentos-dev.app/Contents/MacOS/agentos` (with the isolation env above) |
| Drive that browser                       | `node scripts/cdp.mjs http://127.0.0.1:34777/ 1512 945 '<steps>'`                                        |

The window itself cannot be screenshotted by a session (macOS blocks it); `docs/testing.md`
has the full method and what only the owner can check by eye.

## Code

- Go: plain, small packages, injected runners (`gh`, `git`, `claude`) so tests never
  call the real thing. Each service has a `Service` struct for the front end and, when the
  CLI reaches it, a `Commands` struct. Tests sit beside the code and build the app with `apptest`.
  Comments only for a non-obvious why.
- Agent-facing texts live in `internal/prompts/*.md`, never inline in Go.
- Front end: one component per file, props typed inline as `{Name}Props`, early returns,
  every colour a token in `tokens.css`, one shared action registry drives shortcuts,
  palette and command line. Keep `types.ts` and `api.ts` in step with the Go `Service` structs, and
  `mock.ts` acting like the real thing. No new dependencies without a reason in the PR.
- Change `docs/config.md` in the same PR as a config key or command.
- Minimal, clear UI: flat surfaces, colour only for meaning (state, issue type, focus).

## Workflow

Issues on `nednella/agentos` are the queue. One issue → branch `issue-<n>` in its own
worktree → one draft PR → the owner merges → the app cleans up. Conventional commits, one
logical change each. Manual issues follow `.github/ISSUE_TEMPLATE/issue.md`: a
`## Description` written by a human, never edited by a session; a session appends its
findings below it under `## Agent Review`.
