package cmd

import (
	"testing"
	"time"

	"rmx/internal/rmux"
	"rmx/internal/store"
)

func TestExitedToSessionMarksExitedAndUsesExitTime(t *testing.T) {
	rec := store.ExitedSession{
		Name:      "codex/task",
		Windows:   3,
		CreatedAt: time.Unix(1700000000, 0),
		ExitedAt:  time.Unix(1700000500, 0),
	}

	got := exitedToSession(rec)

	if !got.Exited {
		t.Fatal("exitedToSession Exited = false, want true")
	}
	if got.Attached {
		t.Fatal("exitedToSession Attached = true, want false")
	}
	if !got.LastActiveAt.Equal(rec.ExitedAt) {
		t.Fatalf("LastActiveAt = %v, want exit time %v", got.LastActiveAt, rec.ExitedAt)
	}
	if got.Name != "codex/task" || got.Windows != 3 || !got.CreatedAt.Equal(rec.CreatedAt) {
		t.Fatalf("metadata not carried: %#v", got)
	}
}

func TestMergeSessionsOverlaysExitedSortedByRecency(t *testing.T) {
	live := []rmux.Session{
		{Name: "claude/live", LastActiveAt: time.Unix(1700000400, 0)},
	}
	exited := []store.ExitedSession{
		{Name: "codex/old", ExitedAt: time.Unix(1700000100, 0)},
		{Name: "codex/recent", ExitedAt: time.Unix(1700000450, 0)},
	}

	got := mergeSessions(live, exited)

	names := []string{got[0].Name, got[1].Name, got[2].Name}
	want := []string{"codex/recent", "claude/live", "codex/old"}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("merged order = %v, want %v", names, want)
		}
	}
	if got[0].Exited != true || got[1].Exited != false || got[2].Exited != true {
		t.Fatalf("exited flags wrong: %v", []bool{got[0].Exited, got[1].Exited, got[2].Exited})
	}
}

func TestMergeSessionsLiveWinsOnNameCollision(t *testing.T) {
	live := []rmux.Session{
		{Name: "codex/task", LastActiveAt: time.Unix(1700000500, 0)},
	}
	exited := []store.ExitedSession{
		{Name: "codex/task", ExitedAt: time.Unix(1700000100, 0)},
		{Name: "codex/other", ExitedAt: time.Unix(1700000200, 0)},
	}

	got := mergeSessions(live, exited)

	if len(got) != 2 {
		t.Fatalf("merged length = %d, want 2 (collision deduped)", len(got))
	}
	for _, s := range got {
		if s.Name == "codex/task" && s.Exited {
			t.Fatal("codex/task should be the live session, not exited")
		}
	}
}
