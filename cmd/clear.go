package cmd

import (
	"fmt"
	"io"

	"rmx/internal/store"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clearCmd)
}

var clearCmd = &cobra.Command{
	Use:         "clear",
	Aliases:     []string{"clr"},
	Annotations: map[string]string{"group": "Sessions:"},
	Short:       "Clear recorded exited sessions",
	Args:        cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runClear(cmd.OutOrStdout(), store.Default())
	},
}

// runClear drops every recorded exited session (the inactive ones lingering in
// `rmx ls`) and reports how many were cleared.
func runClear(out io.Writer, st store.Store) error {
	n, err := st.Clear()
	if err != nil {
		return err
	}
	if n == 0 {
		fmt.Fprintln(out, "No exited sessions to clear.")
		return nil
	}
	fmt.Fprintf(out, "Cleared %d exited session(s).\n", n)
	return nil
}
