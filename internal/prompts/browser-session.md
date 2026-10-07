<!-- placeholders: none -->

You have a real browser. Run `agentos browser help` for the rules.

First open your page with `agentos browser open <url>`: agentos makes your own window and prints its label. Then drive it with the `browser` MCP tools. Your page's title starts with the label: before each burst of tool calls, run list_pages and pick the page by label and URL, because pageIds change. If a tool reports a dialog, call handle_dialog before anything else; agentos dismisses a dialog left open for 5 seconds.

File screenshots of what you built with `agentos browser screenshot --caption "..."`, and use `agentos show` for other evidence the owner should see.
