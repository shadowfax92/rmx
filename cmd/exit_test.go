package cmd

import (
	"context"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"
)

func TestExitCurrentSessionRecordsAndKills(t *testing.T) {
	runner := &exitRecordingRunner{
		outputs: map[string]string{
			"display-message": "codex/task\n",
			"capture-pane":    "line one\nline two\n",
			"list-sessions":   "codex/task\t2\t1\t1700000000\t1700000300",
		},
	}
	client := rmux.Client{Binary: "rmux", Runner: runner}
	st := store.Store{Dir: t.TempDir()}
	now := time.Unix(1700009999, 0)

	name, err := exitCurrentSession(context.Background(), client, st, func(key string) (string, bool) {
		if key != "RMUX" {
			t.Fatalf("lookup key = %q, want RMUX", key)
		}
		return "/tmp/rmux/default,1,2", true
	}, now, io.Discard)
	if err != nil {
		t.Fatalf("exitCurrentSession returned error: %v", err)
	}
	if name != "codex/task" {
		t.Fatalf("session name = %q, want codex/task", name)
	}

	want := [][]string{
		{"display-message", "-p", "#{session_name}"},
		{"capture-pane", "-p", "-t", "codex/task", "-S", "-1000", "-E", "-1"},
		{"list-sessions", "-F", "#{session_name}\t#{session_windows}\t#{session_attached}\t#{session_created}\t#{session_activity}"},
		{"kill-session", "-t", "=codex/task"},
	}
	if !reflect.DeepEqual(runner.args(), want) {
		t.Fatalf("rmux calls = %v, want %v", runner.args(), want)
	}

	recs, err := st.List()
	if err != nil {
		t.Fatalf("store List returned error: %v", err)
	}
	if len(recs) != 1 || recs[0].Name != "codex/task" {
		t.Fatalf("store records = %#v, want one codex/task", recs)
	}
	if !recs[0].ExitedAt.Equal(now) {
		t.Fatalf("ExitedAt = %v, want now %v", recs[0].ExitedAt, now)
	}
	if recs[0].Windows != 2 || !recs[0].CreatedAt.Equal(time.Unix(1700000000, 0)) {
		t.Fatalf("recorded metadata = %#v, want windows 2 / created 1700000000", recs[0])
	}
	out, ok, err := st.Output("codex/task")
	if err != nil || !ok {
		t.Fatalf("store Output ok=%v err=%v, want recorded output", ok, err)
	}
	if out != "line one\nline two\n" {
		t.Fatalf("stored output = %q, want captured pane text", out)
	}
}

func TestExitRecordsEvenWhenCaptureFails(t *testing.T) {
	runner := &exitRecordingRunner{
		outputs: map[string]string{
			"display-message": "codex/task",
			"list-sessions":   "codex/task\t1\t1\t1700000000\t1700000300",
		},
		errs: map[string]error{"capture-pane": errors.New("pane gone")},
	}
	client := rmux.Client{Binary: "rmux", Runner: runner}
	st := store.Store{Dir: t.TempDir()}

	name, err := exitCurrentSession(context.Background(), client, st, func(string) (string, bool) {
		return "/tmp/rmux/default,1,2", true
	}, time.Unix(1700009999, 0), io.Discard)
	if err != nil {
		t.Fatalf("exitCurrentSession returned error: %v", err)
	}
	if name != "codex/task" {
		t.Fatalf("name = %q, want codex/task", name)
	}

	// Kill still happened despite the capture failure.
	killed := false
	for _, call := range runner.args() {
		if len(call) > 0 && call[0] == "kill-session" {
			killed = true
		}
	}
	if !killed {
		t.Fatal("kill-session not called after capture failure")
	}

	// Record still written, with empty output.
	out, ok, err := st.Output("codex/task")
	if err != nil {
		t.Fatalf("store Output returned error: %v", err)
	}
	if !ok || out != "" {
		t.Fatalf("stored output ok=%v text=%q, want recorded empty output", ok, out)
	}
}

func TestExitCurrentSessionRequiresRmuxPane(t *testing.T) {
	runner := &exitRecordingRunner{}
	client := rmux.Client{Binary: "rmux", Runner: runner}

	_, err := exitCurrentSession(context.Background(), client, store.Store{Dir: t.TempDir()}, func(string) (string, bool) {
		return "", false
	}, time.Unix(1, 0), io.Discard)
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

// exitRecordingRunner returns per-command output/errors keyed by the rmux
// subcommand (args[0]) and records every call in order.
type exitRecordingRunner struct {
	outputs map[string]string
	errs    map[string]error
	calls   []exitRecordedCall
}

type exitRecordedCall struct {
	binary string
	args   []string
}

func (r *exitRecordingRunner) result(args []string) (string, error) {
	cmd := ""
	if len(args) > 0 {
		cmd = args[0]
	}
	return r.outputs[cmd], r.errs[cmd]
}

func (r *exitRecordingRunner) Run(ctx context.Context, binary string, args ...string) (string, error) {
	r.calls = append(r.calls, exitRecordedCall{binary: binary, args: append([]string(nil), args...)})
	return r.result(args)
}

func (r *exitRecordingRunner) RunInteractive(ctx context.Context, binary string, args ...string) error {
	r.calls = append(r.calls, exitRecordedCall{binary: binary, args: append([]string(nil), args...)})
	_, err := r.result(args)
	return err
}

func (r *exitRecordingRunner) args() [][]string {
	args := make([][]string, 0, len(r.calls))
	for _, call := range r.calls {
		args = append(args, call.args)
	}
	return args
}
