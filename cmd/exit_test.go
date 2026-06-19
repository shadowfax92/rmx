package cmd

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"rmx/internal/rmux"
)

func TestExitCurrentSessionKillsCurrentRmuxSession(t *testing.T) {
	runner := &exitRecordingRunner{output: "codex/task\n"}
	client := rmux.Client{Binary: "rmux", Runner: runner}

	name, err := exitCurrentSession(context.Background(), client, func(key string) (string, bool) {
		if key != "RMUX" {
			t.Fatalf("lookup key = %q, want RMUX", key)
		}
		return "/tmp/rmux/default,1,2", true
	})
	if err != nil {
		t.Fatalf("exitCurrentSession returned error: %v", err)
	}
	if name != "codex/task" {
		t.Fatalf("session name = %q, want codex/task", name)
	}

	want := [][]string{
		{"display-message", "-p", "#{session_name}"},
		{"kill-session", "-t", "=codex/task"},
	}
	if !reflect.DeepEqual(runner.args(), want) {
		t.Fatalf("rmux calls = %v, want %v", runner.args(), want)
	}
}

func TestExitCurrentSessionRequiresRmuxPane(t *testing.T) {
	runner := &exitRecordingRunner{output: "codex/task"}
	client := rmux.Client{Binary: "rmux", Runner: runner}

	_, err := exitCurrentSession(context.Background(), client, func(string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatal("exitCurrentSession returned nil error outside rmux")
	}
	if !strings.Contains(err.Error(), "inside a rmux pane") {
		t.Fatalf("error = %q, want inside-rmux guidance", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("rmux calls = %v, want none", runner.calls)
	}
}

func TestExitCommandIsRegisteredWithAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"exit"})
	if err != nil {
		t.Fatalf("Find(exit) returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Find(exit) returned nil command")
	}
	if cmd.Name() != "exit" {
		t.Fatalf("command name = %q, want exit", cmd.Name())
	}
	if cmd.Annotations["group"] != "Sessions:" {
		t.Fatalf("group = %q, want Sessions:", cmd.Annotations["group"])
	}
	if !contains(cmd.Aliases, "e") {
		t.Fatalf("aliases = %v, want e", cmd.Aliases)
	}

	aliasCmd, _, err := rootCmd.Find([]string{"e"})
	if err != nil {
		t.Fatalf("Find(e) returned error: %v", err)
	}
	if aliasCmd != cmd {
		t.Fatalf("alias command = %v, want exit command", aliasCmd)
	}
}

func TestFishShortcutsForwardExitVerbsToExit(t *testing.T) {
	content, err := os.ReadFile("../rmx.fish")
	if err != nil {
		t.Fatalf("ReadFile(rmx.fish) returned error: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "case e exit quit") {
		t.Fatalf("fish helper missing exit verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx exit $rest") {
		t.Fatalf("fish helper should forward exit verbs to rmx exit: %s", text)
	}
}

func TestReadmeDocumentsExitCommand(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) returned error: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"rmx exit",
		"Exit current session",
		"rmx e",
		"### Exit",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q", want)
		}
	}
}

type exitRecordingRunner struct {
	output string
	err    error
	calls  []exitRecordedCall
}

type exitRecordedCall struct {
	binary string
	args   []string
}

func (r *exitRecordingRunner) Run(ctx context.Context, binary string, args ...string) (string, error) {
	r.calls = append(r.calls, exitRecordedCall{binary: binary, args: append([]string(nil), args...)})
	return r.output, r.err
}

func (r *exitRecordingRunner) RunInteractive(ctx context.Context, binary string, args ...string) error {
	r.calls = append(r.calls, exitRecordedCall{binary: binary, args: append([]string(nil), args...)})
	return r.err
}

func (r *exitRecordingRunner) args() [][]string {
	args := make([][]string, 0, len(r.calls))
	for _, call := range r.calls {
		args = append(args, call.args)
	}
	return args
}
