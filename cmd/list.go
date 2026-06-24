package cmd

import (
	"fmt"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:         "ls",
	Aliases:     []string{"list", "l"},
	Annotations: map[string]string{"group": "Sessions:"},
	Short:       "List rmux sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		now := time.Now()
		live, err := rmux.DefaultClient().ListSessions(cmd.Context())
		if err != nil {
			return err
		}
		exited := loadExited(store.Default(), now, cmd.ErrOrStderr())
		sessions := mergeSessions(live, exited)
		if len(sessions) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No rmux sessions.")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), renderSessionTable(sessions, now))
		return nil
	},
}

func renderSessionTable(sessions []rmux.Session, now time.Time) string {
	dim := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(clrHiGreen)

	rows := make([][]string, 0, len(sessions))
	for _, session := range sessions {
		rows = append(rows, []string{
			nameStyle.Render(session.Name),
			fmt.Sprintf("%d", session.Windows),
			activeStyle.Render(formatTimestamp(session.LastActiveAt)),
			relativeAge(session.LastActiveAt, now),
			dim.Render(session.CreatedAt.Format("Jan 02 15:04")),
			renderState(session),
		})
	}

	return table.New().
		Border(lipgloss.HiddenBorder()).
		Headers("SESSION", "WIN", "LAST ACTIVE", "AGE", "CREATED", "STATE").
		Rows(rows...).
		StyleFunc(func(row, col int) lipgloss.Style {
			s := lipgloss.NewStyle().PaddingRight(2)
			if row == table.HeaderRow {
				return s.Bold(true).Faint(true)
			}
			return s
		}).
		String()
}

// renderState styles a session's STATE cell: exited (red), attached (yellow),
// or detached (dim). Shared by the list table and the fzf picker.
func renderState(session rmux.Session) string {
	switch {
	case session.Exited:
		return lipgloss.NewStyle().Foreground(clrRed).Render("exited")
	case session.Attached:
		return lipgloss.NewStyle().Foreground(clrYellow).Render("attached")
	default:
		return lipgloss.NewStyle().Faint(true).Render("detached")
	}
}

const timestampLayout = "Jan 02 15:04"

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format(timestampLayout)
}
