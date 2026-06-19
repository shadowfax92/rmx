package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"rmx/internal/rmux"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

const defaultTailInterval = 5 * time.Second

var tailPrefixColors = []lipgloss.Color{
	clrCyan,
	clrHiGreen,
	clrYellow,
	lipgloss.Color("13"),
	lipgloss.Color("14"),
	clrRed,
	lipgloss.Color("12"),
}

type tailCaptureFunc func(context.Context, string) (string, error)

type tailState struct {
	sessions []rmux.Session
	previous map[string]string
}

func init() {
	rootCmd.AddCommand(tailCmd)
}

var tailCmd = &cobra.Command{
	Use:         "tail [session...]",
	Annotations: map[string]string{"group": "Output:"},
	Short:       "Follow output from rmux sessions",
	Args:        cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := rmux.DefaultClient()
		sessions, err := sessionsForTargets(cmd.Context(), client, args, "tail > ")
		if err != nil {
			return err
		}
		return tailSessions(cmd.Context(), cmd.OutOrStdout(), sessions, defaultTailInterval, func(ctx context.Context, name string) (string, error) {
			return client.CapturePane(ctx, name, defaultCaptureLines)
		})
	},
}

// tailSessions follows selected sessions by polling rmux captures until the context ends.
func tailSessions(ctx context.Context, out io.Writer, sessions []rmux.Session, interval time.Duration, capture tailCaptureFunc) error {
	state, err := newTailState(ctx, sessions, capture)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := state.poll(ctx, out, capture); err != nil {
				return err
			}
		}
	}
}

// newTailState captures the initial baseline so tail only prints future output.
func newTailState(ctx context.Context, sessions []rmux.Session, capture tailCaptureFunc) (*tailState, error) {
	previous := make(map[string]string, len(sessions))
	for _, session := range sessions {
		text, err := capture(ctx, session.Name)
		if err != nil {
			return nil, err
		}
		previous[session.Name] = text
	}
	return &tailState{sessions: sessions, previous: previous}, nil
}

func (s *tailState) poll(ctx context.Context, out io.Writer, capture tailCaptureFunc) error {
	for index, session := range s.sessions {
		current, err := capture(ctx, session.Name)
		if err != nil {
			return err
		}
		chunk := appendedCapture(s.previous[session.Name], current)
		s.previous[session.Name] = current
		renderTailChunk(out, session.Name, index, chunk)
	}
	return nil
}

func appendedCapture(previous string, current string) string {
	if current == previous {
		return ""
	}
	if strings.HasPrefix(current, previous) {
		return strings.TrimPrefix(current, previous)
	}
	overlap := longestSuffixPrefixOverlap(previous, current)
	if overlap > 0 {
		return current[overlap:]
	}
	return current
}

func longestSuffixPrefixOverlap(previous string, current string) int {
	limit := min(len(previous), len(current))
	for length := limit; length > 0; length-- {
		if strings.HasSuffix(previous, current[:length]) {
			return length
		}
	}
	return 0
}

func renderTailChunk(out io.Writer, name string, index int, chunk string) {
	chunk = strings.TrimPrefix(chunk, "\n")
	chunk = strings.TrimRight(chunk, "\n")
	if chunk == "" {
		return
	}

	prefix := tailPrefixStyle(index).Render("[" + name + "]")
	for _, line := range strings.Split(chunk, "\n") {
		fmt.Fprintf(out, "%s %s\n", prefix, line)
	}
}

func tailPrefixStyle(index int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(tailPrefixColor(index))
}

func tailPrefixColor(index int) lipgloss.Color {
	return tailPrefixColors[index%len(tailPrefixColors)]
}
