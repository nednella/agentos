# How the front end and the Go core talk

The Go side is the reference: `desktop/app.go` and the types beside it. This file says the same in
one place. The front end never imports generated Wails files. It calls

- methods: `window.go.main.App.<Method>(...args)`, which return a Promise;
- events: `window.runtime.EventsOn(name, handler)`.

A method that fails rejects with a short human-readable string. A method with no result resolves
to `undefined`. Empty lists are `[]`, never `null`.

When `window.go` is undefined (plain `vite dev`) the front end uses its mock. When the binary runs
with `AGENTOS_HTTP=127.0.0.1:PORT` it serves the real front end in a browser with a shim that
provides `window.go` and `window.runtime` over `POST /__call/<Method>` (a JSON array of
arguments) and the event stream `/__events`.

## Types

```ts
type State = 'waiting' | 'working' | 'idle' | 'ended'
// waiting: blocked on the user (a permission prompt or a question). working: running.
// idle: started, or a turn ended and a reply landed: the user's move.
// ended: the session's process is gone; the row stays until dismissed, for at most 7 days.

type Session = {
  id: string            // tmux session name "<project key>/<n>"; stable; key for every call
  n: number             // small number shown to the user
  title: string
  state: State
  detail: string        // short live line: "Edit route-list.tsx", ""
  lastEventAt: number   // unix ms; 0 if none
  createdAt: number     // unix ms
  issue: number         // GitHub issue the session was started for; 0 if none
  history: { state: State; at: number }[]   // state changes, oldest first, at most 200
  branch: string        // branch of the issue's work (from the project's `branch` pattern); "" without an issue
  worktree: string      // path of the git worktree on that branch; "" if none
  pr: PR | null
  prAttention: '' | 'checks' | 'comments'   // failing checks, or comments newer than the last AckPR
  cleanup: '' | 'pending' | 'blocked' | 'ask'
  cleanupReason: string // why blocked, in a sentence
  browser: boolean      // the session has a browser tab
  evidence: number      // number of evidence items
  endedAt: number       // unix ms when the session ended; 0 while it runs
}

type PR = {
  number: number
  url: string
  state: 'draft' | 'open' | 'merged' | 'closed'
  checks: 'none' | 'pending' | 'passing' | 'failing'
  comments: number      // issue comments plus reviews
  updatedAt: number
}

type Project = {
  name: string
  dir: string
  repo: string          // "owner/name", or ""; filled for the current project, and for others once known
  needsYou: number      // sessions waiting, or idle after a reply landed (not just started)
  working: number
  sessions: number      // all running sessions; ended ones do not count
}

type Issue = {
  number: number
  title: string
  type: 'bug' | 'feature' | 'refactor' | 'chore' | ''   // from a type:<x> label
  lane: 'ready' | 'plan' | 'you' | 'idea' | 'inbox'
  url: string
  sessionId: string     // live session started for this issue, else ""
  author: string        // GitHub login, "" if unknown
  assignees: string[]
  labels: string[]
  createdAt: number
  updatedAt: number
}

type Note = {
  id: string
  text: string          // the whole note; its first line is the title when one is needed
  createdAt: number
  updatedAt: number
  pinned: boolean
  archived: boolean
  images: string[]      // "/media/notes-media/<project key>/<file>"
  issue: number         // GitHub issue filed from the note, 0 if none
  issueUrl: string
}

type Snapshot = {
  project: Project
  projects: Project[]   // every known project: configured, current, and any with live sessions
  sessions: Session[]   // current project, sorted: waiting, idle, working, ended; newest event first in a group
  notes: Note[]         // current project: not archived before archived; pinned first, then newest first
  version: string
}

type Evidence = {
  id: string
  kind: 'image' | 'text'
  url: string           // "/media/evidence/<project key>/<n>/<file>" for images, "" for text
  text: string          // for kind 'text'
  caption: string
  source: 'agent' | 'user'
  at: number
}

// A wait of kind idle opens when a Stop ends a working turn. An idle reminder that stops a working
// session (a turn lost to an error fires no Stop) opens none.
type Wait = {           // one time a session waited on the user
  sessionTitle: string
  issue: number
  kind: 'permission' | 'question' | 'idle'
  label: string         // "Bash: yarn test", "Edit", "Question", "Reply landed"
  startedAt: number
  waitedMs: number      // 0 while still waiting
}

type Stats = {
  days: number
  total: number
  totalWaitMs: number
  medianWaitMs: number  // over finished waits
  byCause: { kind: Wait['kind']; label: string; count: number; totalWaitMs: number }[]  // most frequent first
  byDay: { day: string; count: number }[]   // "2026-10-02", oldest first, every day of the range present
  recent: Wait[]        // newest first, at most 50
}

type Cleanup = {
  at: number
  sessionTitle: string
  issue: number
  pr: number
  status: 'done' | 'blocked'
  removed: string[]     // "worktree trees/issue-394", "branch issue-394", "temp files", "evidence", "session"
  reason: string        // when blocked
}

type BrowserState = {
  id: string            // session id
  open: boolean
  url: string
  title: string
  loading: boolean
  canGoBack: boolean
  canGoForward: boolean
  error: string         // why the browser could not start, e.g. "no Brave, Chrome, Chromium or Edge browser found"
}

type BrowserInput =
  | { type: 'mouse'; action: 'move' | 'down' | 'up'; x: number; y: number; button: 'left' | 'middle' | 'right' | 'none'; clickCount: number; modifiers: number }
  | { type: 'wheel'; x: number; y: number; deltaX: number; deltaY: number; modifiers: number }
  | { type: 'key'; action: 'down' | 'up'; key: string; code: string; text: string; modifiers: number }
  | { type: 'paste'; text: string }
// x, y: CSS pixels in the page viewport. modifiers: 1 alt, 2 ctrl, 4 meta, 8 shift.

type DigestItem = {
  id: string
  title: string
  why: string           // one line: why it matters to this project
  url: string
  source: string
  at: number
  noteId: string        // note it was saved to, "" if not saved
}

type Digest = {
  running: boolean
  lastRunAt: number     // 0 if never
  nextRunAt: number     // 0 if the project's digest is off
  error: string         // short reason a run failed, "" otherwise
  items: DigestItem[]   // newest run first, at most 30
}
```

## Methods

### Projects and sessions

| Method | Returns | What it does |
|---|---|---|
| `Snapshot()` | `Snapshot` | everything the screen needs; call on load |
| `SwitchProject(name)` | `Snapshot` | makes the project current; remembered for the next start |
| `AddProject()` | `Snapshot` | opens the folder picker, adds the folder (named after it, `-2` on a clash), saves the config, switches to it; unchanged snapshot on cancel |
| `AddProjectDir(dir)` | `Snapshot` | the same without the picker; `~` is expanded; rejects a folder that does not exist |
| `RemoveProject(name)` | `Snapshot` | forgets a configured project, keeps its sessions; switches away if it was current |
| `NewSession(title, prefill)` | `Session` | starts the agent in the project folder; a non-empty `prefill` is typed in, not sent, once the agent is ready |
| `KillSession(id)` | | stops the agent; the row stays as `ended` |
| `DismissSession(id)` | | removes the row of an ended session (and its evidence); rejects a running one |
| `RenameSession(id, title)` | | |
| `TypeInto(id, text)` | | types text into the prompt, not sent; line breaks cannot submit it |
| `HarnessCheck()` | `Session` | starts a session titled "Harness check" with the review prompt typed in |

### Terminal

| Method | What it does |
|---|---|
| `TermOpen(id, cols, rows)` | attaches a terminal stream; opening an open one attaches afresh, which redraws everything |
| `TermWrite(id, data)` | raw input bytes as a string (xterm.js `onData`) |
| `TermResize(id, cols, rows)` | |
| `TermClose(id)` | detaches the stream; the agent keeps running |

### Issues

| Method | Returns | What it does |
|---|---|---|
| `Issues(refresh)` | `Issue[]` | the current project's open issues, cached unless `refresh` |
| `StartIssue(number)` | `Session` | opens a session titled `#<n> <short title>` with the lane's command typed in (see config); an issue that has a live session gets that session back |

### Pull requests and clean-up

| Method | Returns | What it does |
|---|---|---|
| `RefreshPRs()` | | polls the PRs of issue sessions now (they are also polled every 3 minutes); results arrive as `sessions` |
| `AckPR(id)` | | the user has seen the PR's comments and failing checks; clears `prAttention` until something new arrives |
| `Cleanup(id, force)` | | cleans up after a session now: worktree, branch, temp files, evidence, browser tab, session. Without `force` the safety checks apply |
| `Cleanups()` | `Cleanup[]` | the current project's log, newest first, at most 100 |

A session is cleaned up on its own when its PR is merged after the session saw it draft or open. A PR first seen already merged, or a
closed unmerged one, sets `cleanup: 'ask'`: the user decides. Blocked means the worktree has changes or the branch has commits that are not on origin;
`Cleanup(id, true)` overrides.

### Notes

| Method | Returns | What it does |
|---|---|---|
| `AddNote(text)` | `Note` | |
| `UpdateNote(id, text)` | `Note` | |
| `SetNotePinned(id, pinned)` | `Note` | |
| `SetNoteArchived(id, archived)` | `Note` | |
| `SetNoteDone(id, done)` | `Note` | old name of `SetNoteArchived` |
| `DeleteNote(id)` | | also deletes its pictures |
| `AddNoteImage(id, base64, mime)` | `Note` | png, jpeg, gif or webp, at most 10 MB; the type is checked against the bytes |
| `RemoveNoteImage(id, url)` | `Note` | |
| `NoteToIssue(id)` | `Note` | files a GitHub issue; title is the first line, body is `## Description`, a blank line, and the note text (plus a line if the note has pictures); sets `issue` and `issueUrl`; rejects without a repo |
| `NoteToSession(id)` | `Session` | starts a session titled with the first line and the project's `note` command typed in |

### Browser

One hidden browser per project (separate profile, so logins persist), one tab per session. Call `BrowserResize` before `BrowserView(id, true)`.

| Method | Returns | What it does |
|---|---|---|
| `BrowserOpen(id, url)` | `BrowserState` | opens the session's tab, starting the browser if needed; `url` "" means the project's `url`, else a blank page |
| `BrowserGoto(id, url)` | | a bare host gets `https://`; localhost, IPs and `.local` get `http://` |
| `BrowserNav(id, action)` | | `'back' \| 'forward' \| 'reload' \| 'stop'` |
| `BrowserInput(id, input)` | | |
| `BrowserResize(id, width, height)` | | CSS pixels of the panel; the page viewport follows |
| `BrowserView(id, visible)` | | frames are sent only while visible |
| `BrowserState(id)` | `BrowserState` | |
| `BrowserScreenshot(id, caption)` | `Evidence` | the user's own capture, filed as evidence (`source: 'user'`) |
| `BrowserClose(id)` | | closes the tab; the browser stops a minute after its last tab |

Links with `target=_blank` and `window.open` stay in the session's tab. Meta+A, C, X, Z run the matching edit command; send paste as `{type:'paste'}`.

### Evidence, stats, digest

| Method | Returns | What it does |
|---|---|---|
| `Evidence(id)` | `Evidence[]` | oldest first |
| `DeleteEvidence(id, evidenceId)` | | |
| `Stats(days)` | `Stats` | the current project's interruption tally |
| `Digest()` | `Digest` | the current project's digest |
| `RunDigest()` | | starts a run in the background; progress arrives as `digest` |
| `DigestToNote(itemId)` | `Note` | saves the item as a note (title, why, link) and marks it saved |
| `DismissDigestItem(itemId)` | | |
| `OpenURL(url)` | | opens an http or https address in the default browser |

## Events

| Name | Payload | When |
|---|---|---|
| `sessions` | `Session[]` | a session of the current project was added, removed, renamed, or changed (state, detail, PR, clean-up, browser, evidence) |
| `projects` | `Project[]` | a project was added or removed, or its counts changed |
| `notes` | `Note[]` | the current project's notes changed |
| `issues` | `Issue[]` | the cached issue list changed (a session started or ended for an issue, a note was filed) |
| `term:data` | `{ id, data }` | `data` is base64 of raw terminal output |
| `term:exit` | `{ id }` | the session ended on its own |
| `attention` | `{ id, state }` | `state` is `'waiting'`, `'replied'` (a turn ended with a reply: working to idle through Stop, not through a reminder), `'pr'` (PR checks or comments) or `'evidence'` (an agent filed evidence); once per change, current project only for waiting and replied |
| `stats` | none | a wait opened or closed; refetch `Stats` if the view is open |
| `cleanups` | `Cleanup[]` | a clean-up finished or was blocked |
| `evidence` | `{ id, items }` | a session's evidence changed |
| `browser:frame` | `{ id, data, width, height }` | `data` is base64 JPEG; width and height are the viewport's CSS pixels |
| `browser:state` | `BrowserState` | URL, title, loading or open changed |
| `digest` | `Digest` | the current project's digest changed |

## Media

Pictures are served by the Go side under `/media/…`, in the window and in the `AGENTOS_HTTP` mode: note images at
`/media/notes-media/<project key>/<file>` and evidence at `/media/evidence/<project key>/<n>/<file>`. Only png, jpeg, gif and webp files of those two
folders are served. Use the URL in `<img src>`.

## Config file

`~/.config/agentos/config.yaml` (`AGENTOS_CONFIG` overrides the path). The app writes it when projects are added or removed, and keeps every key below.

```yaml
data_dir: ~/Library/Mobile Documents/com~apple~CloudDocs/agentos   # optional; default ~/.local/share/agentos
agent: claude                  # claude (the default, with hooks) or any command, which runs plain
projects:
  - name: livedocument
    dir: /Users/me/code/livedocument
    # everything below is optional; the values shown are the defaults
    commands:
      ready: "/work {n}"       # typed into a session started from a ready issue; {n} is the issue number
      plan: "/investigate {n}"
      inbox: ""                # "" types nothing
      idea: ""
      note: "{text}"           # a session started from a note; {text} is the note
    lanes:                     # GitHub label -> lane; when set it replaces the defaults
      ready: ready
      needs-plan: plan
      needs-human: you
      idea: idea
    branch: "issue-{n}"        # branch of an issue's work
    cleanup: ""                # shell command that removes a worktree; {branch}, {worktree}; "" runs git worktree remove
    guard:                     # regexes of shell commands agents may not run; empty means these defaults
      - "gh\\s+pr\\s+merge"
      - "gh\\s+pr\\s+ready"
      - "gh\\s+pr\\s+(edit|create)\\b.*--(add-)?reviewer"
      - "gh\\s+api\\b.*requested_reviewers"
      - "git\\s+push\\b.*(--force(\\s|$)|\\s-f(\\s|$)|\\s\\+\\S)"
    url: ""                    # page a session's browser opens first
    browser: true              # tell sessions about the browser and evidence commands
    digest: weekly             # weekly or off
```

A label that maps to no lane puts an issue in `idea`; an issue with no labels is in `inbox`. When labels map to several lanes the first of
ready, plan, you, idea wins. Only the lanes `ready`, `plan`, `inbox` and `idea` have a command.

### Where things are stored

| What | Where |
|---|---|
| notes, note images, stats, digests | `data_dir` (default `~/.local/share/agentos`): `notes/<key>.json`, `notes-media/<key>/`, `stats/<key>.jsonl`, `digest/<key>.json` |
| evidence, PR tracking, clean-up log, browser profile | always `~/.local/share/agentos`: `evidence/`, `prs/`, `cleanups/`, `browser/<key>/` |
| state files, sockets, tmux config, last project | `~/.local/state/agentos` |

All files are written by writing a temp file and renaming it into place; folders are created as needed.

### Environment

`AGENTOS_CONFIG`, `AGENTOS_STATE_DIR` (state dir), `AGENTOS_DATA_DIR` (replaces both data locations), `AGENTOS_TMUX_SOCKET` (default `agentos`),
`AGENTOS_DIR` (the project folder to start in instead of the current folder), `AGENTOS_HTTP` (browser mode), `AGENTOS_BROWSER` (path of the
browser to drive). Inside a session: `AGENTOS_SESSION`, `AGENTOS_SOCKET`, `AGENTOS_GUARD`. A digest run gets `AGENTOS_DIGEST_PROJECT`.

## Command line

`agentos` opens the app. For agents inside a session: `agentos browser help` (open, snapshot, click, type, press, select, hover, scroll, wait,
wait-for, text, eval, console, screenshot), `agentos show <file> [--caption …] | --text …`, `agentos note <text>`, `agentos stats [--days N] [--json]`.
Used by the app itself: `agentos hook <Event>`, `agentos guard`, `agentos digest add`. Also `agentos kill [--all]` and `agentos version`.
