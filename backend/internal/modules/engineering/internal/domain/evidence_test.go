package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEvidenceEnvelopeIsDeterministicAndImmutable(t *testing.T) {
	now := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	subject, err := NewNodeRef(NodeKindRun, "run-1")
	if err != nil {
		t.Fatal(err)
	}
	source, err := NewEvidenceSource("runtime", "runtime://workspace-1/project-1/evidence-1", "event-17", strings.Repeat("a", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	first, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindExecution, subject, source, "runner-1", "artifact://runtime/550e8400-e29b-41d4-a716-446655440000", strings.Repeat("b", 64), now)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEvidenceEnvelope("evidence-other-id", "workspace-1", EvidenceKindExecution, subject, source, "runner-1", "artifact://runtime/550e8400-e29b-41d4-a716-446655440000", strings.Repeat("b", 64), now.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentChecksum() == "" || first.ContentChecksum() != second.ContentChecksum() {
		t.Fatalf("semantic evidence checksum must ignore envelope id/capture time: first=%q second=%q", first.ContentChecksum(), second.ContentChecksum())
	}
	if _, err := RehydrateEvidenceEnvelope(first.ID(), first.WorkspaceID(), first.Kind(), first.Subject(), first.Source(), first.ProducerID(), first.ArtifactURI(), first.ArtifactDigest(), first.CapturedAt(), strings.Repeat("c", 64)); !errors.Is(err, ErrEvidenceChecksumMismatch) {
		t.Fatalf("rehydrate mismatch error = %v", err)
	}
}

func TestEvidenceEnvelopeRejectsWeakOrUnsafeEvidence(t *testing.T) {
	now := time.Date(2026, 9, 1, 15, 0, 0, 0, time.UTC)
	subject, err := NewNodeRef(NodeKindChange, "change-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEvidenceSource("github", "github://acme/device-gateway/pull/42", "", "", now); !errors.Is(err, ErrEvidenceSourceIdentityWeak) {
		t.Fatalf("weak source error = %v", err)
	}
	if _, err := NewEvidenceSource("ci", "https://token@example.com/build/7", "build-7", "", now); !errors.Is(err, ErrEvidenceSourceLocatorInvalid) {
		t.Fatalf("credential-bearing locator error = %v", err)
	}
	source, err := NewEvidenceSource("github", "github://acme/device-gateway/pull/42", strings.Repeat("d", 40), "", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKind("unknown"), subject, source, "github", "", "", now); !errors.Is(err, ErrEvidenceKindInvalid) {
		t.Fatalf("unknown kind error = %v", err)
	}
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindSourceChange, subject, source, "github", "https://ci.example/artifact/1?token=secret", strings.Repeat("e", 64), now); !errors.Is(err, ErrEvidenceArtifactURIInvalid) {
		t.Fatalf("secret-bearing artifact uri error = %v", err)
	}
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindSourceChange, subject, source, "github", "https://ci.example/artifact/1", "bad", now); !errors.Is(err, ErrEvidenceArtifactInvalid) {
		t.Fatalf("artifact digest error = %v", err)
	}
}
