package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRewriteTree(t *testing.T) {
	tests := []struct {
		name         string
		relativePath string
		content      string
		wantCount    int
		wantError    bool
	}{
		{
			name:         "marked dddgen file",
			relativePath: "internal/modules/workspace/project_service.go",
			content:      generatedMarker + "\npackage workspace\nimport _ \"" + oldPrefix + "/workspace/v1\"\n",
			wantCount:    1,
		},
		{
			name:         "primary module base file",
			relativePath: "internal/modules/auth/module.go",
			content:      "package auth\nimport _ \"" + oldPrefix + "/auth/v1\"\n",
			wantCount:    1,
		},
		{
			name:         "primary proto",
			relativePath: "api/space/v1/space.proto",
			content:      "option go_package = \"" + oldPrefix + "/space/v1;spacev1\";\n",
			wantCount:    1,
		},
		{
			name:         "unknown file rejected",
			relativePath: "internal/modules/system/custom.go",
			content:      "package system\nimport _ \"" + oldPrefix + "/system/v1\"\n",
			wantError:    true,
		},
		{
			name:         "already normalized",
			relativePath: "internal/modules/workspace/module.go",
			content:      "package workspace\nimport _ \"" + newPrefix + "/workspace/v1\"\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			for _, requiredRoot := range []string{"api", filepath.Join("internal", "modules")} {
				if err := os.MkdirAll(filepath.Join(root, requiredRoot), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(root, test.relativePath)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(test.content), 0o640); err != nil {
				t.Fatal(err)
			}

			count, err := rewriteTree(root)
			if (err != nil) != test.wantError {
				t.Fatalf("rewriteTree() error = %v, wantError %v", err, test.wantError)
			}
			if count != test.wantCount {
				t.Fatalf("rewriteTree() count = %d, want %d", count, test.wantCount)
			}

			updated, readErr := os.ReadFile(path)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if test.wantError {
				if string(updated) != test.content {
					t.Fatal("rejected file was modified")
				}
				return
			}
			if strings.Contains(string(updated), oldPrefix) {
				t.Fatal("old prefix remains after rewrite")
			}

			secondCount, secondErr := rewriteTree(root)
			if secondErr != nil || secondCount != 0 {
				t.Fatalf("second rewriteTree() = %d, %v; want idempotent success", secondCount, secondErr)
			}
		})
	}
}
