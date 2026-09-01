package application

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

func TestEvidenceApplicationAuthorizationIsolationAndImmutability(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}
	member := contract.Actor{UserID: "member"}
	outsider := contract.Actor{UserID: "outsider"}

	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	subject := contract.NodeRef{Kind: "engineering_entity", ID: "service-1"}
	request := contract.RecordEvidenceRequest{
		ID: "evidence-1", Kind: "build", Subject: subject,
		Source: contract.EvidenceSource{
			SourceType: "ci", Locator: "https://ci.example/builds/7", Revision: "build-7", Digest: strings.Repeat("a", 64),
		},
		ProducerID: "ci/github-actions",
		ArtifactURI: "https://artifacts.example/builds/7/binary",
		ArtifactDigest: strings.Repeat("b", 64),
	}

	created, err := service.RecordEvidence(ctx, owner, "workspace-a", request)
	if err != nil {
		t.Fatal(err)
	}
	if created.ContentChecksum == "" || created.Source.ObservedAt.IsZero() || created.CapturedAt.IsZero() {
		t.Fatalf("created evidence = %+v", created)
	}
	if _, err := service.RecordEvidence(ctx, member, "workspace-a", request); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("member write error = %v", err)
	}
	read, err := service.GetEvidence(ctx, member, "workspace-a", "evidence-1")
	if err != nil {
		t.Fatalf("member read: %v", err)
	}
	if read.ContentChecksum != created.ContentChecksum {
		t.Fatalf("read checksum=%q want=%q", read.ContentChecksum, created.ContentChecksum)
	}
	if _, err := service.GetEvidence(ctx, outsider, "workspace-a", "evidence-1"); !errors.Is(err, contract.ErrForbidden) {
		t.Fatalf("outsider read error = %v", err)
	}
	if _, err := service.GetEvidence(ctx, owner, "workspace-b", "evidence-1"); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("cross-workspace evidence read error = %v", err)
	}
	if _, err := service.RecordEvidence(ctx, owner, "workspace-b", request); !errors.Is(err, contract.ErrNotFound) {
		t.Fatalf("cross-workspace subject error = %v", err)
	}

	replayed, err := service.RecordEvidence(ctx, owner, "workspace-a", request)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if replayed.ContentChecksum != created.ContentChecksum || !replayed.CapturedAt.Equal(created.CapturedAt) {
		t.Fatalf("replayed evidence mutated persisted identity: created=%+v replayed=%+v", created, replayed)
	}

	conflicting := request
	conflicting.Source.Revision = "build-8"
	if _, err := service.RecordEvidence(ctx, owner, "workspace-a", conflicting); !errors.Is(err, contract.ErrConflict) {
		t.Fatalf("conflicting evidence error = %v", err)
	}

	values, err := service.ListEvidence(ctx, member, "workspace-a", &subject)
	if err != nil || len(values) != 1 || values[0].ID != "evidence-1" {
		t.Fatalf("subject evidence = %+v err=%v", values, err)
	}
}

func TestEvidenceDoesNotAcceptChange(t *testing.T) {
	service, _, closeDB := newTestService(t)
	defer closeDB()
	ctx := context.Background()
	owner := contract.Actor{UserID: "owner"}

	if _, err := service.CreateEntity(ctx, owner, "workspace-a", contract.CreateEntityRequest{ID: "service-1", Type: "service", Name: "Device Gateway", Status: "active"}); err != nil {
		t.Fatal(err)
	}
	change, err := service.CreateChange(ctx, owner, "workspace-a", contract.CreateChangeRequest{
		ID: "change-1", Summary: "Change gateway", AffectedEntityIDs: []string{"service-1"},
		Provenance: contract.Provenance{SourceType: "github", Locator: "github://acme/device-gateway/pull/42", Revision: strings.Repeat("c", 40)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if change.Status != "proposed" {
		t.Fatalf("initial change status=%q", change.Status)
	}
	_, err = service.RecordEvidence(ctx, owner, "workspace-a", contract.RecordEvidenceRequest{
		ID: "evidence-change-1", Kind: "source_change", Subject: contract.NodeRef{Kind: "change", ID: "change-1"},
		Source: contract.EvidenceSource{SourceType: "github", Locator: "github://acme/device-gateway/pull/42", Revision: strings.Repeat("d", 40)},
		ProducerID: "github",
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetChange(ctx, owner, "workspace-a", "change-1")
	if err != nil {
		t.Fatal(err)
	}
	if current.Status != "proposed" || current.AcceptedAt != nil {
		t.Fatalf("recording evidence mutated Change lifecycle: %+v", current)
	}
}
