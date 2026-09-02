package contract

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEvidenceEnvelopeV1WireContract(t *testing.T) {
	now := time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC)
	value := EvidenceEnvelope{
		SchemaVersion: EvidenceEnvelopeSchemaV1,
		ID:            "evidence-1",
		WorkspaceID:   "workspace-1",
		Kind:          EvidenceKindValidation,
		Outcome:       EvidenceOutcomePassed,
		Source: EvidenceSource{
			Type:       "run",
			ID:         "run-1",
			Locator:    "runtime://workspace-1/runs/run-1",
			Revision:   "event-17",
			Digest:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ObservedAt: now,
		},
		ProducerID: "validator-1",
		Artifact: &EvidenceArtifact{
			URI:    "artifact://ci/builds/build-17",
			Digest: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Payload:         json.RawMessage(`{"suite":"go test"}`),
		CapturedAt:      now.Add(time.Minute),
		ContentChecksum: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	}

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"schema_version":"engineering.evidence/v1","id":"evidence-1","workspace_id":"workspace-1","kind":"validation","outcome":"passed","source":{"type":"run","id":"run-1","locator":"runtime://workspace-1/runs/run-1","revision":"event-17","digest":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","observed_at":"2026-09-02T00:00:00Z"},"producer_id":"validator-1","artifact":{"uri":"artifact://ci/builds/build-17","digest":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},"payload":{"suite":"go test"},"captured_at":"2026-09-02T00:01:00Z","content_checksum":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"}`
	if string(encoded) != expected {
		t.Fatalf("wire contract changed:\nwant %s\n got %s", expected, encoded)
	}
}
