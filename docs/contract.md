# How the front end and the Go core talk

The Go side is the reference: the `Service` structs in `desktop/<pkg>/service.go`, wired in
`desktop/internal/app`. This file says the same in one place. The front end never imports generated
Wails files. Each package binds one `Service`; it calls

- methods: `window.go.<pkg>.Service.<Method>(...args)`, which return a Promise;
- events: `window.runtime.EventsOn(name, handler)`.

A method that fails rejects with a short human-readable string. A method with no result resolves
to `undefined`. Empty lists are `[]`, never `null`.

When `window.go` is undefined (plain `vite dev`) the front end uses its mock. When the binary runs
with `AGENTOS_HTTP=127.0.0.1:PORT` it serves the real front end in a browser with a shim that
provides `window.go` and `window.runtime` over `POST /__call/<pkg>.Service/<Method>` (a JSON array
of arguments, with the header `X-Agentos: 1`) and the event stream `/__events`. The server answers only requests whose
`Host` is the listen address and whose `Origin`, if any, is `http://<listen address>`; anything else gets 403. `/__call/` also
gets 403 without the `X-Agentos` header, which the shim sends. Every method below is in the service named by its section heading.

## Types

```ts
type State = 'waiting' | 'working' | 'idle' | 'ended'
// waiting: blocked on the user (a permission prompt or a question). working: running.
// idle: started, or a turn ended and a reply landed: the user's move.
// ended: the session's process is gone; the row stays until dismissed, for at most 7 days.

type Session = {
  id: string            // tmux session name "<project key>/<token>", a token of 8 random a-z0-9 chars; unique, never reused; key for every call
  n: number             // number shown to the user; counts up per project for this run of the app, restarts at 1 on relaunch, never reused while the app runs
  title: string
  state: State
  detail: string        // short live line: "Edit route-list.tsx", ""
  lastEventAt: number   // unix ms; 0 if none
  createdAt: number     // unix ms
  issue: number         // GitHub issue the session was started for; 0 if none
  model: string         // model the agent was started with ("sonnet"); "" when none was set or found, for an agent other than claude, or for a session from before the field
  effort: string        // effort it was started with ("medium"); "" likewise
  history: { state: State; at: number }[]   // state changes, oldest first, at most 200
  branch: string        // the branch the session works on: the one it reported with `agentos track`, else the one checked out in its working folder; for an issue session with neither, the project's `session_branch_fallback` pattern. The branch its PR was found on, else the newest; "" if none
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
  key: string           // the project key: the "project" of events and the first part of a session id
  name: string
  dir: string
  repo: string          // "owner/name", or ""; filled for the current project, and for others once known
  needsYou: number      // sessions waiting; the same count the top bar shows
  working: number
  sessions: number      // all running sessions; ended ones do not count
}

type Issue = {
  number: number
  title: string
  type: 'bug' | 'feature' | 'refactor' | 'chore' | ''   // from a type:<x> label
  section: string       // the queue section it is in (see config); "" when the project defines none
  actions: string[]     // names of what can start a session for it; the first is the default
  url: string
  sessionId: string     // live session started for this issue, else ""
  author: string        // GitHub login, "" if unknown
  assignees: string[]
  labels: string[]
  createdAt: number
  updatedAt: number
}

// The issue dialog: what a row does not carry. The HTML is GitHub's own rendering of the Markdown, the same
// github.com shows and sanitizes; the front end places it as it is and routes its links through openURL.
type IssueDetail = {
  number: number
  bodyHTML: string      // "" for an empty body
  comments: { author: string; createdAt: number; bodyHTML: string }[]   // oldest first, at most 100
}

type Note = {
  id: string
  text: string          // the whole note; its first line is the title when one is needed
  createdAt: number
  updatedAt: number
  pinned: boolean
  archived: boolean
  images: string[]      // "/media/<project key>/notes-media/<file>"
}

type Snapshot = {
  project: Project      // project.key says which project the rest is about; drop events of another key
  projects: Project[]   // every known project: configured with a folder on this machine, current, and any with live sessions
  sessions: Session[]   // current project, sorted: waiting, idle, working, ended; newest event first in a group
  shells: string[]      // the current project's shell session ids in tab order ("<key>/shell", "<key>/shell-2", ...), empty until ShellOpen
  notes: Note[]         // current project: not archived before archived; pinned first, then newest first
  version: string       // the running version: "1.2.3", or "dev" for a local build
  update: string        // the version of a newer release, "" when the app is current
}

// The sessions, notes, issues and cleanups events carry a list of one project. Drop an event whose project is not
// the current one: it can arrive after a switch.
type ProjectList<T> = { project: string; items: T[] }   // project: the project key

// The warnings event: a failure that would otherwise be silent. Show it once; an empty message says the source works again.
type Warning = {
  source: 'tmux' | 'github' | 'pull requests' | 'worktrees'
  message: string       // first line of the failure, "" when the source recovered
}

type Evidence = {
  id: string
  kind: 'image' | 'text'
  url: string           // "/media/<project key>/evidence/<token>/<file>" for images, "" for text
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
  model: string         // the session's model, "" when unknown
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
  merged: boolean       // the PR was merged when the clean-up ran
  status: 'done' | 'blocked'
  removed: string[]     // "clean-up command for issue-394", "temp files", "evidence", "session"
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
  headed: boolean       // the page lives in its own browser window (the default); false: hidden browser, frames over `browser:frame`
  loadedAt: number      // unix ms the page last finished loading; 0 if it never has
  console: string[]     // the page's last 20 console errors, uncaught errors and failed requests, oldest first; a copy, so the agent's `agentos browser console` does not empty it
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
  project: string       // project key
  running: boolean
  lastRunAt: number     // 0 if never
  nextRunAt: number     // 0 if the project's digest is off
  error: string         // short reason a run failed, "" otherwise
  items: DigestItem[]   // newest run first, at most 30
}
```

## Methods

### Projects (`projects`)

| Method | Returns | What it does |
|---|---|---|
| `Snapshot()` | `Snapshot` | everything the screen needs; call on load |
| `SwitchProject(name)` | `Snapshot` | makes the project current; remembered for the next start |
| `AddProject()` | `Snapshot` | opens the folder picker, adds the folder (named after it, `-2` on a clash), saves the config, switches to it; unchanged snapshot on cancel |
| `AddProjectDir(dir)` | `Snapshot` | the same without the picker; `~` is expanded; rejects a folder that does not exist |
| `NewProject(name)` | `Snapshot` | opens the folder picker for the parent folder, creates `<parent>/<name>` with a README, `git init -b main` and a first commit, creates a public GitHub repo `<name>` as its `origin` with `gh repo create --push`, then adds and switches to it. Rejects a name outside letters, digits, `.`, `-`, `_` and a folder that exists; a failed git step removes the new folder, a failed `gh` step keeps it and says so; unchanged snapshot on cancel |
| `RemoveProject(name)` | `Snapshot` | forgets a configured project and ends its sessions; switches away if it was current, even when the config never listed it (the folder the app started in). Rejects while the project has live issue sessions (the message names them): their branch pattern and clean-up command come from the project. Editing the config file keeps its comments and `~` paths |

### Sessions (`sessions`)

| Method | Returns | What it does |
|---|---|---|
| `NewSession(title, prefill)` | `Session` | starts the agent in the project folder with the model and effort of claude's own settings (see config); a non-empty `prefill` is typed in, not sent, once the agent is ready |
| `KillSession(id)` | | stops the agent; the row stays as `ended`; rejects the shell |
| `DismissSession(id)` | | removes the row of an ended session, with its evidence, its browser tab and its PR tracking; rejects a running one. A row left for 7 days goes the same way |
| `RenameSession(id, title)` | | rejects the shell |
| `TypeInto(id, text)` | | types text into the prompt, not sent; line breaks cannot submit it |
| `ShellOpen()` | `{ id: string }` | makes sure the current project has a shell session and returns the first one's id. A shell runs (`$SHELL -l` in the project folder, kept in the hidden tmux, with `AGENTOS_PROJECT` and `AGENTOS_SOCKET` set and no `AGENTOS_SESSION`); attach with `TermOpen` like any session. It is not a session: not in `sessions`, the counts, clean-up or the hook states |
| `ShellNew()` | `{ id: string }` | starts another shell in the current project, numbered with the lowest free number, and returns its id |
| `ShellClose(id)` | | stops one shell; rejects an agent session |

### Terminal (`terminal`)

| Method | What it does |
|---|---|
| `TermOpen(id, cols, rows)` | attaches a terminal stream. Opening an open one keeps its stream, sets the new size and redraws everything, so an open that follows a late close never kills the new stream |
| `TermWrite(id, data)` | raw input bytes as a string (xterm.js `onData`) |
| `TermResize(id, cols, rows)` | |
| `TermClose(id)` | detaches the stream; the agent keeps running |

### Issues (`issues`)

| Method | Returns | What it does |
|---|---|---|
| `Issues(refresh)` | `Issue[]` | the current project's open issues, section by section in the order the project lists them (the order GitHub gave inside a section), cached unless `refresh`, which the front end passes when the queue is shown, when the project changes and when the window regains focus while it is shown; rejects with exactly `issues are disabled for this repo` when the repo has issues turned off, and the front end shows that as a notice, not an error |
| `StartIssue(number, action)` | `Session` | opens a session titled `#<n> <short title>` and starts the agent with the model and effort the action and the issue's labels pick (see config), then types the action's command once the agent is ready, and sends it unless `session_prompt_send` is `manual`. `action` is one of the issue's `actions`; `""` is the first. Rejects an action the issue does not have. An issue that has a live session gets that session back |
| `IssueDetail(number)` | `IssueDetail` | the issue's body and comments, rendered by GitHub (`gh api` with `Accept: application/vnd.github.html+json`); not cached; rejects when gh cannot read the issue |

### Pull requests and clean-up (`sessions`)

| Method | Returns | What it does |
|---|---|---|
| `RefreshPRs()` | | fetches the PRs of sessions on a branch now (the app also watches them, see below); results arrive as `sessions` |
| `AckPR(id)` | | the user has seen the PR's comments and failing checks; clears `prAttention` until something new arrives |
| `Cleanup(id, force)` | | cleans up after a session now: runs the project's `session_cleanup_command` for each of its branches, then removes temp files, evidence, browser tab and session. Without `force` the safety checks apply |
| `Cleanups()` | `Cleanup[]` | the current project's log, newest first, at most 100 |

A session with an issue works on the issue's branch. A session without one works on the branches it was seen on: each hook reports the agent's
working directory (`cwd`), and on a change of directory and at the end of each turn the app reads the branch checked out there. A directory in another
repo, a detached head and the main branch (origin's default, else `main`) are ignored. The branches are remembered per session in `prs.json`, oldest
first, and the PR of a session is the one on the newest branch that has one. A session with no such branch has no PR and nothing to clean up. Every
branch of the session goes through the clean-up command, whether the branch is checked out in the project folder or in a linked worktree. A new comment or failing check wakes a live session
of this kind; an ended one is not recreated.

Whether a session is cleaned up on its own is the project's `session_cleanup_mode`, per event: `merge` is `auto` by default, `close` is `manual`.
An `auto` event cleans up when the PR reached it after the session saw the PR draft or open. A PR first seen already merged or closed, or an event set to
`manual`, sets `cleanup: 'ask'`: the user decides. The safety checks apply in both modes. Blocked means the worktree has changes or the branch has commits that are not on origin; for a merged PR whose remote branch is gone, the branch must be contained in the commit the PR merged (`headRefOid`), else the reason is "the branch has commits that are not in the merged pull request". Closing the app during a clean-up stops it and leaves a `blocked` entry with the reason "app closed during clean-up".
`Cleanup(id, true)` overrides.

The git side of a clean-up belongs to the project: the app runs `session_cleanup_command` through `sh -c` in the project folder, once per branch of the session that has a
worktree or a local branch, after the safety checks and before it removes its own session. `{branch}`, `{worktree}` (the folder the branch is checked out in, `''` if none)
and `{dir}` (the project folder) are shell-quoted. `{force}` is `--force` on a forced clean-up (`Cleanup(id, true)`) and empty otherwise, so
`git worktree remove {force} {worktree}` removes a worktree with changes only when the user forced it. A command that fails stops the clean-up with `blocked` and the command's error, unless the branch no longer exists locally and no worktree has it checked out: then the error is logged on the clean-up entry and the clean-up carries on. With no command, a clean-up removes only the session:
branches and folders stay. An ended session's PR is tracked like a live one's until its row is dismissed or cleaned up, so a merge after the
agent finished still cleans up, and the row shows the PR merged.

How the app watches: every 5 minutes it fetches every tracked PR. In between, each project with a session on a branch has a watcher. Unless
the project's `pr_watch_method` says `poll`, the watcher streams the repo's `pull_request`, `pull_request_review`, `issue_comment` and `check_suite` events
through `gh webhook forward` (needs the extension and admin on the repo; a stream that ends within 10 s never worked, and the project is polled
instead; one that drops later is retried after 30 s, with a warning when `pr_watch_method: webhook` asked for it). Polling asks GitHub for the repo's
PR list every `pr_poll_interval` (10 s) with `If-None-Match`, so an unchanged list costs no request, and fetches only the PRs whose `updated_at` moved,
plus those whose checks are still running. Whatever the watcher says, a session's Stop hook looks its branch's PR up at once, so a PR the agent opened during a turn shows when the turn ends. When a tracked PR merges or closes, the queue is read again (`issues`). When a PR gets a new
comment or review, the app types the project's `pr_review_command` into its session and sends it; failing checks on a new commit send
`pr_checks_command` (`{n}` is the PR number). A key that is not set sends nothing: the row is still flagged and the notice still shows. A session
waiting on the user gets the text once it is not; an ended session gets a new session for its issue with the text sent, and its row goes. The new session resumes the old one's conversation
(`claude --resume <id>`, the `session_id` of the hook events, kept in the state file) in the project folder. An agent that cannot resume, or a session with no
recorded id, gets a fresh conversation.
Each comment count and each failing commit counts once, remembered in `prs.json`, whether or not a command is set.

### Notes (`notes`)

| Method | Returns | What it does |
|---|---|---|
| `AddNote(text)` | `Note` | `text` may be empty: a note made to hold pictures is created first, and the pictures are attached right after. (`UpdateNote` and `agentos note` still reject empty text; `NoteToIssue` rejects a note with no text) |
| `UpdateNote(id, text)` | `Note` | |
| `SetNotePinned(id, pinned)` | `Note` | |
| `SetNoteArchived(id, archived)` | `Note` | |
| `DeleteNote(id)` | | also deletes its pictures |
| `AddNoteImage(id, base64, mime)` | `Note` | png, jpeg, gif or webp, at most 10 MB; the type is checked against the bytes |
| `RemoveNoteImage(id, url)` | `Note` | |
| `NoteToIssue(id)` | | files a GitHub issue, then deletes the note and its pictures; title is the first line, body is `## Description`, a blank line, and the note text; each picture is uploaded with `gh issue create --attach` (gh 2.99 or later) and appended to the body. When gh files the issue but a picture fails to upload, the note is kept and the error names the issue; rejects without a repo |
| `NoteToSession(id)` | `Session` | starts a session titled with the first line and the project's `note_session_command` typed in (the note itself when unset), sent unless `session_prompt_send` is `manual` |

### Browser (`browser`)

One browser per project (separate profile, so logins persist), one window per session, opened behind the other windows so it takes no focus. The page follows the window's real size and pixel ratio. With `AGENTOS_BROWSER_HEADLESS=1` the browser is hidden instead and each session has a tab the front end draws from `browser:frame`; call `BrowserResize` before `BrowserView(id, true)` there.

| Method | Returns | What it does |
|---|---|---|
| `BrowserOpen(id, url)` | `BrowserState` | opens the session's tab, starting the browser if needed; `url` "" means the project's `browser_start_url`, else a blank page. Rejects when that first page cannot be opened; the tab then stays open and `browser:state` has it. Two calls for one session share one tab |
| `BrowserGoto(id, url)` | | a bare host gets `https://`; localhost, IPs and `.local` get `http://` |
| `BrowserNav(id, action)` | | `'back' \| 'forward' \| 'reload' \| 'stop'` |
| `BrowserInput(id, input)` | | |
| `BrowserResize(id, width, height)` | | headless only (no-op when `headed`): CSS pixels of the panel; the page viewport follows |
| `BrowserView(id, visible)` | | headless only (no-op when `headed`): frames are sent only while visible |
| `BrowserShow(id)` | | brings the session's window to the front and focuses it |
| `BrowserState(id)` | `BrowserState` | |
| `BrowserScreenshot(id, caption)` | `Evidence` | the user's own capture, filed as evidence (`source: 'user'`) |
| `BrowserClose(id)` | | closes the tab and its window; closing the window closes the tab; the browser stops a minute after its last tab |

A headed window's title reads `[<project name> · <n> <session title>] <page title>`, or just the bracketed label when the page has no title; the label is fixed when the window opens, so a rename shows only in windows opened later. `BrowserState.title`, `agentos browser open` and `agentos browser url` report the page's own title. `agentos browser open <url> --front` also brings the window to the front; without `--front` the window stays behind.

Links with `target=_blank` and `window.open` stay in the session's tab. Meta+A, C, X, Z run the matching edit command; send paste as `{type:'paste'}`.

### Evidence, stats, digest (`evidence`, `stats`, `digest`)

| Method | Returns | What it does |
|---|---|---|
| `Evidence(id)` | `Evidence[]` | oldest first |
| `DeleteEvidence(id, evidenceId)` | | |
| `Stats(days)` | `Stats` | the current project's interruption tally |
| `Digest()` | `Digest` | the current project's digest |
| `RunDigest()` | | starts a run in the background; progress arrives as `digest` |
| `DigestToNote(itemId)` | `Note` | saves the item as a note (title, why, link) and marks it saved |
| `DismissDigestItem(itemId)` | | |

### Update (`update`)

| Method | What it does |
|---|---|
| `Update()` | downloads the newer release (`agentos-darwin-arm64.zip` of the latest GitHub release, through curl, so no quarantine flag is set), checks its sha256, swaps the `.app` bundle and relaunches the app; sessions live in tmux and survive. Resolves once the relaunch is arranged; rejects with the reason otherwise, and when the app runs from no bundle |

The app asks GitHub's releases API on start and every hour. A release newer than `Snapshot.version` arrives as `update`; a dev build is never behind.

### Sleep (`awake`)

| Method | Returns | What it does |
|---|---|---|
| `Awake()` | `boolean` | whether the app holds off idle sleep now; later changes arrive as `awake` |

### Settings (`settings`)

| Method | Returns | What it does |
|---|---|---|
| `Settings()` | `Settings` | the settings: `{theme: "system" \| "light" \| "dark", textScale, keepAwake, cleanup: {merge, close}, promptSend, browserEnabled, digestSchedule}`. `textScale` is a multiple of the default text size (1 when unset). `keepAwake` is the top-level `keep_mac_awake`. `cleanup`, `promptSend`, `browserEnabled` and `digestSchedule` are the current project's, the defaults filled in: `cleanup` has each of `merge` and `close` as `"auto"` or `"manual"`, `promptSend` is `"auto"` or `"manual"`, `digestSchedule` is `"weekly"` or `"off"`. `dataDir` is the folder the running app keeps notes, evidence, stats and digests in; `dataDirFixed` is true when `AGENTOS_DEV_DATA_DIR` sets it |
| `SetTheme(theme)` | `Settings` | saves the theme to `app_theme` in the config file; `system` removes the key. Rejects any other value |
| `SetCleanup(event, mode)` | `Settings` | saves the current project's clean-up mode for `merge` or `close` to `auto` or `manual`. Rejects any other value, and a project that is not in the config file |
| `SetTextScale(scale)` | `Settings` | saves the text size to `app_text_scale`; 1 removes the key. Rejects a value below 0.85 or above 1.5 |
| `SetKeepAwake(on)` | `Settings` | saves the top-level `keep_mac_awake`. A project's own `keep_mac_awake` still wins |
| `SetPromptSend(mode)` | `Settings` | saves the current project's `session_prompt_send`, `auto` or `manual`. Rejects any other value, and a project that is not in the config file |
| `SetBrowserEnabled(on)` | `Settings` | saves the current project's `browser_enabled`. Rejects a project that is not in the config file |
| `SetDigestSchedule(schedule)` | `Settings` | saves the current project's `digest_schedule`, `weekly` or `off`. Rejects any other value, and a project that is not in the config file |
| `PickDataDir()` | `{dir, empty}` | opens the folder picker for a new data folder and changes nothing. `dir` is `""` when cancelled; `empty` is true when the folder holds nothing but hidden files. Rejects when `dataDirFixed` |
| `SetDataDir(dir, withData)` | — | saves `data_dir`, then relaunches the app, as an update does, to use it. With `withData` it first copies each project's notes, note images, evidence, stats and digest into `dir`, which must be empty, and removes what it copied if the copy fails; the old folder stays as it was. Without, the app uses what `dir` holds. Rejects the current folder, and any change when `dataDirFixed`. When the relaunch fails (no .app bundle) the folder is saved and the error says to reopen the app |

A saved setting applies at once; the app reads the config file only when it starts.

## Events

| Name | Payload | When |
|---|---|---|
| `sessions` | `ProjectList<Session>` | a session of the current project was added, removed, renamed, or changed (state, detail, PR, clean-up, browser, evidence) |
| `projects` | `Project[]` | a project was added or removed, or its counts changed |
| `notes` | `ProjectList<Note>` | the current project's notes changed |
| `issues` | `ProjectList<Issue>` | the cached issue list changed (a session started or ended for an issue, a note was filed) |
| `term:data` | `{ id, data }` | `data` is base64 of raw terminal output |
| `term:exit` | `{ id }` | the session ended on its own |
| `attention` | `{ id, state }` | `state` is `'waiting'`, `'replied'` (a turn ended with a reply: working to idle through Stop, not through a reminder), `'opened'` (the session is idle and its PR is new to the app: it opened the PR and stopped; sent instead of `replied` when both hold, once per PR), `'pr'` (PR checks or comments) or `'evidence'` (an agent filed evidence); once per change, current project only for waiting and replied |
| `stats` | none | a wait opened or closed; refetch `Stats` if the view is open |
| `cleanups` | `ProjectList<Cleanup>` | a clean-up finished or was blocked; the project is the one the clean-up ran in, which may not be the current one |
| `evidence` | `{ id, items }` | a session's evidence changed |
| `browser:frame` | `{ id, data, width, height }` | headless mode only. `data` is base64 JPEG; width and height are the viewport's CSS pixels |
| `browser:state` | `BrowserState` | URL, title, loading, open or `loadedAt` changed, or `console` gained lines (sent at most every 500 ms, the lines of that span in one event) |
| `ui:command` | `{ name, args: string[] }` | a CLI command wants the front end to change the view: `queue`, `notes`, `evidence`, `term`, `browser`, `next`, `digest`, `stats` (no args); `filter` (the query words); `open` (a session number or title, already checked to exist) |
| `digest` | `Digest` | the current project's digest changed; `project` is its key |
| `update` | `{ version: string }` | a release newer than the running one is out; once per release |
| `awake` | `boolean` | the app took or let go of its idle-sleep assertion: it holds one while a session of any project whose `keep_mac_awake` is on is `working` |
| `warnings` | `Warning` | a service met a failure it cannot show otherwise: tmux could not be listed (`tmux`; the sessions stay as they were), `gh` could not name the repo (`github`; retried after 30 s), pull requests or worktrees could not be read (`pull requests`, `worktrees`). Sent once per distinct message of a source, and again with `message: ""` when the source works |

Opening a web address is the front end's job: the window runtime's `BrowserOpenURL`, else `window.open`. No Go method does it.

## Media

Pictures are served by the Go side under `/media/…`, in the window and in the `AGENTOS_HTTP` mode: note images at
`/media/<project key>/notes-media/<file>` and evidence at `/media/<project key>/evidence/<token>/<file>`. Only png, jpeg, gif and webp files of those two
folders are served. Use the URL in `<img src>`.

## Config file

`~/.config/agentos/config.yaml` (`AGENTOS_CONFIG` overrides the path). The app writes it when projects are added or removed and when a setting changes, and keeps every key below, its comments and its `~` paths.

```yaml
data_dir: ~/Library/Mobile Documents/com~apple~CloudDocs/agentos   # optional; default ~/.local/share/agentos. The settings panel writes it; the app reads it when it starts
agent_command: claude          # claude (the default, with hooks) or any command, which runs plain
keep_mac_awake: true          # stop the Mac idle-sleeping while a session works; a project may set its own. The settings panel writes it
app_theme: dark                # light or dark; unset follows the macOS appearance. The settings panel writes it
app_text_scale: 1.2            # text size as a multiple of the default, 0.85 to 1.5; unset is 1. The settings panel writes it
projects:
  - name: storefront
    directory: /Users/me/code/storefront
    # everything below is optional; the values shown are the defaults
    queue_sections: []         # the groups of the queue, in order; see below
    note_session_command: ""   # typed into a session started from a note; {text} is the note; "" types the note itself
    session_prompt_send: auto  # auto types what a session starts with and sends it at once; manual leaves it on the prompt for Enter. The settings panel writes it
    session_branch_fallback: "" # branch of an issue's work, for an issue session that reports none; {n} is the issue number; "" names none
    session_cleanup_command: "" # shell command that cleans up the git side of a session; {branch}, {worktree}, {dir}, {force} (--force when forced, else empty), e.g. "git worktree remove {force} {worktree} && git branch -D {branch}"; "" leaves branches and folders alone
    session_cleanup_mode:      # does the app clean up by itself? The settings panel writes it
      merge: auto              # auto or manual: after the pull request merged; auto needs the session to have seen it open
      close: manual            # auto or manual: after the pull request closed unmerged; auto needs the session to have seen it open
    browser_start_url: ""      # page a session's browser opens first
    browser_enabled: true      # tell sessions about the browser and evidence commands. The settings panel writes it
    digest_schedule: weekly    # weekly or off. The settings panel writes it
    keep_mac_awake: true       # this project's choice; unset follows the top-level one
    pr_watch_method: ""        # webhook or poll; "" tries gh webhook forward and polls when it does not work
    pr_poll_interval: 10s      # how often to poll the pull requests; at least 1s
    pr_review_command: ""      # typed into a session whose PR got a review or comment, {n} the PR number, e.g. "/address-review {n}"; "" sends nothing
    pr_checks_command: ""      # typed into a session whose PR has failing checks, {n} the PR number; "" sends nothing
```

### Queue sections and actions

`queue_sections` groups the queue and says what a session can start from each group. Each section matches issues by label and lists actions.

```yaml
queue_sections:
  - name: Inbox
    labels: []                  # issues with no labels
    actions:
      - {name: Plan, command: "/plan {n}", model: opus, effort: high}
      - {name: Investigate, command: "/investigate {n}", model: opus}
      - {name: Work, command: "/work {n}"}
  - name: Ready
    labels: [ready]
    actions:
      - {name: Work, command: "/work {n}"}
```

- An issue goes in the first section it matches: it has any of the section's `labels`, or the section's `labels` is empty and the issue has none.
- An issue no section matches goes in an `Other` section after the last, so no issue is hidden.
- In a command, `{n}` is the issue number and `{title}` its title. An empty `command` starts the agent with nothing typed. `model` and `effort` are optional.
- The first action of a section is its default. A section needs a name, an action needs a name, and each is unique in its list; a config that breaks this does not load.
- With no `queue_sections` the queue is one list, `Issue.section` is `""`, and each issue has one action, `Start`, which types `Work on issue #<n>: <title>`. A section with no actions has that same action.

The old `lanes`, `commands`, `models`, `session_model` and `session_effort` keys, and the built-in model, effort and commands, are gone. They are not read, and the app removes `session_model` and `session_effort` from the file the next time it saves a setting.

### How a session picks its model

Only the `claude` agent gets `--model` and `--effort`; any other agent runs as configured. The choice comes from the first of these that sets
a value, model and effort each on their own: the issue's `model:<x>` or `effort:<y>` label (a value claude would not accept is ignored); the
action's `model` and `effort`. A session started for an issue uses
the action and the labels; a new session or one started from a note has neither. Last comes claude's own settings, read when the session
starts: `model`, and the effort in `modelSettings.<model>.effortLevel` for the model chosen, else `effortLevel`, from the project folder's
`.claude/settings.local.json`, then its `.claude/settings.json`, then `settings.json` in `$CLAUDE_CONFIG_DIR` or `~/.claude`. When none sets a
value, no flag is passed and claude uses its own default. The choice is saved on the tmux session (`@agentos-model`, `@agentos-effort`) and in the state file of an
ended one, so it outlives a restart. `Session.model` and `Session.effort` show it ("" when none was set), and each wait in the tally records the model.

### Keeping the Mac awake

While at least one session is `working`, in any project whose `keep_mac_awake` is on (the project's own setting, else the top-level one, else on), the
app runs `caffeinate -i -w <app pid>`: the Mac does not idle-sleep and the display still may. It stops when no such session works, when the app
quits, and when the app dies. It does not stop a closed lid from sleeping the Mac unless the Mac is in
clamshell mode (external display and power connected), and it does not stop a manual sleep or a sleep from low battery.

### Where things are stored

Everything is grouped by project (`<key>` is the project name, lower-cased, with other characters as `-`).

| What | Where |
|---|---|
| notes, note images, evidence, stats, digest | `<data_dir>/<key>/notes.json`, `notes-media/`, `evidence/<token>/`, `stats.jsonl`, `digest.json` (`data_dir` defaults to `~/.local/share/agentos`) |
| PR tracking (acks, live flags, wakes), clean-up log, browser profile, session temp folders | always under `~/.local/share/agentos/<key>/`: `prs.json`, `cleanups.json`, `browser/`, `tmp/<token>/` (removed when the session is dismissed) |
| state files, sockets, tmux config, last project, app location, release check | `~/.local/state/agentos` (`control.sock`, `tmux.conf`, `last-project`, `app-path`: the bundle the app runs from, for `agentos`; `update.json`: the last release check, so the command asks GitHub at most once a day) |

All files are written by writing a temp file and renaming it into place; folders are created as needed.

### Environment

`AGENTOS_CONFIG`, `AGENTOS_STATE_DIR` (state dir), `AGENTOS_DEV_DATA_DIR` (replaces both data locations), `AGENTOS_TMUX_SOCKET` (default `agentos`),
`AGENTOS_DIR` (the project folder to start in instead of the current folder), `AGENTOS_HTTP` (browser mode), `AGENTOS_BROWSER` (path of the
browser to drive), `AGENTOS_BROWSER_HEADLESS` (`1`: hide the browser and stream frames instead of opening windows). Inside a session: `AGENTOS_SESSION`, `AGENTOS_SOCKET`, `TMPDIR` and `CLAUDE_CODE_TMPDIR` (both the session's temp folder), and for a session started for an issue `AGENTOS_ISSUE` (its number). Claude starts with `--settings` holding the hooks and `permissions.additionalDirectories` set to `data_dir` and `~/.local/share/agentos`, so it uses the app's folders, its temp folder among them, without asking. In the shell session: `AGENTOS_PROJECT`, `AGENTOS_SOCKET`. A digest run gets
`AGENTOS_DIGEST_PROJECT`. It runs on the first agent that is installed and signed in, found without spending a request (`claude auth status` exits 0 when signed in); only `claude` is supported so far. With none, the digest's `error` says so. The run happens in an empty temporary folder with only `WebSearch`, `WebFetch` and `agentos digest add`, and an environment cut to `PATH`, `HOME`, the two `AGENTOS_` variables and what `claude` needs to log in and reach its provider (`ANTHROPIC_*`, `CLAUDE_*`, `AWS_*`, proxy and certificate variables). The prompt lists the package and module names the app read from `package.json` and `go.mod` files (placeholder `{dependencies}`). `agentos digest add` takes only `http` and `https` links.

The texts given to agents (the digest run, the browser lines in a session's system prompt, `agentos browser help`)
are Markdown files in `internal/prompts`.

## Command line

The `agentos` command is the app's own binary: `~/.local/bin/agentos` is a link to `agentos.app/Contents/MacOS/agentos`, so hooks, the
command and the app are always one version. `agentos` with no command, typed at a terminal, opens the app (`open -a` on `agentos.app`, looked
for next to the command, in `~/Applications`, in `/Applications`, then at the path in `app-path`; it prints where it looked when there is
none). With no terminal attached (Finder, `open`, the dock) it is the app starting; `AGENTOS_HTTP` makes it serve the browser mode instead.
`agentos --help` lists the commands by group. Every command below except `kill` without a number talks to the running app over its control
socket, and exits 1 with "agentos is not running: open the app and try again" when there is none. The project is `AGENTOS_PROJECT` (set in the shell
session) or the asking session's, else the current one; the app switches to it. Commands that start or change sessions do the work and say what
happened ("started 2 sessions: #394 (3), #393 (4)"); view commands send a `ui:command` event and answer "ok".

| Group | Command | Does |
|---|---|---|
| Work | `issue <n...>` | starts a session per issue; an issue with a live session is reported as already running |
| Work | `new [title]`, `kill <n>`, `open <n\|title>`, `next` | start, stop, show a session, or show the one that needs you most |
| Work | `refresh`, `pr [n]`, `cleanup [n]` | reload issues and PRs, show PRs, clean up or list what waits |
| Views | `queue`, `notes`, `evidence`, `term`, `browser`, `digest`, `stats --open`, `filter [query]` | show that view; `digest --run` starts a run |
| Projects | `project [name]`, `project add [path]`, `project remove <name>` | list, switch, add (the current folder by default), forget |
| From inside a session | `browser <command>`, `show <file> [--caption …] \| --text …`, `note <text>`, `track --branch <name>` | `agentos browser help` lists the browser commands |

`stats [--days N] [--json]` prints the interruption tally. `agentos kill` without a number stops this project's agents directly through tmux (works with
the app closed); `--all` does it for every project and stops the shells too. Used by the app itself: `agentos hook <Event>`, `agentos digest add`.
Also `agentos version`, and `agentos update`: it installs the latest release over the bundle it finds (as `Update()` does), then asks a
running app to relaunch (`relaunch` over the control socket), or says to run `agentos` when none runs. Every command typed at a terminal
prints `agentos vX.Y.Z is out: run agentos update` on stderr when a newer release exists, asking GitHub at most once a day; hooks and
agents have no terminal and never see or wait for it.
