<!-- placeholders: none -->

agentos browser: your session's browser window.

  open <url> [--front]        open a page in your window, wait for it to load, print its title and your page label;
                              --front also raises the window, for when the owner must see it (to log in, say)
  screenshot [--caption <text>] [--full]
                              save a PNG as evidence for this session and print its path

Everything else on the page goes through the `browser` MCP tools:

  1. Run `agentos browser open <url>` first. It makes your window, titled with the label it prints.
  2. Your page's title starts with your label. Before each burst of tool calls, run list_pages and pick the page by
     that label and its URL. pageIds change, so do not keep them.
  3. When a tool reports a dialog, call handle_dialog before anything else. agentos dismisses a dialog left open for 5 seconds.

Use agentos show <file> [--caption <text>] for other evidence.
