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
	Aliases:     []string{"follow"},
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
			return client.CapturePaneHistory(ctx, name)
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
		if err := renderTailChunk(out, session.Name, index, chunk); err != nil {
			return err
		}
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
	overlap := trustedOverlap(previous, current)
	if overlap > 0 {
		return current[overlap:]
	}
	return current
}

// trustedOverlap accepts rollover matches only when complete lines make a reset unlikely.
func trustedOverlap(previous string, current string) int {
	overlap := longestSuffixPrefixOverlap(previous, current)
	if overlap == 0 || !isCompleteLineOverlap(previous, current, overlap) {
		return 0
	}
	if overlapLineCount(current[:overlap]) < 2 {
		return 0
	}
	return overlap
}

func isCompleteLineOverlap(previous string, current string, overlap int) bool {
	previousStart := len(previous) - overlap
	if previousStart > 0 && previous[previousStart-1] != '\n' {
		return false
	}
	return overlap == len(current) || current[overlap] == '\n'
}

func overlapLineCount(text string) int {
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

// longestSuffixPrefixOverlap stays linear when full-history snapshots roll over.
func longestSuffixPrefixOverlap(previous string, current string) int {
	if previous == "" || current == "" {
		return 0
	}
	pattern := current
	if len(pattern) > len(previous) {
		pattern = pattern[:len(previous)]
	}

	prefix := make([]int, len(pattern))
	for i := 1; i < len(pattern); i++ {
		j := prefix[i-1]
		for j > 0 && pattern[i] != pattern[j] {
			j = prefix[j-1]
		}
		if pattern[i] == pattern[j] {
			j++
		}
		prefix[i] = j
	}

	match := 0
	for i := 0; i < len(previous); i++ {
		for match > 0 && (match == len(pattern) || previous[i] != pattern[match]) {
			match = prefix[match-1]
		}
		if previous[i] == pattern[match] {
			match++
		}
	}
	return match
}

func renderTailChunk(out io.Writer, name string, index int, chunk string) error {
	lines := tailChunkLines(chunk)
	if len(lines) == 0 {
		return nil
	}

	prefix := tailPrefixStyle(index).Render("[" + name + "]")
	for _, line := range lines {
		if _, err := fmt.Fprintf(out, "%s %s\n", prefix, line); err != nil {
			return err
		}
	}
	return nil
}

func tailChunkLines(chunk string) []string {
	chunk = strings.TrimPrefix(chunk, "\n")
	if chunk == "" {
		return nil
	}
	if strings.HasSuffix(chunk, "\n") {
		chunk = strings.TrimSuffix(chunk, "\n")
	}
	if chunk == "" {
		return []string{""}
	}
	return strings.Split(chunk, "\n")
}

func tailPrefixStyle(index int) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(tailPrefixColor(index))
}

func tailPrefixColor(index int) lipgloss.Color {
	return tailPrefixColors[index%len(tailPrefixColors)]
}
