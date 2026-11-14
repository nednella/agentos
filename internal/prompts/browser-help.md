<!-- placeholders: none -->

agentos browser: your session's browser window.

  open <url> [--front]        open a page in your window, wait for it to load, print its title; the window opens behind the owner's; --front raises it: use it every time the owner should look at the page (a design, or to log in), never for your own checks
  screenshot [--caption <text>] [--full]  save a PNG as evidence for this session and print its path (of the tab that is showing)
  close                       close your window; run it when you are done with the browser

Everything else on the page goes through `agent-browser` in Bash:

  1. Run `agentos browser open <url>` first. It makes your window and points agent-browser at its tab. Before that, every agent-browser command fails.
  2. agent-browser is set up for your tab. Never pass --session, --cdp or --pin-tab, and never run agent-browser connect, tab, window, inspect or close: they take another session's tab or bring the browser in front of the owner's windows.
  3. A link that opens a new tab opens outside your reach: open its address with `agent-browser open` instead.
  4. Run `agent-browser screenshot` with no path and Read the file it prints. A relative path is saved in another folder.
  5. agent-browser accepts alert and beforeunload dialogs itself. Answer a confirm or prompt with `agent-browser dialog accept` or `dialog dismiss`; agentos dismisses a dialog left open for 5 seconds.
  6. A wait times out after 5 seconds.

Use agentos show <file> [--caption <text>] for other evidence.
