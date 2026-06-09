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
