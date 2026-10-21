# Testing a change

## Isolation first

Twice a test that used the default tmux socket killed the owner's live sessions. Never run anything that starts the app, a session or the CLI against the default socket or the real state folders. Every run outside `go test` MUST set all of these:

```sh
S=$(mktemp -d "${TMPDIR:-/tmp}/aos.XXXX")
export AGENTOS_TMUX_SOCKET=aos-test-$$      # never "agentos", the owner's socket
export AGENTOS_STATE_DIR=$S                 # never ~/.local/state/agentos
export AGENTOS_DEV_DATA_DIR=$S/data             # never ~/.local/share/agentos (also moves notes and the event log)
export AGENTOS_CONFIG=$S/config.yaml        # never ~/.config/agentos/config.yaml
export AGENTOS_AGY_CONFIG_DIR=$S/agy        # never ~/.gemini/config, the owner's own agy hooks
```

A throwaway config with a plain shell as the agent keeps Claude out of it:

```yaml
agent_command: bash
projects:
  - {name: demo, directory: <$S>/demo}
```

Clean up afterwards: `tmux -L $AGENTOS_TMUX_SOCKET kill-server`, kill the app and any browser started on `$S/data/<key>/browser`, `rm -rf $S`. Check with `pgrep -fl "$S"`.

An agentos session runs with `TMPDIR` set to its own temp folder, which it may use without asking and which goes when the session is dismissed. Put screenshots and scratch files there, never in `/tmp`.

## Unit and integration tests

```sh
go vet ./...
go test -race ./...
```

Each `desktop/<pkg>` service has its tests beside it. They build the whole app through `desktop/internal/apptest`, which starts real tmux servers on private sockets (`aostest-<pid>`), real git repos in temp folders, and real headless Chrome for Testing on temp profiles. GitHub and `claude` are always replaced by fake runners (`apptest.FakeGH`, `FakeClaude`); commands run through `desktop/internal/run`. The browser tests skip when Chrome for Testing is not installed; CI points `AGENTOS_BROWSER` at its Google Chrome. `desktop` embeds `frontend/dist`; build it first, or run with `-tags stub`, which embeds a small page instead. The hooks in the tests run the `agentos` command built from `./desktop` with that tag: the app's binary is the command. The `cli` tests answer from a fake app on a temp control socket. The release check never reaches GitHub: the harness answers curl with `Options.Releases` and refuses every other download.

## Recorded agents

`internal/agent/testdata/<provider>/` holds what the real agents sent: for each scenario in `scenario_test.go` (a prompt, a tool call, a question, a permission prompt, an exit, a resume) the hook payloads and the transcript lines with usage, and in `version` the agent's version. `TestScenarios` replays them through the provider as the app reads them and checks the state, reply and usage each row must end with. A scenario a provider cannot do is marked in the table with the reason and skipped for it.

After an update of `claude` or `agy`, record again with the isolation env above set:

```sh
go test ./internal/agent -run TestRecord -record claude,agy -v
git diff internal/agent/testdata
go test ./internal/agent -run TestScenarios
```

The diff shows what the agent changed, the test what that breaks. Recording runs each agent for real, on a cheap model, in a throwaway folder on a tmux server of its own with no agentos session around it, and spends a few requests. The agents use the owner's login; the recording keeps no line of the transcripts besides the model calls, and puts `$WORK`, `$HOME` and `$RECORDING` in place of the machine's paths.

## The real front end, in a browser

`AGENTOS_HTTP` serves the embedded front end over HTTP with a shim for `window.go` and `window.runtime`: `POST /__call/<pkg>.Service/<Method>` with a JSON array of arguments and the header `X-Agentos: 1`, and server-sent events at `/__events`. The server answers only requests addressed to its own listen address (`127.0.0.1:PORT`, not `localhost`) and refuses a foreign `Origin` with 403.

```sh
make -o desktop-frontend desktop-app          # or: make desktop-app (rebuilds the front end)
(cd / && AGENTOS_HTTP=127.0.0.1:18772 bin/agentos-dev.app/Contents/MacOS/agentos) &
curl -s -XPOST -H 'X-Agentos: 1' 127.0.0.1:18772/__call/projects.Service/Snapshot -d '[]'
```

Agents' commands reach the same app: `AGENTOS_SESSION=<id> AGENTOS_SOCKET=$S/agentos.sock agentos note "x"`. The shell session (`ShellOpen`) runs `$SHELL -l`; set `SHELL=/bin/sh` and a throwaway `HOME` in tests so no profile of the owner runs. Stop it with SIGTERM so it stops its browsers.

## Driving the page

`scripts/cdp.mjs` opens a page in its own headless Chrome for Testing (its own temp profile, never the owner's) and runs steps: wait, click at a point, type, key, eval JavaScript, screenshot.

```sh
node scripts/cdp.mjs http://127.0.0.1:18772/ 1400 900 \
  '[{"wait":1500},{"eval":"document.title"},{"click":[700,400]},{"wait":300},{"shot":"aos-shot.png"}]'
```

A shot path that is not absolute lands in `$TMPDIR`. Read the screenshot to check what the page shows. Resize with the width and height arguments to check the layouts.

## Releases

`make release VERSION=v1.2.3` builds `bin/agentos-darwin-arm64.zip` and its `.sha256` the way the release workflow does, with the version stamped in `agentos version` and the bundle's Info.plist. To try the installer or `agentos update` without a real release, serve `bin/` over HTTP and point them at it with a throwaway `HOME`; never run either against the owner's `~/Applications` or `~/.local/bin`.

## What needs a human

The real Wails window: the title bar, drag and resize, native menus and shortcuts, the folder picker, the dock icon, clipboard through the system, and a launch from Finder (its PATH and locale). The browser mode covers everything else.
