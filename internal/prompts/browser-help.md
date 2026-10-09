<!-- placeholders: none -->

agentos browser: your session's browser window.

  open <url> [--front]        open a page in your window, wait for it to load, print its title and your page label; the window opens behind the owner's; --front raises it: use it every time the owner should look at the page (a design, or to log in), never for your own checks
  tab <url>                   open a page in a new tab of your window, wait for it to load, print its title and label
  screenshot [--caption <text>] [--full]  save a PNG as evidence for this session and print its path (of the tab that is showing)
  close                       close your window and all its tabs; run it when you are done with the browser

Everything else on the page goes through the `browser` MCP tools:

  1. Run `agentos browser open <url>` first. It makes your window, titled with the label it prints.
  2. Every page of yours has your label at the start of its title. Before each burst of tool calls, run list_pages and pick the page by that label and its URL. pageIds change, so do not keep them.
  3. Open more tabs with `agentos browser tab <url>`, or by following links.
  4. new_page and close_page are unavailable: they could open or close pages in other sessions' windows.
  5. When a tool reports a dialog, call handle_dialog before anything else. agentos dismisses a dialog left open for 5 seconds.

Use agentos show <file> [--caption <text>] for other evidence.
