package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"rmx/internal/rmux"

	"github.com/charmbracelet/lipgloss"
)

// defaultPreviewCmd previews a live rmux pane; the empty result for an exited
// session is why the cat picker uses catPreviewCmd instead.
const defaultPreviewCmd = "rmux capture-pane -p -t {1} -S -80 -E -1 2>/dev/null"

// pickSessions shows the rmux session inventory in fzf and returns selected names.
func pickSessions(ctx context.Context, sessions []rmux.Session, prompt string, multi bool) ([]string, error) {
	return pickSessionsPreview(ctx, sessions, prompt, multi, defaultPreviewCmd)
}

// pickSessionsPreview is pickSessions with a caller-chosen fzf preview command,
// so the cat picker can preview exited sessions via `rmx cat`.
func pickSessionsPreview(ctx context.Context, sessions []rmux.Session, prompt string, multi bool, preview string) ([]string, error) {
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no rmux sessions")
	}
	args := []string{
		"--ansi",
		"--prompt", prompt,
		"--height", "100%",
		"--reverse",
		"--delimiter", "\t",
		"--with-nth", "2..",
		"--preview", preview,
		"--preview-window", "right:60%",
	}
	if multi {
		args = append(args, "--multi")
	}

	fzfCmd := exec.CommandContext(ctx, "fzf", args...)
	fzfCmd.Stdin = strings.NewReader(strings.Join(sessionPickerLines(sessions, time.Now()), "\n"))
	fzfCmd.Stderr = os.Stderr

	out, err := fzfCmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return nil, ErrCancelled
		}
		if len(out) == 0 {
			return nil, ErrCancelled
		}
		return nil, fmt.Errorf("fzf failed: %w", err)
	}
	return parseFzfTargets(string(out))
}

func parseFzfTargets(out string) ([]string, error) {
	var targets []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		if idx := strings.Index(line, "\t"); idx >= 0 {
			targets = append(targets, line[:idx])
			continue
		}
		targets = append(targets, line)
	}
	if len(targets) == 0 {
		return nil, ErrCancelled
	}
	return targets, nil
}

func sessionPickerLines(sessions []rmux.Session, now time.Time) []string {
	nameStyle := lipglossStyle().Bold(true)
	dim := lipglossStyle().Faint(true)
	lines := make([]string, 0, len(sessions))
	for _, session := range sessions {
		line := fmt.Sprintf("%s\t%s\t%d windows\t%s\t%s",
			session.Name,
			nameStyle.Render(session.Name),
			session.Windows,
			relativeAge(session.LastActiveAt, now)+" "+dim.Render(formatTimestamp(session.LastActiveAt)),
			renderState(session),
		)
		lines = append(lines, line)
	}
	return lines
}

func relativeAge(then, now time.Time) string {
	if then.IsZero() {
		return "unknown"
	}
	if then.After(now) {
		return "now"
	}
	d := now.Sub(then)
	switch {
	case d < time.Minute:
		seconds := int(d.Seconds())
		if seconds < 1 {
			seconds = 1
		}
		return fmt.Sprintf("%ds ago", seconds)
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

func lipglossStyle() lipgloss.Style {
	return lipgloss.NewStyle()
}
