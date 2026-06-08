package cmd

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wrapux/internal/rmux"
)

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
	if strings.Count(got, "wrapux capture") != 2 {
		t.Fatalf("rendered output = %q, want two capture headers", got)
	}
}
