# agentos

One desktop window for every coding-agent session you run. It shows which session needs you, your
project's GitHub issues as a queue, your notes, and each agent's real terminal. Agents keep running
in a hidden tmux when the window closes.

![agentos](docs/screenshot.png)

- **Sessions** — each agent in its own tmux session, with its state (needs you, working, idle), its
  pull request and whether the checks pass. One shortcut jumps to the session that needs you most.
- **Queue** — the project's open issues by lane (ready, needs plan, needs you, ideas, inbox). Start a
  session from an issue and it opens with the right command typed in.
- **Pull requests** — the app watches each session's PR; a new review comment or a failing check
  wakes the session with what to fix, and a merged PR cleans up the worktree and branch.
- **Notes, evidence, browser** — notes per project, screenshots and text an agent files as evidence,
  and a hidden browser an agent can drive and you can watch.
- **Stats and digest** — what interrupts you most, and a weekly digest of what changed in the tools
  the project depends on.

## Prerequisites

- macOS on Apple silicon
- [tmux](https://github.com/tmux/tmux) — `brew install tmux`
- [GitHub CLI](https://cli.github.com), logged in — `brew install gh && gh auth login`
- [Claude Code](https://docs.claude.com/en/docs/claude-code)
- a Chromium browser (Brave, Chrome, Chromium or Edge), optional, for the agent browser

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/nednella/agentos/main/install.sh | bash
```

This downloads the latest release, checks its sha256, puts `agentos.app` in `~/Applications`, links
the `agentos` command into `~/.local/bin` and tells you what is missing. No sudo.

## Update

The app checks for a new release when it starts and every hour. When one is out, the top bar shows
`vX.Y.Z available · Update`; click it and the app downloads the release, swaps itself and relaunches.
Your sessions live in tmux and survive. From a terminal:

```sh
agentos update
```

Any `agentos` command mentions a newer release once a day.

## First run

Run `agentos`, or open it from Launchpad. Add a project folder with `⌘P` → *Add project*, or from
a terminal in the folder:

```sh
agentos project add
```

The queue fills from the folder's GitHub repository; press `⌘N` for a session, or start one from an
issue with `Enter` on its row.

## Keys

| Keys | Does |
|---|---|
| `⌘N` | New session |
| `⌘E` | Next session that needs you |
| `⌘[` `⌘]` | Previous, next session |
| `⌘1`–`⌘9` | Jump to session n |
| `⌘A` `⌘D` | Focus the queue and notes, the sessions |
| `⌘S` | Focus the shell |
| `⌘B` `⌥⌘B` | Toggle the left panel, the sessions panel |
| `⌘P` | Switch project |
| `⌘R` | Refresh issues and pull requests |
| `⇧⌘[` `⇧⌘]` | Previous, next view (terminal, browser, evidence) |
| `⇧⌘S` `⇧⌘D` | Stats, weekly digest |
| `⌘K` | Command palette: every action, by name |
| `⌘/` | All shortcuts |

## The `agentos` command

The command is the app's own binary; hooks, the command and the app are always one version.

```sh
agentos                      # open the app
agentos issue 394 393        # start a session per issue
agentos new "spike: retries" # start a session
agentos open 2               # show session 2
agentos next                 # show the session that needs you most
agentos pr                   # the sessions' pull requests
agentos cleanup 2            # clean up after session 2
agentos kill                 # stop this project's agents, app closed or not
agentos queue | notes | evidence | term | filter @me type:bug
agentos project              # list projects; agentos project add [path]
agentos stats --days 30
agentos update
agentos --help
```

Inside a session, agents have `agentos note`, `agentos show` (file evidence) and `agentos browser`.

## Config

`~/.config/agentos/config.yaml`. The app writes it when you add or remove projects; everything below
the project name is optional.

```yaml
data_dir: ~/Library/Mobile Documents/com~apple~CloudDocs/agentos   # notes and stats; default ~/.local/share/agentos
agent: claude
projects:
  - name: livedocument
    dir: ~/code/livedocument
    commands:
      ready: "/work {n}"       # sent to a session started from a ready issue
      plan: "/investigate {n}"
    lanes:                     # GitHub label -> lane
      ready: ready
      needs-plan: plan
      needs-human: you
      idea: idea
    branch: "issue-{n}"
    url: "http://localhost:3000"   # the page a session's browser opens first
    digest: weekly
```

`docs/contract.md` lists every key.

## How it works

Each session is a tmux session on a private socket; the window attaches a terminal to it and detaches
when it closes, so agents never stop with the window. Claude Code hooks call `agentos hook`, which
records the session's state and tells the app over a unix socket. Issues and pull requests come from
`gh`; the app streams the repository's events through `gh webhook forward` when it can, and polls
otherwise. The Go core is one Wails window; the front end is React and xterm.js, and talks to the
core through the methods and events in `docs/contract.md`.

## Developing

```sh
make desktop-app     # builds the front end and bin/agentos-dev.app
open bin/agentos-dev.app
make desktop-install # copies it to ~/Applications and links ~/.local/bin/agentos into it
go test -race ./... && go vet ./...
```

Conventional commits drive the releases: release-please keeps a release pull request open, and
merging it tags `vX.Y.Z`, builds the bundle on a macOS runner and attaches
`agentos-darwin-arm64.zip` with its sha256. `docs/testing.md` says how to check a change without
touching your live sessions.
