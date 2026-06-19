package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"rmx/internal/rmux"
)

func TestTailCommandIsRegisteredInOutputGroup(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"tail"})
	if err != nil {
		t.Fatalf("Find(tail) returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("Find(tail) returned nil command")
	}
	if cmd.Name() != "tail" {
		t.Fatalf("command name = %q, want tail", cmd.Name())
	}
	if cmd.Annotations["group"] != "Output:" {
		t.Fatalf("group = %q, want Output:", cmd.Annotations["group"])
	}
	if !contains(cmd.Aliases, "follow") {
		t.Fatalf("aliases = %v, want follow", cmd.Aliases)
	}
}

func TestTailStateUsesFirstCaptureAsBaseline(t *testing.T) {
	sessions := []rmux.Session{{Name: "codex/task"}}
	captures := map[string][]string{
		"codex/task": {"old output", "old output\nnew output"},
	}
	capture := sequenceCapture(captures)
	var buf bytes.Buffer

	state, err := newTailState(context.Background(), sessions, capture)
	if err != nil {
		t.Fatalf("newTailState returned error: %v", err)
	}
	if buf.String() != "" {
		t.Fatalf("baseline wrote %q, want no output", buf.String())
	}

	err = state.poll(context.Background(), &buf, capture)
	if err != nil {
		t.Fatalf("poll returned error: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "old output") {
		t.Fatalf("tail output = %q, should not replay baseline", got)
	}
	if !strings.Contains(got, "[codex/task] new output") {
		t.Fatalf("tail output = %q, want prefixed appended output", got)
	}
}

func TestTailStatePrefixesMultilineOutputPerSession(t *testing.T) {
	sessions := []rmux.Session{{Name: "codex/task"}, {Name: "claude/task"}}
	captures := map[string][]string{
		"codex/task":  {"codex old", "codex old\ncodex one\ncodex two"},
		"claude/task": {"claude old", "claude old\nclaude one"},
	}
	capture := sequenceCapture(captures)
	var buf bytes.Buffer

	state, err := newTailState(context.Background(), sessions, capture)
	if err != nil {
		t.Fatalf("newTailState returned error: %v", err)
	}
	err = state.poll(context.Background(), &buf, capture)
	if err != nil {
		t.Fatalf("poll returned error: %v", err)
	}

	got := buf.String()
	for _, want := range []string{
		"[codex/task] codex one",
		"[codex/task] codex two",
		"[claude/task] claude one",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("tail output = %q, want %q", got, want)
		}
	}
}

func TestTailStateReturnsWriteErrors(t *testing.T) {
	sessions := []rmux.Session{{Name: "codex/task"}}
	captures := map[string][]string{
		"codex/task": {"old", "old\nnew"},
	}
	capture := sequenceCapture(captures)

	state, err := newTailState(context.Background(), sessions, capture)
	if err != nil {
		t.Fatalf("newTailState returned error: %v", err)
	}
	err = state.poll(context.Background(), failingWriter{}, capture)
	if !errors.Is(err, errFailingWrite) {
		t.Fatalf("poll error = %v, want failing write", err)
	}
}

func TestRenderTailChunkPreservesIntentionalBlankLines(t *testing.T) {
	var buf bytes.Buffer

	err := renderTailChunk(&buf, "codex/task", 0, "line\n\n")
	if err != nil {
		t.Fatalf("renderTailChunk returned error: %v", err)
	}

	got := buf.String()
	if strings.Count(got, "[codex/task]") != 2 {
		t.Fatalf("tail output = %q, want line plus one blank prefixed line", got)
	}
	if !strings.Contains(got, "[codex/task] \n") {
		t.Fatalf("tail output = %q, want prefixed blank line", got)
	}
}

func TestAppendedCaptureUsesOverlapWhenScrollbackRolls(t *testing.T) {
	got := appendedCapture("one\ntwo\nthree", "two\nthree\nfour")

	if got != "\nfour" {
		t.Fatalf("appendedCapture = %q, want rolled-over suffix", got)
	}
}

func TestTailPrefixColorsAreDeterministicAndDistinct(t *testing.T) {
	if tailPrefixColor(0) != tailPrefixColor(len(tailPrefixColors)) {
		t.Fatalf("color cycle did not wrap deterministically")
	}
	if tailPrefixColor(0) == tailPrefixColor(1) {
		t.Fatalf("first two tail prefix colors should differ")
	}
}

func TestFishShortcutsForwardTailVerbs(t *testing.T) {
	content, err := os.ReadFile("../rmx.fish")
	if err != nil {
		t.Fatalf("ReadFile(rmx.fish) returned error: %v", err)
	}
	text := string(content)

	if !strings.Contains(text, "case t tail follow") {
		t.Fatalf("fish helper missing tail verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx tail $rest") {
		t.Fatalf("fish helper should forward tail verbs to rmx tail: %s", text)
	}
}

func TestReadmeDocumentsTail(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) returned error: %v", err)
	}
	text := string(content)

	for _, want := range []string{"rmx tail", "polls every 5 seconds"} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q", want)
		}
	}
}

var errFailingWrite = errors.New("failing write")

type failingWriter struct{}

func (failingWriter) Write(p []byte) (int, error) {
	return 0, errFailingWrite
}

func sequenceCapture(captures map[string][]string) func(context.Context, string) (string, error) {
	indexes := map[string]int{}
	return func(ctx context.Context, name string) (string, error) {
		values := captures[name]
		idx := indexes[name]
		if idx >= len(values) {
			return values[len(values)-1], nil
		}
		indexes[name] = idx + 1
		return values[idx], nil
	}
}
