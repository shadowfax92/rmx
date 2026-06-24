package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"
)

func TestCapturePlanDispatchesStoreForExitedAndLiveForLive(t *testing.T) {
	runner := &exitRecordingRunner{
		outputs: map[string]string{
			"list-sessions": "codex/live\t1\t0\t1700000000\t1700000300",
			"capture-pane":  "live pane text",
		},
	}
	client := rmux.Client{Binary: "rmux", Runner: runner}
	st := store.Store{Dir: t.TempDir()}
	if err := st.Record(store.ExitedSession{Name: "codex/gone", ExitedAt: time.Unix(1700000100, 0)}, "ghost output"); err != nil {
		t.Fatalf("seed store: %v", err)
	}

	sessions, capture, err := capturePlan(context.Background(), client, st, time.Unix(1700001000, 0), []string{"codex/gone", "codex/live"}, io.Discard)
	if err != nil {
		t.Fatalf("capturePlan returned error: %v", err)
	}
	if len(sessions) != 2 || sessions[0].Name != "codex/gone" || !sessions[0].Exited {
		t.Fatalf("selected sessions = %#v, want exited codex/gone first", sessions)
	}
	if sessions[1].Name != "codex/live" || sessions[1].Exited {
		t.Fatalf("second session = %#v, want live codex/live", sessions[1])
	}

	gone, err := capture(context.Background(), "codex/gone")
	if err != nil {
		t.Fatalf("capture(codex/gone) error: %v", err)
	}
	if gone != "ghost output" {
		t.Fatalf("exited capture = %q, want stored output (not a live capture)", gone)
	}
	got, err := capture(context.Background(), "codex/live")
	if err != nil {
		t.Fatalf("capture(codex/live) error: %v", err)
	}
	if got != "live pane text" {
		t.Fatalf("live capture = %q, want rmux pane text", got)
	}

	for _, call := range runner.args() {
		if len(call) > 0 && call[0] == "capture-pane" && contains(call, "codex/gone") {
			t.Fatalf("exited session was captured via rmux: %v", call)
		}
	}
}

func TestRenderCapturesNotesExitedSessions(t *testing.T) {
	var buf bytes.Buffer
	err := renderCaptures(context.Background(), &buf, []rmux.Session{
		{Name: "codex/gone", Exited: true, LastActiveAt: time.Now().Add(-2 * time.Hour)},
	}, func(ctx context.Context, name string) (string, error) {
		return "ghost output", nil
	})
	if err != nil {
		t.Fatalf("renderCaptures returned error: %v", err)
	}
	got := buf.String()
	if !strings.Contains(got, "exited") {
		t.Fatalf("exited cat header %q missing exited note", got)
	}
	if !strings.Contains(got, "ghost output") {
		t.Fatalf("exited cat output %q missing replayed text", got)
	}
}

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
