# Engineering Repository Manifest v1

`engineering.yaml` is a repository-owned self-description for durable engineering reality. It is not a project plan, deployment record, or replacement CMDB.

## Example

```yaml
schema_version: v1
entity:
  id: service-device-gateway
  type: service
  name: Device Gateway
  status: active
  owner_ref: team:iot-platform
source:
  type: github
  locator: github://acme/device-gateway
domains:
  - connectivity
dependencies:
  - service-device-session
relations:
  - relation: part_of
    target: system-device-cloud
interfaces:
  - id: api-device-session-v1
    type: api
    direction: provides
  - id: thing-model-coffee-machine-v3
    type: thing_model
    direction: uses
knowledge:
  architecture:
    - docs/architecture.md
  adr:
    - docs/adr/ADR-001.md
  standards:
    - docs/standards/mqtt.md
  runbooks:
    - docs/runbook.md
```

## Authority boundary

The repository manifest states how the repository describes its durable engineering identity. A future reconciler records the GitHub repository/revision as provenance. The manifest cannot declare its own authority class and cannot fabricate work, release, deployment, incident, or runtime facts.

V1 source locators use canonical `github://owner/repository` form. Credentials and tokens are never stored in the manifest.

## Allowed entity types

V1 reuses the Engineering ontology: `product`, `engineering_system`, `application`, `service`, `component`, `repository`, `api`, `thing_model`, `environment`.

## Dependencies, relations, and interfaces

`dependencies` is the sole V1 authoring surface for `depends_on`. Interfaces are the sole V1 authoring surface for `provides`/`uses` API and thing-model references. Keeping these relations in dedicated fields prevents two equivalent encodings from producing graph ambiguity.

The generic `relations` list is limited to durable structural/governance relations: `part_of`, `implements`, `constrains`, `governs`, `operates`.

Dynamic relations such as `changes`, `affects`, `introduced_by`, `included_in`, and `deployed_to` must come from their owning work/source/runtime systems and are rejected by the V1 parser.

Interface IDs must be unique inside one manifest. Dependencies must be unique and cannot point back to the manifest entity itself.

## Knowledge references

Knowledge references are repository-relative paths only. Absolute paths, URLs, backslashes, empty segments, and `.`/`..` traversal are rejected.

## Strictness and determinism

The parser rejects unknown fields, multiple YAML documents, unsupported source schemes, unsafe knowledge paths, duplicate canonical values, and manifests larger than 256 KiB. It normalizes source locator casing, domain identifiers, dependency/relation/interface ordering, and knowledge-reference ordering.

The SHA-256 checksum is calculated from the normalized representation, so semantically identical YAML field/list ordering produces the same manifest checksum.

## V1 scope

V1 intentionally describes one primary EngineeringEntity per repository manifest. Multi-entity monorepo declarations are deferred until the reconciliation model proves a stable need; a repository may still reference existing EngineeringEntity IDs through dependencies, relations, and interfaces. This keeps the first source-backed contract small and deterministic.
