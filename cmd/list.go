package cmd

import (
	"fmt"
	"time"

	"rmx/internal/rmux"

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
		sessions, err := rmux.DefaultClient().ListSessions(cmd.Context())
		if err != nil {
			return err
		}
		if len(sessions) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No rmux sessions.")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), renderSessionTable(sessions, time.Now()))
		return nil
	},
}

func renderSessionTable(sessions []rmux.Session, now time.Time) string {
	dim := lipgloss.NewStyle().Faint(true)
	nameStyle := lipgloss.NewStyle().Bold(true)
	activeStyle := lipgloss.NewStyle().Foreground(clrHiGreen)

	rows := make([][]string, 0, len(sessions))
	for _, session := range sessions {
		attached := dim.Render("detached")
		if session.Attached {
			attached = lipgloss.NewStyle().Foreground(clrYellow).Render("attached")
		}
		rows = append(rows, []string{
			nameStyle.Render(session.Name),
			fmt.Sprintf("%d", session.Windows),
			activeStyle.Render(formatTimestamp(session.LastActiveAt)),
			relativeAge(session.LastActiveAt, now),
			dim.Render(session.CreatedAt.Format("Jan 02 15:04")),
			attached,
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

const timestampLayout = "Jan 02 15:04"

func formatTimestamp(value time.Time) string {
	if value.IsZero() {
		return "unknown"
	}
	return value.Format(timestampLayout)
}
