package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"

	"github.com/spf13/cobra"
)

// exitCaptureLines is how much pane scrollback is saved when a session exits, so
// `rmx cat` can show what happened after the session is gone.
const exitCaptureLines = 1000

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
		_, err := exitCurrentSession(cmd.Context(), rmux.DefaultClient(), store.Default(), os.LookupEnv, time.Now(), cmd.ErrOrStderr())
		return err
	},
}

// exitCurrentSession records the current session (so it shows as exited in
// `rmx ls` and stays cat-able) and then kills it. Recording is best-effort —
// only the kill is essential, so a capture or store failure just warns.
func exitCurrentSession(ctx context.Context, client rmux.Client, st store.Store, lookupEnv func(string) (string, bool), now time.Time, warn io.Writer) (string, error) {
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

	recordExit(ctx, client, st, name, now, warn)
	return name, client.KillSession(ctx, name)
}

// recordExit captures the pane and writes the exited-session record before the
// kill. Each step warns on failure rather than aborting, so exit never fails to
// kill just because the sidecar could not be written.
func recordExit(ctx context.Context, client rmux.Client, st store.Store, name string, now time.Time, warn io.Writer) {
	output, err := client.CapturePane(ctx, name, exitCaptureLines)
	if err != nil {
		fmt.Fprintf(warn, "rmx exit: capture %s: %v\n", name, err)
		output = ""
	}

	rec := store.ExitedSession{Name: name, ExitedAt: now}
	if meta, ok := sessionMeta(ctx, client, name); ok {
		rec.Windows = meta.Windows
		rec.CreatedAt = meta.CreatedAt
	}

	if err := st.Record(rec, output); err != nil {
		fmt.Fprintf(warn, "rmx exit: record %s: %v\n", name, err)
	}
}

// sessionMeta looks up the live session's window count and created time so the
// exited row in `rmx ls` keeps that metadata after the session is gone.
func sessionMeta(ctx context.Context, client rmux.Client, name string) (rmux.Session, bool) {
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		return rmux.Session{}, false
	}
	for _, session := range sessions {
		if session.Name == name {
			return session, true
		}
	}
	return rmux.Session{}, false
}
