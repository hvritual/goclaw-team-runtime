package cli

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAccessToken(t *testing.T) {
	token, err := generateAccessToken()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 32 {
		t.Fatalf("decoded token length = %d, want 32", len(raw))
	}
}

func TestStageSecretFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "token")
	cleanup, err := stageSecretFile(path, "secret")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != "secret" {
		t.Fatalf("token content = %q", data)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("token permissions = %o, want 600", info.Mode().Perm())
	}
	if _, err := stageSecretFile(path, "replacement"); err == nil {
		t.Fatal("expected existing token file to be rejected")
	}
	cleanup()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("cleanup did not remove token file: %v", err)
	}
}
