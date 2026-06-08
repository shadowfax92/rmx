package rmux

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestParseSessionsReadsFormatFields(t *testing.T) {
	out := "codex/beta\t2\t1\t1700000000\t1700000300\nclaude/alpha\t1\t0\t1700000100\t1700000105"

	sessions, err := parseSessions(out)
	if err != nil {
		t.Fatalf("parseSessions returned error: %v", err)
	}

	want := []Session{
		{
			Name:         "codex/beta",
			Windows:      2,
			Attached:     true,
			CreatedAt:    time.Unix(1700000000, 0),
			LastActiveAt: time.Unix(1700000300, 0),
		},
		{
			Name:         "claude/alpha",
			Windows:      1,
			Attached:     false,
			CreatedAt:    time.Unix(1700000100, 0),
			LastActiveAt: time.Unix(1700000105, 0),
		},
	}
	if !reflect.DeepEqual(sessions, want) {
		t.Fatalf("sessions = %#v, want %#v", sessions, want)
	}
}

func TestParseSessionsRejectsMalformedRows(t *testing.T) {
	_, err := parseSessions("alpha\t1\t0")
	if err == nil {
		t.Fatal("parseSessions returned nil error for malformed row")
	}
}

func TestSortSessionsByActivityThenName(t *testing.T) {
	sessions := []Session{
		{Name: "older", LastActiveAt: time.Unix(10, 0)},
		{Name: "same-b", LastActiveAt: time.Unix(20, 0)},
		{Name: "same-a", LastActiveAt: time.Unix(20, 0)},
	}

	SortSessions(sessions)

	got := []string{sessions[0].Name, sessions[1].Name, sessions[2].Name}
	want := []string{"same-a", "same-b", "older"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sorted names = %v, want %v", got, want)
	}
}

func TestClientListSessionsRunsRmuxFormat(t *testing.T) {
	runner := &recordingRunner{
		output: "alpha\t1\t0\t1700000000\t1700000005",
	}
	client := Client{Binary: "rmux", Runner: runner}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}

	if len(sessions) != 1 || sessions[0].Name != "alpha" {
		t.Fatalf("sessions = %#v, want alpha", sessions)
	}
	wantArgs := []string{"list-sessions", "-F", sessionListFormat}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, wantArgs)
	}
}

func TestClientTreatsNoSessionsAsEmptyList(t *testing.T) {
	client := Client{Runner: &recordingRunner{err: errors.New("rmux list-sessions: no sessions")}}

	sessions, err := client.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(sessions) != 0 {
		t.Fatalf("sessions = %#v, want empty", sessions)
	}
}

func TestCapturePaneUsesNegativeStartForLineLimit(t *testing.T) {
	runner := &recordingRunner{output: "tail"}
	client := Client{Binary: "rmux", Runner: runner}

	out, err := client.CapturePane(context.Background(), "codex/task", 20)
	if err != nil {
		t.Fatalf("CapturePane returned error: %v", err)
	}
	if out != "tail" {
		t.Fatalf("output = %q, want tail", out)
	}
	wantArgs := []string{"capture-pane", "-p", "-t", "codex/task", "-S", "-20"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, wantArgs)
	}
}

func TestKillSessionTargetsExactName(t *testing.T) {
	runner := &recordingRunner{}
	client := Client{Binary: "rmux", Runner: runner}

	if err := client.KillSession(context.Background(), "codex/task"); err != nil {
		t.Fatalf("KillSession returned error: %v", err)
	}

	wantArgs := []string{"kill-session", "-t", "=codex/task"}
	if !reflect.DeepEqual(runner.calls[0].args, wantArgs) {
		t.Fatalf("args = %v, want %v", runner.calls[0].args, wantArgs)
	}
}

type recordingRunner struct {
	output string
	err    error
	calls  []recordedCall
}

type recordedCall struct {
	binary string
	args   []string
}

func (r *recordingRunner) Run(ctx context.Context, binary string, args ...string) (string, error) {
	r.calls = append(r.calls, recordedCall{binary: binary, args: append([]string(nil), args...)})
	return r.output, r.err
}

func (r *recordingRunner) RunInteractive(ctx context.Context, binary string, args ...string) error {
	r.calls = append(r.calls, recordedCall{binary: binary, args: append([]string(nil), args...)})
	return r.err
}
