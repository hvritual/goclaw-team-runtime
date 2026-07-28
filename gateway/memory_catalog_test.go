package gateway

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/smallnest/goclaw/governance"
	"github.com/smallnest/goclaw/memory/catalog"
)

func TestMemoryCatalogGatewaySeparatesProposalAndApproval(t *testing.T) {
	cfg := catalog.DefaultConfig()
	cfg.DatabasePath = filepath.Join(t.TempDir(), "catalog.db")
	service, err := catalog.NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer service.Close()

	handler := &Handler{registry: NewMethodRegistry()}
	handler.SetMemoryCatalog(service)
	created, err := handler.registry.Call(
		"memory.catalog.candidate.create",
		"agent-session",
		map[string]interface{}{
			"project_id": "alpha",
			"title":      "Retain evidence",
			"content":    "Every durable memory keeps a stable source URI.",
			"kind":       "constraint",
			"source_uri": "trace:test",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created == nil {
		t.Fatal("unexpected empty create result")
	}

	records, err := service.List("alpha", catalog.StatusPending, 10)
	if err != nil || len(records) != 1 {
		t.Fatalf("pending=%d err=%v", len(records), err)
	}
	review := governance.Review{
		ReviewerID:    "reviewer",
		Rationale:     "Evidence and project scope were verified.",
		Role:          governance.RoleMemoryApprove,
		Source:        "local-cli",
		Authenticated: true,
		CreatedAt:     time.Now().UTC(),
	}
	if _, err := service.ApproveCandidate(records[0].ID, review); err != nil {
		t.Fatal(err)
	}
	status, err := handler.registry.Call(
		"memory.catalog.status",
		"session",
		map[string]interface{}{"project_id": "alpha"},
	)
	if err != nil || status == nil {
		t.Fatalf("status=%v err=%v", status, err)
	}
}
