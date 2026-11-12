# Verify map

Start the real app in a browser with the isolation env from `docs/testing.md`: `AGENTOS_HTTP=127.0.0.1:PORT` on a bare copy of `bin/agentos-dev.app/Contents/MacOS/agentos`. The app starts its services only when a page connects, so open the page before you call `RefreshPRs`.

- [Session row status](session-row-status.md): the status line of PR and branch chips, its hover card, and the same line in the panel header.
- [Collapsed groups](collapsed-groups.md): the Chats and Tasks groups of the sessions panel and the sections of the queue, and which of them each project keeps closed.
- [Empty session panel](empty-session-panel.md): what the centre panel shows when no session is selected, with or without sessions in the project.
- [Shell tabs](shell-tabs.md): the tab row of the shell drawer, open and collapsed, and the same row in the narrow layout.
- [Session views](session-views.md): the Agent, Shell, Browser and Media views of the session panel, and the session's own shell.
