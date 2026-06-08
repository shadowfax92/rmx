package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"wrapux/internal/rmux"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const defaultCaptureLines = 80

var captureLines int

func init() {
	captureCmd.Flags().IntVarP(&captureLines, "lines", "l", defaultCaptureLines, "Capture the last N lines from each session")
	rootCmd.AddCommand(captureCmd)
}

var captureCmd = &cobra.Command{
	Use:         "capture [session...]",
	Aliases:     []string{"cap"},
	Annotations: map[string]string{"group": "Output:"},
	Short:       "Capture output from rmux sessions",
	Args:        cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := rmux.DefaultClient()
		sessions, err := sessionsForCapture(cmd.Context(), client, args)
		if err != nil {
			return err
		}
		return renderCaptures(cmd.Context(), cmd.OutOrStdout(), sessions, func(ctx context.Context, name string) (string, error) {
			return client.CapturePane(ctx, name, captureLines)
		})
	},
}

func sessionsForCapture(ctx context.Context, client rmux.Client, args []string) ([]rmux.Session, error) {
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]rmux.Session, len(sessions))
	for _, session := range sessions {
		byName[session.Name] = session
	}
	if len(args) == 0 {
		targets, err := pickSessions(ctx, sessions, "capture > ", true)
		if err != nil {
			return nil, err
		}
		args = targets
	}

	selected := make([]rmux.Session, 0, len(args))
	for _, name := range args {
		session, ok := byName[name]
		if !ok {
			session = rmux.Session{Name: name}
		}
		selected = append(selected, session)
	}
	return selected, nil
}

// renderCaptures prints each selected session with a colored header and stable separator.
func renderCaptures(ctx context.Context, out io.Writer, sessions []rmux.Session, capture func(context.Context, string) (string, error)) error {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(clrCyan)
	metaStyle := lipgloss.NewStyle().Faint(true)
	for idx, session := range sessions {
		if idx > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintln(out, headerStyle.Render("wrapux capture: "+session.Name))
		if !session.LastActiveAt.IsZero() {
			fmt.Fprintln(out, metaStyle.Render("last active "+relativeAge(session.LastActiveAt, time.Now())))
		}
		fmt.Fprintln(out, metaStyle.Render(strings.Repeat("-", 72)))

		text, err := capture(ctx, session.Name)
		if err != nil {
			return err
		}
		if text == "" {
			fmt.Fprintln(out, metaStyle.Render("(empty)"))
			continue
		}
		fmt.Fprint(out, strings.TrimRight(text, "\n"))
		fmt.Fprintln(out)
	}
	return nil
}
