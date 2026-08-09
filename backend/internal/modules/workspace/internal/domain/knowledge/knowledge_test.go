package knowledge

import (
	"errors"
	"testing"
	"time"
)

func TestKnowledgeStartsAsCandidate(t *testing.T) {
	value, err := New("knowledge-1", "workspace-1", "  Runbook  ", "summary", []string{" asset-1 "}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if value.Title != "Runbook" || value.Status != StatusCandidate || len(value.AssetIDs) != 1 || value.AssetIDs[0] != "asset-1" {
		t.Fatalf("Knowledge = %+v", value)
	}
	if _, err := New("knowledge-1", "workspace-1", "Runbook", "", []string{""}, time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank Asset error = %v", err)
	}
}
