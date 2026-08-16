package workspace

import (
	"testing"
)

func TestGovernanceFoundationIsExplicitlyOptIn(t *testing.T) {
	if _, err := NewSQLiteGovernance(SqlitePersistenceConfig{}); err == nil {
		t.Fatal("expected nil database to be rejected")
	}
	db := openWorkspaceTestDB(t)
	foundation, err := NewSQLiteGovernance(SqlitePersistenceConfig{DB: db})
	if err != nil {
		t.Fatal(err)
	}
	if foundation == nil {
		t.Fatal("expected governance foundation")
	}
}
