# IoT Engineering Digital Thread — Phase 3 Plan v4

**Status:** ACTIVE — P3-S01 clean restart  
**Target branch:** `agent/iot-edt-p3-s01-evidence-envelope-v2`  
**Canonical base:** `codex/multica-six-domain-baseline@f590d56fe5cc019489ac61ac378f1fffba55ee50`  
**Superseded implementation branch:** `agent/iot-edt-p3-s01-evidence-envelope` (audit history only; do not merge)

## Phase 3 goal

Turn Engineering Thread execution output into evidence-grade, machine-readable records without turning Engineering into a project-management domain. Phase 3 owns execution evidence and later derives execution-item state from that evidence; it must not infer or mutate requirement/design maturity, sprint state, milestone state, or cross-domain business workflow.

## Frozen slice sequence

| Slice | Delivery | State |
| --- | --- | --- |
| P3-S01 | Normalized Evidence Envelope | IN_PROGRESS |
| P3-S02 | Execution Item storage + evidence attachment | NOT_STARTED |
| P3-S03 | Evidence ingestion/read API | NOT_STARTED |
| P3-S04 | Deterministic status derivation | NOT_STARTED |
| P3-S05 | Engineering execution UI | NOT_STARTED |
| P3-S06 | Hardening + Phase 3 exit certification | NOT_STARTED |

## P3-S01 objective

Freeze one immutable normalized evidence envelope that later slices can persist, attach, ingest, and evaluate without changing its semantics.

P3-S01 delivers only:

- public Engineering evidence contract;
- domain evidence value model;
- evidence kind/outcome normalization and compatibility validation;
- immutable source provenance with stable source identity;
- optional content-addressed artifact reference;
- canonical JSON object payload;
- deterministic semantic SHA-256 checksum;
- unit/contract tests.

## P3-S01 hard boundary

P3-S01 does **not** add:

- execution-item aggregate or tables;
- evidence repository, GraphDB edge, SQLite migration, or other persistence;
- HTTP/MCP ingestion or read APIs;
- authorization callbacks or subject lookup services;
- execution-item status derivation;
- frontend changes;
- requirement/design/project/sprint/milestone mutation.

The envelope is intentionally **subject-agnostic**. An evidence record describes an immutable engineering fact. Association of that fact with an `ExecutionItem` is owned by P3-S02/P3-S03. This prevents the envelope model from pulling persistence, authorization, or cross-domain subject verification into P3-S01.

## Normalized Evidence Envelope v1

Schema identifier: `engineering.evidence/v1`.

Required semantic fields:

- workspace scope;
- evidence kind;
- normalized outcome;
- producer identity;
- immutable source reference (`type`, Engineering-owned source id, canonical locator, revision and/or SHA-256 digest);
- canonical JSON object payload.

Optional semantic field:

- artifact URI + SHA-256 digest, supplied as an atomic pair.

Capture metadata:

- envelope id;
- source observation time;
- capture time.

The semantic content checksum excludes envelope id and timestamps so retrying capture of the same immutable fact is content-identical. It includes workspace, schema, kind, outcome, producer, source identity, artifact identity, and canonical payload.

### Initial evidence kinds

- `validation`
- `deployment`
- `trace`

### Normalized outcomes

- validation: `passed`, `failed`, `errored`
- deployment: `succeeded`, `failed`, `rolled_back`
- trace: `observed`

These are evidence facts, not execution-item status policy. P3-S04 owns the mapping from evidence facts to execution-item state.

## Validation invariants

1. Workspace, envelope id, producer, source type/id/locator are non-empty after trimming.
2. Source provenance has an immutable revision or SHA-256 digest.
3. Source/artifact locators are canonical absolute URIs without credentials, query strings, or fragments.
4. Artifact URI and digest are either both absent or both present.
5. Payload is a JSON object, canonicalized before hashing, and limited to 256 KiB after canonicalization.
6. Kind/outcome combinations outside the frozen matrix are rejected.
7. Rehydration recomputes and verifies the semantic checksum.
8. Domain getters do not expose mutable payload backing storage.

## P3-S01 acceptance

Implementation diff is limited to this plan plus Engineering evidence contract/domain/tests. There must be no repository, migration, service, API, authorization, or UI diff.

Validation commands:

```bash
cd backend
go test ./internal/modules/engineering/...
go test ./...
go build ./...
```

Architecture and temporary-rule checks remain required before integration according to the existing Engineering canonical gate.

P3-S01 is complete only after tests are green and the resulting feature commit/PR remains strictly within this slice boundary.