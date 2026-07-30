package main

import (
	"testing"
)

func TestLegacyCompatibilityCommandsRemainAvailable(t *testing.T) {
	t.Run("workspace get remains available", func(t *testing.T) {
		if _, _, err := workspaceCmd.Find([]string{"get"}); err != nil {
			t.Fatalf("expected workspace get command to exist: %v", err)
		}
	})

	t.Run("workspace member list remains available", func(t *testing.T) {
		if _, _, err := workspaceCmd.Find([]string{"member", "list"}); err != nil {
			t.Fatalf("expected workspace member list command to exist: %v", err)
		}
	})

}
