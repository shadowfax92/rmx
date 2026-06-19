package cmd

import (
	"context"
	"os"
	"path/filepath"
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
	if client.listCalls != 0 {
		t.Fatalf("ListSessions called %d times, want 0 for explicit target", client.listCalls)
	}
}

func TestRunSendTextPicksTargetWhenTargetMissing(t *testing.T) {
	installFakeFzf(t, "picked/task")
	client := &fakeSendClient{
		sessions: []rmux.Session{{Name: "picked/task", Windows: 1}},
	}

	err := runSendText(context.Background(), client, "", []string{"hello", "from", "picker"})
	if err != nil {
		t.Fatalf("runSendText returned error: %v", err)
	}

	if client.listCalls != 1 {
		t.Fatalf("ListSessions called %d times, want 1", client.listCalls)
	}
	if client.textTarget != "picked/task" {
		t.Fatalf("text target = %q, want picked/task", client.textTarget)
	}
	if client.text != "hello from picker" {
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

func TestRunSendEnterPicksTargetWhenTargetMissing(t *testing.T) {
	installFakeFzf(t, "picked/task")
	client := &fakeSendClient{
		sessions: []rmux.Session{{Name: "picked/task", Windows: 1}},
	}

	err := runSendEnter(context.Background(), client, "")
	if err != nil {
		t.Fatalf("runSendEnter returned error: %v", err)
	}

	if client.listCalls != 1 {
		t.Fatalf("ListSessions called %d times, want 1", client.listCalls)
	}
	if client.enterTarget != "picked/task" {
		t.Fatalf("enter target = %q, want picked/task", client.enterTarget)
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

func TestFishShortcutsForwardSendVerbs(t *testing.T) {
	content, err := os.ReadFile("../rmx.fish")
	if err != nil {
		t.Fatalf("ReadFile(rmx.fish) returned error: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "case s send") {
		t.Fatalf("fish helper missing send verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx send $rest") {
		t.Fatalf("fish helper should forward send verbs to rmx send: %s", text)
	}
	if !strings.Contains(text, "case text enter") {
		t.Fatalf("fish helper missing text/enter verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx send $argv") {
		t.Fatalf("fish helper should forward text/enter verbs to rmx send with verb: %s", text)
	}
}

func TestReadmeDocumentsSendInput(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) returned error: %v", err)
	}
	text := string(content)

	for _, want := range []string{"rmx send text -t codex/feat-example", "rmx send enter -t codex/feat-example"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q", want)
		}
	}
}

type fakeSendClient struct {
	textTarget  string
	text        string
	enterTarget string
	listCalls   int
	sessions    []rmux.Session
}

func (f *fakeSendClient) ListSessions(ctx context.Context) ([]rmux.Session, error) {
	f.listCalls++
	if f.sessions != nil {
		return f.sessions, nil
	}
	return []rmux.Session{{Name: "picked/task", Windows: 1}}, nil
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

func installFakeFzf(t *testing.T, selected string) {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "fzf")
	script := "#!/bin/sh\n" +
		"input=$(cat)\n" +
		"case \"$input\" in\n" +
		"  *\"" + selected + "\"*) printf '" + selected + "\t" + selected + "\t1 windows\t1m ago\tdetached\n' ;;\n" +
		"  *) echo 'missing selected session in picker input' >&2; exit 2 ;;\n" +
		"esac\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile(fake fzf) returned error: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}
