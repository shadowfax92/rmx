package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newStore(t *testing.T) Store {
	t.Helper()
	return Store{Dir: t.TempDir()}
}

func TestRecordAndListRoundTrip(t *testing.T) {
	s := newStore(t)
	older := ExitedSession{Name: "codex/old", Windows: 1, CreatedAt: time.Unix(1700000000, 0), ExitedAt: time.Unix(1700000100, 0)}
	newer := ExitedSession{Name: "claude/new", Windows: 2, CreatedAt: time.Unix(1700000200, 0), ExitedAt: time.Unix(1700000300, 0)}

	if err := s.Record(older, "old output"); err != nil {
		t.Fatalf("Record(older) returned error: %v", err)
	}
	if err := s.Record(newer, "new output"); err != nil {
		t.Fatalf("Record(newer) returned error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("List returned %d records, want 2", len(got))
	}
	if got[0].Name != "claude/new" {
		t.Fatalf("List not sorted newest-exit first: got %q first", got[0].Name)
	}
	if got[0].Windows != 2 || !got[0].CreatedAt.Equal(newer.CreatedAt) || !got[0].ExitedAt.Equal(newer.ExitedAt) {
		t.Fatalf("record fields not preserved: %#v", got[0])
	}
}

func TestOutputReturnsCapturedText(t *testing.T) {
	s := newStore(t)
	if err := s.Record(ExitedSession{Name: "codex/task", ExitedAt: time.Unix(1700000000, 0)}, "captured lines\n"); err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	text, ok, err := s.Output("codex/task")
	if err != nil {
		t.Fatalf("Output returned error: %v", err)
	}
	if !ok {
		t.Fatal("Output ok = false, want true for recorded session")
	}
	if text != "captured lines\n" {
		t.Fatalf("Output text = %q, want captured lines", text)
	}

	if _, ok, _ := s.Output("does/not-exist"); ok {
		t.Fatal("Output ok = true for missing session, want false")
	}
}

func TestPruneRemovesExpiredKeepsFresh(t *testing.T) {
	s := newStore(t)
	now := time.Unix(1700100000, 0)
	stale := ExitedSession{Name: "codex/stale", ExitedAt: now.Add(-48 * time.Hour)}
	fresh := ExitedSession{Name: "codex/fresh", ExitedAt: now.Add(-1 * time.Hour)}
	if err := s.Record(stale, "stale"); err != nil {
		t.Fatalf("Record(stale): %v", err)
	}
	if err := s.Record(fresh, "fresh"); err != nil {
		t.Fatalf("Record(fresh): %v", err)
	}

	if err := s.Prune(now.Add(-Retention)); err != nil {
		t.Fatalf("Prune returned error: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "codex/fresh" {
		t.Fatalf("after prune = %#v, want only codex/fresh", got)
	}
	if _, ok, _ := s.Output("codex/stale"); ok {
		t.Fatal("pruned session still has output on disk")
	}
}

func TestClearRemovesAll(t *testing.T) {
	s := newStore(t)
	for _, name := range []string{"a/one", "b/two", "c/three"} {
		if err := s.Record(ExitedSession{Name: name, ExitedAt: time.Unix(1700000000, 0)}, "x"); err != nil {
			t.Fatalf("Record(%s): %v", name, err)
		}
	}

	n, err := s.Clear()
	if err != nil {
		t.Fatalf("Clear returned error: %v", err)
	}
	if n != 3 {
		t.Fatalf("Clear count = %d, want 3", n)
	}
	got, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("after clear List = %#v, want empty", got)
	}
}

func TestListMissingDirIsEmpty(t *testing.T) {
	s := Store{Dir: filepath.Join(t.TempDir(), "does-not-exist")}
	got, err := s.List()
	if err != nil {
		t.Fatalf("List on missing dir returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("List on missing dir = %#v, want empty", got)
	}
}

func TestListSkipsMalformedRecords(t *testing.T) {
	s := newStore(t)
	if err := s.Record(ExitedSession{Name: "codex/good", ExitedAt: time.Unix(1700000000, 0)}, "ok"); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(s.Dir, "exited"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir, "exited", "broken.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write malformed: %v", err)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(got) != 1 || got[0].Name != "codex/good" {
		t.Fatalf("List = %#v, want only codex/good (malformed skipped)", got)
	}
}

func TestEmptyDirIsNoOp(t *testing.T) {
	var s Store // Dir == ""
	if err := s.Record(ExitedSession{Name: "x", ExitedAt: time.Unix(1, 0)}, "y"); err == nil {
		t.Fatal("Record on empty-Dir store returned nil error, want a soft error")
	}
	got, err := s.List()
	if err != nil {
		t.Fatalf("List on empty-Dir store returned error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("List on empty-Dir store = %#v, want empty", got)
	}
}

func TestDefaultHonorsXDGStateHome(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	if got := Default().Dir; got != filepath.Join("/tmp/xdg-state", "rmx") {
		t.Fatalf("Default().Dir = %q, want /tmp/xdg-state/rmx", got)
	}
}
