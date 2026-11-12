# Session views

## What exists

The session panel shows one session through four views, in a tab row under its header: Agent (the agent's terminal), Shell (the owner's own shell for that session), Browser and Media. The Shell view shows a hint with the session's folder (its worktree, else the project folder) until a click or Enter starts the shell; then it shows the shell. The Shell tab has a green dot while the shell runs, as the Browser tab has while the browser is open. Each session has at most one shell. It keeps running when the agent ends, goes when the row is dismissed or cleaned up, and passes to a session that resumes the ended one.

## How a user reaches it

Click a tab, or press ⌘⇧[ and ⌘⇧] to cycle the views. The palette has "Show agent" and "Show shell", and `agentos agent` and `agentos shell` in any terminal do the same. A double Esc in the shell goes back to the Agent view.

## How to drive it

The tab row is `[aria-label="Session views"]`, with `[role="tab"]` and `aria-selected`; the dot is a `.dot` titled "Shell running". The hint is the button with "Click or press Enter to open a shell here". In the real app the shell is the tmux session `<session id>-shell` on the test socket; `pwd` in it shows the folder. `exit` in the shell brings the hint back. On `mock.ts` the shell answers `pwd`, `ls` and `exit`.

## What goes wrong

- A command typed after a view change can land in the agent: check which tab is selected and where the focus is (`xterm-helper-textarea` in the shell, the hint button before it starts).
- The shell list in the app updates on its 2-second poll; the front end clears the dot at once on exit, and must not bring it back from an older sessions event.
- From inside an agentos session, `agentos <word>` reaches the owner's app unless `AGENTOS_SOCKET` points at the test app's socket.
