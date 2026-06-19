package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"rmx/internal/rmux"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(exitCmd)
}

var exitCmd = &cobra.Command{
	Use:         "exit",
	Aliases:     []string{"e", "quit"},
	Annotations: map[string]string{"group": "Sessions:"},
	Short:       "Exit the current rmux session",
	Args:        cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := exitCurrentSession(cmd.Context(), rmux.DefaultClient(), os.LookupEnv)
		return err
	},
}

// exitCurrentSession resolves and kills the rmux session attached to this pane.
func exitCurrentSession(ctx context.Context, client rmux.Client, lookupEnv func(string) (string, bool)) (string, error) {
	value, ok := lookupEnv("RMUX")
	if !ok || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("rmx exit must be run from inside a rmux pane")
	}

	name, err := client.CurrentSession(ctx)
	if err != nil {
		return "", err
	}
	if name == "" {
		return "", fmt.Errorf("could not determine current rmux session")
	}
	return name, client.KillSession(ctx, name)
}
