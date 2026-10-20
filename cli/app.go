package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/control"
)

// appCmd is a command that asks the running app to do something and prints its answer.
func appCmd(use, short, group string, args cobra.PositionalArgs) *cobra.Command {
	name, _, _ := strings.Cut(use, " ")
	return &cobra.Command{
		Use: use, Short: short, GroupID: group, Args: args,
		RunE: func(cmd *cobra.Command, args []string) error {
			return askApp(cmd, control.Request{Cmd: name, Args: args})
		},
	}
}

func newAppCmds() []*cobra.Command {
	return []*cobra.Command{
		newIssueCmd(),
		newNewCmd(),
		appCmd("open <n|title>", "Show a session in the terminal", groupWork, cobra.MinimumNArgs(1)),
		appCmd("next", "Show the session that needs you most", groupWork, cobra.NoArgs),
		appCmd("refresh", "Reload the issues and the pull requests", groupWork, cobra.NoArgs),
		appCmd("pr [n]", "Show the pull requests of the sessions", groupWork, cobra.MaximumNArgs(1)),
		appCmd("cleanup [n]", "Clean up after session n, or list what waits for it", groupWork, cobra.MaximumNArgs(1)),
		appCmd("queue", "Show the queue", groupViews, cobra.NoArgs),
		appCmd("notes", "Show the notes", groupViews, cobra.NoArgs),
		appCmd("evidence", "Show the evidence of the session on screen", groupViews, cobra.NoArgs),
		appCmd("term", "Show the terminal", groupViews, cobra.NoArgs),
		appCmd("news", "Show the news", groupViews, cobra.NoArgs),
		appCmd("filter [query]", "Filter the queue, for example: filter @me type:bug", groupViews, nil),
		newProjectCmd(),
	}
}

func newProjectCmd() *cobra.Command {
	project := appCmd("project [name]", "List projects, or switch to one", groupProject, nil)
	project.AddCommand(&cobra.Command{
		Use: "add [path]", Short: "Add a folder as a project (the current folder by default)", Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) == 1 {
				path = args[0]
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("finding %s: %w", path, err)
			}
			if _, err := os.Stat(abs); err != nil {
				return fmt.Errorf("checking %s: %w", abs, err)
			}
			return askApp(cmd, control.Request{Cmd: "project", Args: []string{"add", abs}})
		},
	}, &cobra.Command{
		Use: "remove <name>", Short: "Forget a project", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return askApp(cmd, control.Request{Cmd: "project", Args: []string{"remove", args[0]}})
		},
	})
	return project
}

func newIssueCmd() *cobra.Command {
	var prompt, agent string
	cmd := &cobra.Command{
		Use: "issue <number...>", Short: "Start a session for each issue (--prompt: context for one issue, - reads it from stdin; --agent: its provider)", GroupID: groupWork,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if prompt != "" && len(args) > 1 {
				return errors.New("--prompt takes one issue")
			}
			text, err := readPrompt(cmd, prompt)
			if err != nil {
				return err
			}
			return askApp(cmd, control.Request{Cmd: "issue", Args: args, Opts: map[string]string{"prompt": text, "agent": agent}})
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "context for the new session, from the session that starts it")
	cmd.Flags().StringVar(&agent, "agent", "", "the provider the new session runs, instead of the default: claude")
	return cmd
}

func newNewCmd() *cobra.Command {
	var prompt, agent string
	var chat bool
	cmd := &cobra.Command{
		Use: "new [title]", Short: "Start a session (--prompt: what it starts with, - reads it from stdin; --chat: a chat; --agent: its provider)", GroupID: groupWork,
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := readPrompt(cmd, prompt)
			if err != nil {
				return err
			}
			opts := map[string]string{"prompt": text, "agent": agent}
			if chat {
				opts["chat"] = "true"
			}
			return askApp(cmd, control.Request{Cmd: "new", Args: args, Opts: opts})
		},
	}
	cmd.Flags().StringVar(&prompt, "prompt", "", "the first message of the new session")
	cmd.Flags().BoolVar(&chat, "chat", false, "start a chat, which only reads, instead of a task")
	cmd.Flags().StringVar(&agent, "agent", "", "the provider the new session runs, instead of the default: claude")
	return cmd
}

// maxPrompt keeps a prompt, with the agent's command line or as typed text, under the 16 KB tmux takes in one command.
const maxPrompt = 6 << 10

// readPrompt is the value of a --prompt flag, read from stdin when it is "-".
func readPrompt(cmd *cobra.Command, prompt string) (string, error) {
	if prompt == "-" {
		data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 1<<20))
		if err != nil {
			return "", fmt.Errorf("reading the prompt: %w", err)
		}
		prompt = string(data)
	}
	prompt = strings.TrimSpace(prompt)
	if len(prompt) > maxPrompt {
		return "", fmt.Errorf("the prompt is %d bytes, more than the %d a session takes: put the long parts in a file and give its path", len(prompt), maxPrompt)
	}
	return prompt, nil
}
