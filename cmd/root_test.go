package cmd

import (
	"strings"
	"testing"
)

func TestRootCommandBranding(t *testing.T) {
	if rootCmd.Use != "rmx" {
		t.Fatalf("Use = %q, want rmx", rootCmd.Use)
	}
	if !strings.Contains(rootCmd.Short, "rmux session manager") {
		t.Fatalf("Short = %q, want rmux session manager wording", rootCmd.Short)
	}
	legacy := "wrap" + "per"
	if strings.Contains(strings.ToLower(rootCmd.Short), legacy) {
		t.Fatalf("Short = %q, should not use old %s wording", rootCmd.Short, legacy)
	}
}
