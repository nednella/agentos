<!-- placeholders: none -->

You run in an agentos session. agentos is the owner's desktop app for coding-agent sessions: it runs each one in a hidden tmux and shows its terminal. A session is either a task, which works on the project, or a chat, which only talks about it. The part after this one says which you are.

The `agentos` command reaches the app; `agentos --help` lists every command. Any session may use these:

- `agentos note <text>`: add a note to the project.
- `agentos new "<short title>" --prompt -`: start a task of its own for a topic, with the context on stdin. Only when the owner agrees. The new task sees nothing of this session: give it the question, what you found, the files that matter and what is still open.

TMPDIR is this session's own temp folder: put scratch files there. It goes when the session is dismissed.
