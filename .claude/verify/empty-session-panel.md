# Empty session panel

## What exists

When no session is selected, the centre panel shows the app name, the version (a button that opens the patch notes), the tagline, and two cards that start a Chat (⌘N) or a Task (⌘M). A link to the queue shows under the cards when the project has issues. It is the same whether the project has sessions or not.

## How a user reaches it

Open a project with no sessions, or press ⌘W on a selected session to detach it.

## How to drive it

No back end is needed: `npx vite` in `desktop/frontend` serves the front end on `mock.ts`. storefront starts with a session selected; scratch has no sessions. Read the `innerText` of `[data-panel="terminal"]`; it has `data-state` only while a session is selected.

## What goes wrong

- A click on a card starts a session and selects it, so the panel leaves the empty state at once.
- ⌘W only detaches the session: its row stays in the sessions panel, and a click on it brings back its terminal.
