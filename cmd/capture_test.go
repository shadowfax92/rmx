package cmd

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"rmx/internal/rmux"
)

func TestCatCommandIsPrimaryWithCaptureAliases(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"cat"})
	if err != nil {
		t.Fatalf("Find(cat) returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Find(cat) returned nil command")
	}
	if cmd.Name() != "cat" {
		t.Fatalf("command name = %q, want cat", cmd.Name())
	}
	for _, want := range []string{"capture", "cap"} {
		if !contains(cmd.Aliases, want) {
			t.Fatalf("aliases = %v, want %q", cmd.Aliases, want)
		}
	}
}

func TestRenderCapturesSeparatesEachSession(t *testing.T) {
	outputs := map[string]string{
		"codex/task":  "codex output\n",
		"claude/task": "claude output\n",
	}
	var buf bytes.Buffer

	err := renderCaptures(context.Background(), &buf, []rmux.Session{
		{Name: "codex/task"},
		{Name: "claude/task"},
	}, func(ctx context.Context, name string) (string, error) {
		return outputs[name], nil
	})
	if err != nil {
		t.Fatalf("renderCaptures returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{"codex/task", "codex output", "claude/task", "claude output"} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered output %q missing %q", got, want)
		}
	}
	if strings.Count(got, "rmx cat") != 2 {
		t.Fatalf("rendered output = %q, want two capture headers", got)
	}
}

func TestFishShortcutsForwardCaptureVerbsToCat(t *testing.T) {
	content, err := os.ReadFile("../rmx.fish")
	if err != nil {
		t.Fatalf("ReadFile(rmx.fish) returned error: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "case c cat cap capture") {
		t.Fatalf("fish helper missing capture verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx cat $rest") {
		t.Fatalf("fish helper should forward capture verbs to rmx cat: %s", text)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
