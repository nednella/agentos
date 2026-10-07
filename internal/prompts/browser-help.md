<!-- placeholders: none -->

agentos browser: drive the session's browser. Every command acts on this session's tab.

  open <url> [--front]        go to a page and wait for it to load; --front also raises the window
  back | reload               move in the page history
  url                         print the current address and title
  snapshot                    outline of the page: text, headings, and every control with a ref (e1, e2, ...)
  click <ref>                 click an element
  type <ref> <text> [--append]  replace the element's text (or add to it)
  press <key>                 Enter, Tab, Escape, ArrowDown, Backspace, or a chord like Control+a
  select <ref> <option>       pick an option of a select by its text or value
  hover <ref>                 move the pointer over an element
  scroll <up|down|ref> [px]   scroll the page, or bring an element into view
  wait <ms>                   pause
  wait-for <text> [--timeout <ms>]  wait until the text is on the page (10s by default)
  text [ref]                  plain text of the page or of an element
  eval <js>                   run JavaScript, print the JSON result
  console                     console errors and failed requests since the last call
  screenshot [--caption <text>] [--full] [ref]
                              save a PNG as evidence for this session and print its path

Refs come from the latest snapshot and stop working when you run snapshot again. After a
page change, run snapshot again before you click or type.
Use agentos show <file> [--caption <text>] for other evidence.
