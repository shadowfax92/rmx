package cmd

import (
	"context"
	"strings"
	"testing"

	"rmx/internal/rmux"
)

func TestRunSendTextJoinsTextArgsForTarget(t *testing.T) {
	client := &fakeSendClient{}

	err := runSendText(context.Background(), client, "codex/task", []string{"hello", "from", "wrapper"})
	if err != nil {
		t.Fatalf("runSendText returned error: %v", err)
	}

	if client.textTarget != "codex/task" {
		t.Fatalf("text target = %q, want codex/task", client.textTarget)
	}
	if client.text != "hello from wrapper" {
		t.Fatalf("text = %q, want joined text", client.text)
	}
}

func TestRunSendTextRejectsMissingText(t *testing.T) {
	client := &fakeSendClient{}

	err := runSendText(context.Background(), client, "codex/task", nil)
	if err == nil {
		t.Fatal("runSendText returned nil error for missing text")
	}
	if !strings.Contains(err.Error(), "text") {
		t.Fatalf("error = %q, want text message", err)
	}
}

func TestRunSendEnterTargetsSession(t *testing.T) {
	client := &fakeSendClient{}

	err := runSendEnter(context.Background(), client, "codex/task")
	if err != nil {
		t.Fatalf("runSendEnter returned error: %v", err)
	}

	if client.enterTarget != "codex/task" {
		t.Fatalf("enter target = %q, want codex/task", client.enterTarget)
	}
}

func TestSendCommandHasTextAndEnterSubcommands(t *testing.T) {
	for _, args := range [][]string{
		{"send", "text"},
		{"send", "enter"},
	} {
		cmd, _, err := rootCmd.Find(args)
		if err != nil {
			t.Fatalf("Find(%v) returned error: %v", args, err)
		}
		if cmd == nil {
			t.Fatalf("Find(%v) returned nil command", args)
		}
	}
}

func TestRootHelpIncludesInputGroup(t *testing.T) {
	help := groupedHelp(rootCmd)

	if !strings.Contains(help, "Input:") {
		t.Fatalf("help = %q, want input group", help)
	}
	if !strings.Contains(help, "send") {
		t.Fatalf("help = %q, want send command", help)
	}
}

type fakeSendClient struct {
	textTarget  string
	text        string
	enterTarget string
}

func (f *fakeSendClient) ListSessions(ctx context.Context) ([]rmux.Session, error) {
	return []rmux.Session{{Name: "picked/task"}}, nil
}

func (f *fakeSendClient) SendText(ctx context.Context, target string, text string) error {
	f.textTarget = target
	f.text = text
	return nil
}

func (f *fakeSendClient) SendEnter(ctx context.Context, target string) error {
	f.enterTarget = target
	return nil
}
