<!-- placeholders: none -->

This session has its own browser window. `agentos browser help` has the rules; the ones that matter most:

- Start with `agentos browser open <url>`. It makes your window and prints its label.
- The window opens behind the owner's other windows. When the owner should look at the page, such as a design you made, add `--front` to raise it. Leave it off for your own checks and screenshots.
- Drive the page with the `browser` MCP tools. Other sessions share the browser, so before each burst of calls run list_pages and pick your page by its label, which starts its title, and its URL. pageIds change.
- Open more tabs with `agentos browser tab <url>`. new_page and close_page are unavailable.
- When a tool reports a dialog, call handle_dialog before anything else; agentos dismisses one left open for 5 seconds.
- File what you built with `agentos browser screenshot --caption "<text>"`, and run `agentos browser close` when you are done.
