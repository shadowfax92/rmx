package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const defaultCaptureLines = 80

// catPreviewCmd previews a cat target through rmx itself, so the picker shows
// live panes and replayed exited output the same way.
const catPreviewCmd = "rmx cat -l 80 {1} 2>/dev/null"

var captureLines int

// captureFunc returns a session's output — live from rmux or replayed from the
// exited-session store.
type captureFunc func(context.Context, string) (string, error)

func init() {
	captureCmd.Flags().IntVarP(&captureLines, "lines", "l", defaultCaptureLines, "Print the last N lines from each session")
	rootCmd.AddCommand(captureCmd)
}

var captureCmd = &cobra.Command{
	Use:         "cat [session...]",
	Aliases:     []string{"capture", "cap"},
	Annotations: map[string]string{"group": "Output:"},
	Short:       "Print output from rmux sessions",
	Args:        cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		sessions, capture, err := capturePlan(cmd.Context(), rmux.DefaultClient(), store.Default(), time.Now(), args, cmd.ErrOrStderr())
		if err != nil {
			return err
		}
		return renderCaptures(cmd.Context(), cmd.OutOrStdout(), sessions, capture)
	},
}

// capturePlan resolves the sessions to print and a capture function that reads
// live panes from rmux and exited sessions from the store. Exited sessions join
// the picker so you can replay what happened after a session is gone.
func capturePlan(ctx context.Context, client rmux.Client, st store.Store, now time.Time, args []string, warn io.Writer) ([]rmux.Session, captureFunc, error) {
	live, err := client.ListSessions(ctx)
	if err != nil {
		return nil, nil, err
	}
	exited := loadExited(st, now, warn)

	byName := make(map[string]rmux.Session, len(live)+len(exited))
	for _, session := range live {
		byName[session.Name] = session
	}
	exitedNames := make(map[string]bool, len(exited))
	for _, rec := range exited {
		if _, isLive := byName[rec.Name]; isLive {
			continue
		}
		exitedNames[rec.Name] = true
		byName[rec.Name] = exitedToSession(rec)
	}

	names := args
	if len(names) == 0 {
		picked, err := pickSessionsPreview(ctx, mergeSessions(live, exited), "cat > ", true, catPreviewCmd)
		if err != nil {
			return nil, nil, err
		}
		names = picked
	}

	selected := make([]rmux.Session, 0, len(names))
	for _, name := range names {
		session, ok := byName[name]
		if !ok {
			session = rmux.Session{Name: name}
		}
		selected = append(selected, session)
	}

	capture := func(ctx context.Context, name string) (string, error) {
		if exitedNames[name] {
			text, _, err := st.Output(name)
			return text, err
		}
		return client.CapturePane(ctx, name, captureLines)
	}
	return selected, capture, nil
}

func sessionsForTargets(ctx context.Context, client rmux.Client, args []string, prompt string) ([]rmux.Session, error) {
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]rmux.Session, len(sessions))
	for _, session := range sessions {
		byName[session.Name] = session
	}
	if len(args) == 0 {
		targets, err := pickSessions(ctx, sessions, prompt, true)
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
		fmt.Fprintln(out, headerStyle.Render("rmx cat: "+session.Name))
		if !session.LastActiveAt.IsZero() {
			label := "last active "
			if session.Exited {
				label = "exited "
			}
			fmt.Fprintln(out, metaStyle.Render(label+relativeAge(session.LastActiveAt, time.Now())))
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
