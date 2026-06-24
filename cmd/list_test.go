package cmd

import (
	"strings"
	"testing"
	"time"

	"rmx/internal/rmux"
)

func TestRenderSessionTableIncludesLastActiveTimestamp(t *testing.T) {
	sessions := []rmux.Session{
		{
			Name:         "codex/task",
			Windows:      1,
			CreatedAt:    time.Unix(1700000000, 0),
			LastActiveAt: time.Unix(1700000300, 0),
		},
	}
	now := time.Unix(1700000600, 0)

	table := renderSessionTable(sessions, now)

	if !strings.Contains(table, sessions[0].LastActiveAt.Format(timestampLayout)) {
		t.Fatalf("table %q missing last active timestamp", table)
	}
	if !strings.Contains(table, "5m ago") {
		t.Fatalf("table %q missing relative age", table)
	}
}

func TestRenderSessionTableShowsExitedState(t *testing.T) {
	sessions := []rmux.Session{
		{Name: "codex/live", Windows: 1, LastActiveAt: time.Unix(1700000300, 0)},
		{Name: "codex/gone", Windows: 1, Exited: true, LastActiveAt: time.Unix(1700000100, 0)},
	}
	now := time.Unix(1700000600, 0)

	table := renderSessionTable(sessions, now)

	if !strings.Contains(table, "exited") {
		t.Fatalf("table %q missing exited state", table)
	}
	if !strings.Contains(table, "detached") {
		t.Fatalf("table %q missing detached state for live session", table)
	}
}
