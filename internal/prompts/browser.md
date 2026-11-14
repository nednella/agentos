<!-- placeholders: none -->

This session has its own browser window with one tab. `agentos browser help` has the rules; the ones that matter most:

- Start with `agentos browser open <url>`. It makes your window and loads the page. Until then every `agent-browser` command fails.
- The window opens behind the owner's other windows. When the owner should look at the page, such as a design you made, add `--front` to raise it. Leave it off for your own checks and screenshots.
- Drive the tab with `agent-browser` in Bash. It is set up for your tab: never pass `--session`, `--cdp` or `--pin-tab`, and never run `agent-browser connect`, `tab`, `window`, `inspect` or `close`. They take another session's tab or bring the browser in front of the owner's windows.
- File what you built with `agentos browser screenshot --caption "<text>"`, and run `agentos browser close` when you are done.

agent-browser commands:

- `agent-browser open <url>`: go to a page in your tab. A link that opens a new tab opens outside your reach, so open its address instead.
- `agent-browser snapshot -i -c`: the interactive elements, each with a ref such as `@e3`. Add `-s "<css>"` to scope it to part of the page. After the first snapshot, `snapshot -i -c --delta` prints only what changed.
- `agent-browser click @e3`, `fill @e3 "text"`, `select @e3 "label"`, `check @e3`, `press Enter`: act on refs. Take a new snapshot after the page changes; old refs can go stale.
- `agent-browser wait @e3` or `wait --text "<text>"`: wait for what you expect after an action. Wait only for a ref or for text you saw in a snapshot, because a wait that does not match blocks for 5 seconds. Do not use fixed waits.
- `agent-browser get text @e3`, `get url`, `read`: read text without refs.
- `agent-browser screenshot`, then Read the file it prints. Give no path: a relative path is saved in another folder. Add `--annotate` to label elements with their refs, `--full` for the whole page. Use screenshots for what a snapshot cannot show, such as canvas, images and layout.
- `agent-browser eval "<js>"`: run JavaScript in the page and print the result.
- Several commands at once: `agent-browser batch "open <url>" "snapshot -i -c"`.
- Performance and debugging: `agent-browser vitals [url]` for Core Web Vitals (`--json` for all of it), `network requests [--filter <text>]`, `console`, `errors`, and `trace start`, then `trace stop "$TMPDIR/trace.json"` for a DevTools trace.
