package cmd

import (
	"context"
	"fmt"
	"strings"

	"rmx/internal/rmux"

	"github.com/spf13/cobra"
)

var sendTarget string

type sendClient interface {
	ListSessions(context.Context) ([]rmux.Session, error)
	SendText(context.Context, string, string) error
	SendEnter(context.Context, string) error
}

func init() {
	sendCmd.PersistentFlags().StringVarP(&sendTarget, "target", "t", "", "Target rmux session")
	sendCmd.AddCommand(sendTextCmd)
	sendCmd.AddCommand(sendEnterCmd)
	rootCmd.AddCommand(sendCmd)
}

var sendCmd = &cobra.Command{
	Use:         "send",
	Aliases:     []string{"s"},
	Annotations: map[string]string{"group": "Input:"},
	Short:       "Send input to an rmux session",
}

var sendTextCmd = &cobra.Command{
	Use:   "text [text...]",
	Short: "Send literal text to an rmux session",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSendText(cmd.Context(), rmux.DefaultClient(), sendTarget, args)
	},
}

var sendEnterCmd = &cobra.Command{
	Use:   "enter",
	Short: "Press Enter in an rmux session",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSendEnter(cmd.Context(), rmux.DefaultClient(), sendTarget)
	},
}

// runSendText resolves a target and writes the provided args as literal pane input.
func runSendText(ctx context.Context, client sendClient, target string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("text is required")
	}
	target, err := resolveSendTarget(ctx, client, target, "send text > ")
	if err != nil {
		return err
	}
	return client.SendText(ctx, target, strings.Join(args, " "))
}

// runSendEnter resolves a target and sends the Enter key.
func runSendEnter(ctx context.Context, client sendClient, target string) error {
	target, err := resolveSendTarget(ctx, client, target, "send enter > ")
	if err != nil {
		return err
	}
	return client.SendEnter(ctx, target)
}

// resolveSendTarget uses the explicit target flag or falls back to the session picker.
func resolveSendTarget(ctx context.Context, client sendClient, target string, prompt string) (string, error) {
	if target != "" {
		return target, nil
	}
	sessions, err := client.ListSessions(ctx)
	if err != nil {
		return "", err
	}
	targets, err := pickSessions(ctx, sessions, prompt, false)
	if err != nil {
		return "", err
	}
	return targets[0], nil
}
