package manifest

import (
	"errors"
	"os"
	"testing"
)

func TestParseNormalizesAndHashesManifestDeterministically(t *testing.T) {
	data, err := os.ReadFile("testdata/engineering.valid.yaml")
	if err != nil {
		t.Fatal(err)
	}
	first, err := Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	if first.SchemaVersion != SchemaVersionV1 || first.Entity.ID != "service-device-gateway" || first.Source.Locator != "github://acme/device-gateway" {
		t.Fatalf("manifest = %+v", first)
	}
	if len(first.Domains) != 2 || first.Domains[0] != "connectivity" || first.Domains[1] != "telemetry" {
		t.Fatalf("domains = %#v", first.Domains)
	}
	if len(first.Dependencies) != 1 || first.Dependencies[0] != "service-device-session" {
		t.Fatalf("dependencies = %#v", first.Dependencies)
	}
	if len(first.Relations) != 1 || first.Relations[0].Relation != "part_of" {
		t.Fatalf("relations = %#v", first.Relations)
	}
	if len(first.Interfaces) != 2 || first.Interfaces[0].Direction != "provides" || first.Interfaces[1].Direction != "uses" {
		t.Fatalf("interfaces = %#v", first.Interfaces)
	}
	if first.Checksum() == "" {
		t.Fatal("checksum is empty")
	}

	reordered := []byte(`schema_version: v1
source:
  locator: github://acme/device-gateway
  type: github
entity:
  status: active
  name: Device Gateway
  type: service
  id: service-device-gateway
  owner_ref: team:iot-platform
interfaces:
  - direction: uses
    type: thing_model
    id: thing-model-coffee-machine-v3
  - direction: provides
    type: api
    id: api-device-session-v1
relations:
  - target: system-device-cloud
    relation: part_of
dependencies: [service-device-session]
domains: [connectivity, telemetry]
knowledge:
  runbooks: [docs/runbook.md]
  architecture: [docs/architecture.md]
  adr: [docs/adr/ADR-001.md]
  standards: [docs/standards/mqtt.md]
`)
	second, err := Parse(reordered)
	if err != nil {
		t.Fatal(err)
	}
	if first.Checksum() != second.Checksum() {
		t.Fatalf("checksum mismatch: %s != %s", first.Checksum(), second.Checksum())
	}
}

func TestParseRejectsUnknownFieldsAndMultipleDocuments(t *testing.T) {
	for name, data := range map[string][]byte{
		"unknown": []byte(`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active, unexpected: true}
source: {type: github, locator: github://acme/a}
`),
		"multiple": []byte(`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
---
schema_version: v1
entity: {id: service-b, type: service, name: B, status: active}
source: {type: github, locator: github://acme/b}
`),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(data); !errors.Is(err, ErrInvalidManifest) {
				t.Fatalf("error = %v, want invalid manifest", err)
			}
		})
	}
}

func TestParseRejectsInvalidOntologyAndSourceValues(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want error
	}{
		{
			name: "entity type",
			yaml: `schema_version: v1
entity: {id: service-a, type: galaxy, name: A, status: active}
source: {type: github, locator: github://acme/a}
`,
			want: ErrInvalidManifest,
		},
		{
			name: "dynamic relation",
			yaml: `schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
relations:
  - {relation: changes, target: service-b}
`,
			want: ErrUnsupportedManifestRelation,
		},
		{
			name: "dependency relation must use dependencies field",
			yaml: `schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
relations:
  - {relation: depends_on, target: service-b}
`,
			want: ErrUnsupportedManifestRelation,
		},
		{
			name: "source scheme",
			yaml: `schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: https://github.com/acme/a}
`,
			want: ErrInvalidSourceLocator,
		},
		{
			name: "unsafe knowledge path",
			yaml: `schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
knowledge:
  runbooks: [../runbook.md]
`,
			want: ErrUnsafeKnowledgePath,
		},
		{
			name: "interface type",
			yaml: `schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
interfaces:
  - {id: queue-a, type: queue, direction: provides}
`,
			want: ErrInvalidInterface,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse([]byte(test.yaml)); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestParseRejectsDuplicatesAfterNormalization(t *testing.T) {
	tests := []string{
		`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
domains: [Connectivity, connectivity]
`,
		`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
dependencies: [service-b, service-b]
`,
		`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
interfaces:
  - {id: api-a, type: api, direction: provides}
  - {id: api-a, type: api, direction: uses}
`,
		`schema_version: v1
entity: {id: service-a, type: service, name: A, status: active}
source: {type: github, locator: github://acme/a}
knowledge:
  runbooks: [docs/runbook.md, docs/runbook.md]
`,
	}
	for index, input := range tests {
		if _, err := Parse([]byte(input)); !errors.Is(err, ErrDuplicateValue) {
			t.Fatalf("case %d error = %v, want duplicate", index, err)
		}
	}
}

func TestParseRejectsOversizedDocument(t *testing.T) {
	input := make([]byte, MaxManifestBytes+1)
	if _, err := Parse(input); !errors.Is(err, ErrInvalidManifest) {
		t.Fatalf("error = %v, want invalid manifest", err)
	}
}
