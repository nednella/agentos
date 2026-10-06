package cli

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nednella/agentos/internal/control"
	"github.com/nednella/agentos/internal/prompts"
)

// The commands in this file are for agents running in a session. Each one asks
// the running desktop app to do the work and prints what the app answers.

func newNoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "note <text>",
		Short:   "Add a note to this project",
		GroupID: groupSession,
		RunE: func(cmd *cobra.Command, args []string) error {
			text := strings.Join(args, " ")
			if len(args) == 0 {
				data, err := io.ReadAll(io.LimitReader(cmd.InOrStdin(), 1<<20))
				if err != nil {
					return fmt.Errorf("reading the note: %w", err)
				}
				text = string(data)
			}
			return askApp(cmd, control.Request{Cmd: "note", Args: []string{text}})
		},
	}
}

func newTrackCmd() *cobra.Command {
	var branch string
	cmd := &cobra.Command{
		Use:     "track --branch <name>",
		Short:   "Tell the app which branch this session works on",
		GroupID: groupSession,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return askApp(cmd, control.Request{Cmd: "track", Opts: map[string]string{"branch": branch}})
		},
	}
	cmd.Flags().StringVar(&branch, "branch", "", "the branch name")
	_ = cmd.MarkFlagRequired("branch")
	return cmd
}

func newStatsCmd() *cobra.Command {
	var days int
	var asJSON, open bool
	cmd := &cobra.Command{
		Use:     "stats",
		Short:   "Print what interrupted the owner most in this project (--open: show the view)",
		GroupID: groupViews,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := map[string]string{"days": strconv.Itoa(days)}
			if asJSON {
				opts["json"] = "1"
			}
			if open {
				opts["open"] = "1"
			}
			return askApp(cmd, control.Request{Cmd: "stats", Opts: opts})
		},
	}
	cmd.Flags().IntVar(&days, "days", 7, "how many days to look back")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON")
	cmd.Flags().BoolVar(&open, "open", false, "show the stats view in the app")
	return cmd
}

func newShowCmd() *cobra.Command {
	var caption, text string
	cmd := &cobra.Command{
		Use:     "show [file]",
		Short:   "Show the owner evidence: an image, a text file, or --text \"words\"",
		GroupID: groupSession,
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			req := control.Request{Cmd: "show", Opts: map[string]string{"caption": caption, "text": text}}
			switch {
			case len(args) == 1 && text == "":
				path, err := filepath.Abs(args[0])
				if err != nil {
					return fmt.Errorf("finding %s: %w", args[0], err)
				}
				req.Args = []string{path}
			case len(args) == 0 && text != "":
			default:
				return errors.New("give a file or --text, not both")
			}
			return askApp(cmd, req)
		},
	}
	cmd.Flags().StringVar(&caption, "caption", "", "what the evidence shows")
	cmd.Flags().StringVar(&text, "text", "", "a text card instead of a file")
	return cmd
}

func newBrowserCmd() *cobra.Command {
	return &cobra.Command{
		Use:                "browser [command] [args]",
		Short:              "Show the browser view, or drive this session's browser (agentos browser help)",
		GroupID:            groupViews,
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "help" || args[0] == "-h" || args[0] == "--help") {
				fmt.Fprintln(cmd.OutOrStdout(), prompts.BrowserHelp())
				return nil
			}
			if len(args) == 0 {
				return askApp(cmd, control.Request{Cmd: "browser"})
			}
			pos, opts, err := parseFlags(args[1:], []string{"caption", "timeout"}, []string{"full", "append"})
			if err != nil {
				return err
			}
			return askApp(cmd, control.Request{Cmd: "browser", Args: append([]string{args[0]}, pos...), Opts: opts, TimeoutMs: browserTimeout(args[0], pos, opts)})
		},
	}
}

// browserTimeout is how long the app may spend on a browser command: a page load
// can take 30 seconds, and a wait takes the time it was asked for on top.
func browserTimeout(sub string, pos []string, opts map[string]string) int {
	waited := opts["timeout"]
	if sub == "wait" && len(pos) > 0 {
		waited = pos[0]
	}
	if ms, err := strconv.Atoi(waited); err == nil && ms > 0 {
		return min(ms+15_000, 115_000)
	}
	return 75_000
}

func newDigestCmd() *cobra.Command {
	var run bool
	digest := &cobra.Command{
		Use:     "digest",
		Short:   "Show the digest view (--run: start a run)",
		GroupID: groupViews,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := map[string]string{}
			if run {
				opts["run"] = "1"
			}
			return askApp(cmd, control.Request{Cmd: "digest", Opts: opts})
		},
	}
	digest.Flags().BoolVar(&run, "run", false, "start a digest run")
	var title, why, url, source string
	add := &cobra.Command{
		Use:    "add --title <t> --why <w> --url <u> --source <s>",
		Short:  "File one item in the digest (used by the digest run)",
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			opts := map[string]string{"title": title, "why": why, "url": url, "source": source}
			return askApp(cmd, control.Request{Cmd: "digest-add", Opts: opts})
		},
	}
	add.Flags().StringVar(&title, "title", "", "headline")
	add.Flags().StringVar(&why, "why", "", "why it matters to this project")
	add.Flags().StringVar(&url, "url", "", "link")
	add.Flags().StringVar(&source, "source", "", "where it came from")
	digest.AddCommand(add)
	return digest
}
