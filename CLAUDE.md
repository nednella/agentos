# CLAUDE.md

agentos: one desktop window for every coding-agent session Ned runs. It shows which
session needs him, the project's GitHub issues as a queue, his notes, and each agent's
real terminal. Agents keep running in a hidden tmux when the window closes.

`desktop/` Go core (Wails v2), one package per service (`sessions`, `terminal`, `issues`,
`projects`, `notes`, `stats`, `evidence`, `browser`, `digest`, `control`, `devhttp`), wired in
`desktop/internal/app`; `desktop/internal/apptest` is the test harness, `desktop/internal/run`
runs commands · `desktop/frontend/` React 18, TypeScript strict, Tailwind, xterm.js · `cli/`
the `agentos` command the agents and hooks call · `internal/` shared packages (session,
project, bus, term, control socket, agent, `prompts`: the texts agents read, as Markdown).
`docs/contract.md` is the single description of how the front end and the core talk.
Go is the reference when they disagree.

## Hard rules

- Tests and manual checks run in an isolated setup: always set `AGENTOS_TMUX_SOCKET`,
  `AGENTOS_STATE_DIR`, `AGENTOS_DATA_DIR` and `AGENTOS_CONFIG` to throwaway values. The
  default socket holds Ned's live sessions; a test on it has killed them twice.
- Never install the app to `/Applications` or `~/Applications` and never run
  `make install` unless Ned asks. Live sessions' hooks call the installed `agentos`.
- Draft PRs only; never merge, mark ready or request reviewers. Ned's own Claude settings enforce this.
- Clean-up code deletes worktrees and branches. Any change to it ships with tests against
  throwaway git repos, never a real one.
- Never create a GitHub issue without Ned approving that specific issue.

## Commands

| What | Command |
|---|---|
| Go tests, vet | `go test -race ./...` · `go vet ./...` |
| Front end typecheck + build | `cd desktop/frontend && npm run build` |
| Build the app | `make desktop-app` → `bin/agentos.app` |
| Open the app | `open bin/agentos.app` |
| Run the real app in a browser for checks | `AGENTOS_HTTP=127.0.0.1:34777 bin/agentos.app/Contents/MacOS/agentos` (with the isolation env above) |
| Drive that browser | `node scripts/cdp.mjs http://127.0.0.1:34777/ 1512 945 '<steps>'` |

The window itself cannot be screenshotted by a session (macOS blocks it); `docs/testing.md`
has the full method and what only Ned can check by eye.

## Code

- Go: plain, small packages, injected runners (`gh`, `git`, `claude`) so tests never
  call the real thing. Each service has a `Service` struct for the front end and, when the
  CLI reaches it, a `Commands` struct registered on the `desktop/control` router in
  `desktop/internal/app`. Tests sit beside the code and build the app with `apptest`.
  Comments only for a non-obvious why.
- Agent-facing texts live in `internal/prompts/*.md`, never inline in Go.
- Front end: one component per file, props typed inline as `{Name}Props`, early returns,
  every colour a token in `tokens.css`, one shared action registry drives shortcuts,
  palette and command line. It calls `window.go.<pkg>.Service.<Method>`. No new dependencies
  without a reason in the PR.
- Change the contract and `docs/contract.md` in the same PR.
- Minimal, clear UI: flat surfaces, colour only for meaning (state, issue type, focus).

## Workflow

Issues on `nednella/agentos` are the queue. One issue → branch `issue-<n>` in its own
worktree → one draft PR → Ned merges → the app cleans up. Conventional commits, one
logical change each. Manual issues follow `.github/ISSUE_TEMPLATE/issue.md`: a
`## Description` written by a human, never edited by a session; a session appends its
findings below it under `## Agent Review`.
