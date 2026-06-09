package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"rmx/internal/rmux"
)

func init() {
	rootCmd.AddCommand(rmCmd)
}

var rmCmd = &cobra.Command{
	Use:         "rm [session...]",
	Aliases:     []string{"remove", "kill"},
	Annotations: map[string]string{"group": "Sessions:"},
	Short:       "Remove rmux sessions",
	Args:        cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client := rmux.DefaultClient()
		targets := args
		if len(targets) == 0 {
			sessions, err := client.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			targets, err = pickSessions(cmd.Context(), sessions, "rm > ", true)
			if err != nil {
				return err
			}
		}
		for _, target := range targets {
			if err := client.KillSession(cmd.Context(), target); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %s\n", target)
		}
		return nil
	},
}
