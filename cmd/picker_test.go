package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"wrapux/internal/rmux"
)

func TestParseFzfTargetsReturnsHiddenFirstField(t *testing.T) {
	out := "codex/task\tcodex/task\t3m ago\nclaude/one\tclaude/one\t1h ago\n"

	targets, err := parseFzfTargets(out)
	if err != nil {
		t.Fatalf("parseFzfTargets returned error: %v", err)
	}

	want := []string{"codex/task", "claude/one"}
	if !reflect.DeepEqual(targets, want) {
		t.Fatalf("targets = %v, want %v", targets, want)
	}
}

func TestSessionPickerLinesIncludeMetadataAfterHiddenKey(t *testing.T) {
	sessions := []rmux.Session{
		{
			Name:         "codex/task",
			Windows:      2,
			Attached:     true,
			CreatedAt:    time.Unix(1700000000, 0),
			LastActiveAt: time.Unix(1700000300, 0),
		},
	}
	now := time.Unix(1700000600, 0)

	lines := sessionPickerLines(sessions, now)

	if len(lines) != 1 {
		t.Fatalf("lines = %v, want one line", lines)
	}
	parts := strings.Split(lines[0], "\t")
	if len(parts) < 5 {
		t.Fatalf("picker line has %d fields, want at least 5: %q", len(parts), lines[0])
	}
	if parts[0] != "codex/task" {
		t.Fatalf("hidden field = %q, want session name", parts[0])
	}
	if !strings.Contains(lines[0], "5m ago") {
		t.Fatalf("picker line %q does not include relative activity", lines[0])
	}
	if !strings.Contains(lines[0], "attached") {
		t.Fatalf("picker line %q does not include attached state", lines[0])
	}
}

func TestRelativeAgeRoundsToUsefulUnits(t *testing.T) {
	now := time.Unix(1700003600, 0)

	cases := []struct {
		name string
		then time.Time
		want string
	}{
		{name: "seconds", then: now.Add(-20 * time.Second), want: "20s ago"},
		{name: "minutes", then: now.Add(-5 * time.Minute), want: "5m ago"},
		{name: "hours", then: now.Add(-3 * time.Hour), want: "3h ago"},
		{name: "days", then: now.Add(-48 * time.Hour), want: "2d ago"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := relativeAge(tc.then, now); got != tc.want {
				t.Fatalf("relativeAge = %q, want %q", got, tc.want)
			}
		})
	}
}
