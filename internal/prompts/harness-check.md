<!-- placeholders: none -->

Review this project's Claude Code harness and propose improvements. Change no files.

1. Read the harness: the CLAUDE.md files, the .claude folder (commands, agents, skills, settings.json with its hooks and permission rules), and any scripts they call.
2. Run "agentos stats --days 30". It shows what interrupts the owner most: the permission prompts and questions that stopped an agent until he answered.
3. Check what Claude Code offers today. Run "claude --help" and read the docs at https://docs.claude.com/en/docs/claude-code. Look for features this harness could use, and for anything it uses that is outdated or deprecated.
4. Propose at most seven changes, ranked by how much prompting or risk each one removes. Make each one specific: which file, what change, and why. Prefer permission rules for the commands that interrupt the owner most.
5. Save each proposal with: agentos note "<proposal>". One note per proposal, best first.
6. Finish with a short summary of what you found.

Do not edit, create or delete any file. The only thing you write is the notes.
