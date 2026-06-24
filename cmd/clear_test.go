package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"rmx/internal/store"
)

func TestRunClearRemovesRecordsAndReportsCount(t *testing.T) {
	st := store.Store{Dir: t.TempDir()}
	for _, name := range []string{"codex/one", "claude/two"} {
		if err := st.Record(store.ExitedSession{Name: name, ExitedAt: time.Unix(1, 0)}, "x"); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}

	var buf bytes.Buffer
	if err := runClear(&buf, st); err != nil {
		t.Fatalf("runClear returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "2") {
		t.Fatalf("output %q missing cleared count", buf.String())
	}

	recs, err := st.List()
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(recs) != 0 {
		t.Fatalf("records after clear = %#v, want empty", recs)
	}
}

func TestRunClearEmptyStoreReportsNothing(t *testing.T) {
	st := store.Store{Dir: t.TempDir()}

	var buf bytes.Buffer
	if err := runClear(&buf, st); err != nil {
		t.Fatalf("runClear returned error: %v", err)
	}
	if !strings.Contains(buf.String(), "No exited sessions") {
		t.Fatalf("output %q missing empty-store message", buf.String())
	}
}

func TestClearCommandIsRegisteredWithAlias(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"clear"})
	if err != nil {
		t.Fatalf("Find(clear) returned error: %v", err)
	}
	if cmd == nil || cmd.Name() != "clear" {
		t.Fatalf("Find(clear) = %v, want clear command", cmd)
	}
	if cmd.Annotations["group"] != "Sessions:" {
		t.Fatalf("group = %q, want Sessions:", cmd.Annotations["group"])
	}
	if !contains(cmd.Aliases, "clr") {
		t.Fatalf("aliases = %v, want clr", cmd.Aliases)
	}
}

func TestFishShortcutsForwardClearVerb(t *testing.T) {
	content, err := os.ReadFile("../rmx.fish")
	if err != nil {
		t.Fatalf("ReadFile(rmx.fish) returned error: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, "case clr clear") {
		t.Fatalf("fish helper missing clear verb cases: %s", text)
	}
	if !strings.Contains(text, "command rmx clear $rest") {
		t.Fatalf("fish helper should forward clear verbs to rmx clear: %s", text)
	}
}

func TestReadmeDocumentsClearCommand(t *testing.T) {
	content, err := os.ReadFile("../README.md")
	if err != nil {
		t.Fatalf("ReadFile(README.md) returned error: %v", err)
	}
	text := string(content)
	for _, want := range []string{
		"### Clear",
		"rmx clear",
		"rmx clr",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("README missing %q", want)
		}
	}
}
