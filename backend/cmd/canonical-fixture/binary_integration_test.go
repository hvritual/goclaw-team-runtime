package main_test

import (
	"context"
	"database/sql"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/auth"
	"github.com/hvritual/workspace/internal/modules/workspace"
	_ "modernc.org/sqlite"
)

func TestCanonicalFixtureBinaryRegistersIssueSearchNormalization(t *testing.T) {
	databasePath := filepath.Join(t.TempDir(), "canonical.db")
	db, err := sql.Open("sqlite", databasePath+"?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatal(err)
	}
	if err := workspace.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := auth.MigrateSqlite(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-sqlite-path", databasePath)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("canonical fixture binary failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `"result":"created"`) {
		t.Fatalf("canonical fixture output = %s", output)
	}
}
