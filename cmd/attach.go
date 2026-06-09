package cmd

import (
	"github.com/spf13/cobra"
	"rmx/internal/rmux"
)

func init() {
	rootCmd.AddCommand(attachCmd)
}

var attachCmd = &cobra.Command{
	Use:         "attach [session]",
	Aliases:     []string{"a"},
	Annotations: map[string]string{"group": "Sessions:"},
	Short:       "Attach to an rmux session",
	Args:        cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := rmux.DefaultClient()
		target := ""
		if len(args) == 1 {
			target = args[0]
		} else {
			sessions, err := client.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			targets, err := pickSessions(cmd.Context(), sessions, "attach > ", false)
			if err != nil {
				return err
			}
			target = targets[0]
		}
		return client.AttachSession(cmd.Context(), target)
	},
}
