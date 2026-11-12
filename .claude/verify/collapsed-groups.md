# Collapsed groups

## What exists

The sessions panel lists sessions in two groups, Chats and Tasks. The queue lists issues in the project's sections, with the issues no section matches in a last group that starts closed. Each group has a header that opens and closes it. Each project remembers its own closed groups, in the browser's local storage.

## How a user reaches it

The sessions panel on the right (⌘D) and the queue on the left. Click a group header, or move the cursor to it and press Enter or Space; Left closes it and Right opens it.

## How to drive it

No back end is needed: `npx vite` in `desktop/frontend` serves the front end on `mock.ts`, which has three projects. Close a group, switch project from the top bar, and read `aria-expanded` on the `.section-head` buttons inside `[data-panel="sessions"]` and `[data-panel="sidebar"]`. Only storefront has queue sections at the start; *Set up* in agentos gives it an Inbox, the same name as storefront's.

## What goes wrong

- The peek overlay is a second copy of the list. It reads the stored groups when it opens, so check it after a toggle in the docked panel.
- After a reload the mock is back in its first project, so switch project again to check the others.
- The queue is empty for a moment after a project switch, so wait for the issues before you read its headers.
