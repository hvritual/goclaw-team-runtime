package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEvidenceEnvelopeCanonicalChecksumAndImmutability(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	sourceA, err := NewEvidenceSource(
		"Runtime",
		"run-1",
		"runtime://workspace-1/runs/run-1",
		"event-17",
		strings.Repeat("a", 64),
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := NewEvidenceArtifact(
		"artifact://ci/builds/build-17",
		strings.Repeat("b", 64),
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := NewEvidenceEnvelope(
		"evidence-1",
		"workspace-1",
		EvidenceKindValidation,
		EvidenceOutcomePassed,
		sourceA,
		"validator-1",
		&artifact,
		json.RawMessage(`{"b":2,"a":1}`),
		now.Add(time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}

	sourceB, err := NewEvidenceSource(
		"runtime",
		"run-1",
		"runtime://workspace-1/runs/run-1",
		"event-17",
		strings.Repeat("a", 64),
		now.Add(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEvidenceEnvelope(
		"evidence-retry",
		"workspace-1",
		EvidenceKindValidation,
		EvidenceOutcomePassed,
		sourceB,
		"validator-1",
		&artifact,
		json.RawMessage("{\n  \"a\": 1, \"b\": 2\n}"),
		now.Add(2*time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	if first.SchemaVersion() != EvidenceEnvelopeSchemaV1 {
		t.Fatalf("schema = %q", first.SchemaVersion())
	}
	if got := string(first.Payload()); got != `{"a":1,"b":2}` {
		t.Fatalf("canonical payload = %q", got)
	}
	if first.ContentChecksum() == "" || first.ContentChecksum() != second.ContentChecksum() {
		t.Fatalf("semantic checksum mismatch: first=%q second=%q", first.ContentChecksum(), second.ContentChecksum())
	}

	payloadCopy := first.Payload()
	payloadCopy[2] = 'z'
	if got := string(first.Payload()); got != `{"a":1,"b":2}` {
		t.Fatalf("payload getter leaked mutable storage: %q", got)
	}
	artifactCopy := first.Artifact()
	artifactCopy.uri = "artifact://mutated/value"
	if got := first.Artifact().URI(); got != "artifact://ci/builds/build-17" {
		t.Fatalf("artifact getter leaked mutable storage: %q", got)
	}

	rehydrated, err := RehydrateEvidenceEnvelope(
		first.SchemaVersion(), first.ID(), first.WorkspaceID(), first.Kind(), first.Outcome(), first.Source(), first.ProducerID(),
		first.Artifact(), first.Payload(), first.CapturedAt(), first.ContentChecksum(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if rehydrated.ContentChecksum() != first.ContentChecksum() {
		t.Fatal("rehydrated checksum changed")
	}
	if _, err := RehydrateEvidenceEnvelope(
		first.SchemaVersion(), first.ID(), first.WorkspaceID(), first.Kind(), first.Outcome(), first.Source(), first.ProducerID(),
		first.Artifact(), first.Payload(), first.CapturedAt(), strings.Repeat("c", 64),
	); !errors.Is(err, ErrEvidenceChecksumMismatch) {
		t.Fatalf("checksum mismatch error = %v", err)
	}
	if _, err := RehydrateEvidenceEnvelope(
		"engineering.evidence/v2", first.ID(), first.WorkspaceID(), first.Kind(), first.Outcome(), first.Source(), first.ProducerID(),
		first.Artifact(), first.Payload(), first.CapturedAt(), first.ContentChecksum(),
	); !errors.Is(err, ErrEvidenceSchemaVersionInvalid) {
		t.Fatalf("schema mismatch error = %v", err)
	}
}

func TestEvidenceEnvelopeKindOutcomeMatrix(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	source, err := NewEvidenceSource("run", "run-1", "runtime://workspace-1/runs/run-1", "event-1", "", now)
	if err != nil {
		t.Fatal(err)
	}

	valid := []struct {
		kind    EvidenceKind
		outcome EvidenceOutcome
	}{
		{EvidenceKindValidation, EvidenceOutcomePassed},
		{EvidenceKindValidation, EvidenceOutcomeFailed},
		{EvidenceKindValidation, EvidenceOutcomeErrored},
		{EvidenceKindDeployment, EvidenceOutcomeSucceeded},
		{EvidenceKindDeployment, EvidenceOutcomeFailed},
		{EvidenceKindDeployment, EvidenceOutcomeRolledBack},
		{EvidenceKindTrace, EvidenceOutcomeObserved},
	}
	for _, tc := range valid {
		if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", tc.kind, tc.outcome, source, "producer-1", nil, nil, now); err != nil {
			t.Fatalf("valid %s/%s rejected: %v", tc.kind, tc.outcome, err)
		}
	}

	invalid := []struct {
		kind    EvidenceKind
		outcome EvidenceOutcome
	}{
		{EvidenceKindValidation, EvidenceOutcomeSucceeded},
		{EvidenceKindDeployment, EvidenceOutcomePassed},
		{EvidenceKindTrace, EvidenceOutcomeFailed},
		{EvidenceKind("unknown"), EvidenceOutcomeObserved},
	}
	for _, tc := range invalid {
		_, err := NewEvidenceEnvelope("evidence-1", "workspace-1", tc.kind, tc.outcome, source, "producer-1", nil, nil, now)
		if tc.kind == EvidenceKind("unknown") {
			if !errors.Is(err, ErrEvidenceKindInvalid) {
				t.Fatalf("invalid kind error = %v", err)
			}
			continue
		}
		if !errors.Is(err, ErrEvidenceOutcomeInvalid) {
			t.Fatalf("invalid %s/%s error = %v", tc.kind, tc.outcome, err)
		}
	}
}

func TestEvidenceEnvelopeRejectsWeakUnsafeOrMalformedEvidence(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)

	if _, err := NewEvidenceSource("run", "run-1", "runtime://workspace-1/runs/run-1", "", "", now); !errors.Is(err, ErrEvidenceSourceIdentityWeak) {
		t.Fatalf("weak source error = %v", err)
	}
	if _, err := NewEvidenceSource("run", "run-1", "https://token@example.com/runs/run-1", "event-1", "", now); !errors.Is(err, ErrEvidenceSourceLocatorInvalid) {
		t.Fatalf("credential-bearing locator error = %v", err)
	}
	if _, err := NewEvidenceSource("run", "run-1", "https://ci.example/runs/run-1?token=secret", "event-1", "", now); !errors.Is(err, ErrEvidenceSourceLocatorInvalid) {
		t.Fatalf("query-bearing locator error = %v", err)
	}
	if _, err := NewEvidenceSource("run", "run-1", "runtime://workspace-1/runs/run-1", "", "bad", now); !errors.Is(err, ErrEvidenceSourceDigestInvalid) {
		t.Fatalf("bad source digest error = %v", err)
	}
	if _, err := NewEvidenceArtifact("https://ci.example/artifact/1?token=secret", strings.Repeat("a", 64)); !errors.Is(err, ErrEvidenceArtifactURIInvalid) {
		t.Fatalf("unsafe artifact uri error = %v", err)
	}
	if _, err := NewEvidenceArtifact("artifact://ci/builds/build-1", "bad"); !errors.Is(err, ErrEvidenceArtifactDigestInvalid) {
		t.Fatalf("bad artifact digest error = %v", err)
	}

	source, err := NewEvidenceSource("run", "run-1", "runtime://workspace-1/runs/run-1", "event-1", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindTrace, EvidenceOutcomeObserved, source, "producer-1", nil, json.RawMessage(`[]`), now); !errors.Is(err, ErrEvidencePayloadObjectRequired) {
		t.Fatalf("array payload error = %v", err)
	}
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindTrace, EvidenceOutcomeObserved, source, "producer-1", nil, json.RawMessage(`{"a":1} trailing`), now); !errors.Is(err, ErrEvidencePayloadInvalid) {
		t.Fatalf("trailing payload error = %v", err)
	}
	largePayload := json.RawMessage(`{"blob":"` + strings.Repeat("x", MaxEvidencePayloadBytes) + `"}`)
	if _, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindTrace, EvidenceOutcomeObserved, source, "producer-1", nil, largePayload, now); !errors.Is(err, ErrEvidencePayloadTooLarge) {
		t.Fatalf("large payload error = %v", err)
	}

	emptyPayload, err := NewEvidenceEnvelope("evidence-1", "workspace-1", EvidenceKindTrace, EvidenceOutcomeObserved, source, "producer-1", nil, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(emptyPayload.Payload()); got != `{}` {
		t.Fatalf("empty payload normalized to %q", got)
	}
}
