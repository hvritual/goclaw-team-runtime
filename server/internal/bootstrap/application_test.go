package bootstrap

import "testing"

func TestApplicationRegistersFourAcceptedModules(t *testing.T) {
	application := NewApplication()
	if got := len(application.Modules()); got != 4 {
		t.Fatalf("module count = %d, want 4", got)
	}
}
