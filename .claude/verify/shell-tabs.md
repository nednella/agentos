# Shell tabs

## What exists

The shell drawer at the bottom of the window has a row of tabs, one for each shell of the project. The first tab is the project shell: pinned at the left, labelled with the project name and a folder icon, with no close button. The others are "Shell 2", "Shell 3" by position, each with a close button that shows on hover and on the selected tab. A `+` after the tabs opens a new shell, and a chevron at the right collapses and opens the drawer. When the tabs do not fit, the part between the pinned tab and `+` scrolls sideways, the tabs fade out at each edge that hides more of them, and a thumb in the row's bottom border shows which part of the row is in view. The collapsed drawer shows the same row with no terminal. In the narrow layout (below 640px), the Shell panel shows the same row above the terminal, with no chevron.

## How a user reaches it

The drawer is always at the bottom of the window. ⌘S collapses and opens it, ⌘⇧S puts the focus in the shell, and a double Esc leaves the shell. In the narrow layout, the Shell tab in the bottom bar shows it.

## How to drive it

`npx vite` in `desktop/frontend` serves the front end on `mock.ts`, which starts with no shell. The real app in a browser (see the README) runs real shells in tmux on the test socket. The row is `.shell-tabs` inside `[data-panel="shell"]`; tabs are `[role="tab"]` with `aria-selected`, and the scroll part is `.shell-tabs-scroll`, with `data-hidden-left` and `data-hidden-right` set when tabs are hidden at that edge; the thumb is the `span` after it. Add 15 or more shells with `+` to make the row scroll. A shell ended on the tmux socket drops its tab, which gives the empty state after a reload.

## What goes wrong

- Under `vite dev`, React StrictMode mounts each terminal twice, so the focus after a tab click or `+` does not land in the shell. Check the focus in the real app, which is a production build.
- A page in a background window has no focus, so `data-focused` does not show on the panel. Set it by hand to check the focus line and tint.
- The focus line must run unbroken over the tabs, and the selected tab must keep the terminal background under it.
- A selected tab must stop clear of the 1.5rem fade, not under it.
- The row is as high as the sessions header, open and collapsed, and the thumb sits on its bottom border, not above it.
