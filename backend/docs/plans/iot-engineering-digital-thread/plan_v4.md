# IoT Engineering Digital Thread — Phase 3 Plan v4

**Status:** ACTIVE — P3-S02 execution-item storage + evidence attachment  
**Target branch:** `agent/iot-edt-p3-s02-execution-item-evidence-attachment`  
**Canonical base:** `codex/multica-six-domain-baseline@49b061f06a651afee6f0220629e2993879ac174b`  
**P3-S01 integration:** PR #19 merged at canonical commit `49b061f06a651afee6f0220629e2993879ac174b`  
**Superseded P3-S01 branch:** `agent/iot-edt-p3-s01-evidence-envelope` (audit history only; do not merge)

## Phase 3 goal

Turn Engineering Thread execution output into evidence-grade, machine-readable records without turning Engineering into a project-management or execution-authority domain. Runtime, CI, release, deployment and observability systems continue to own their native execution truth. Engineering stores normalized immutable evidence, canonical projection identities and explicit relationships required for traceability.

## Frozen slice sequence

| Slice | Delivery | State |
| --- | --- | --- |
| P3-S01 | Normalized Evidence Envelope | COMPLETE |
| P3-S02 | Execution Item storage + evidence attachment | IN_PROGRESS |
| P3-S03 | Evidence ingestion/read API | NOT_STARTED |
| P3-S04 | Deterministic status derivation | NOT_STARTED |
| P3-S05 | Engineering execution UI | NOT_STARTED |
| P3-S06 | Hardening + Phase 3 exit certification | NOT_STARTED |

## P3-S01 accepted foundation

Schema identifier: `engineering.evidence/v1`.

The accepted envelope is intentionally subject-agnostic and immutable. It contains workspace scope, evidence kind/outcome, producer identity, immutable source provenance, optional content-addressed artifact, canonical JSON-object payload, capture metadata and a deterministic semantic SHA-256 checksum.

Initial evidence kinds/outcomes remain frozen:

- `validation`: `passed`, `failed`, `errored`
- `deployment`: `succeeded`, `failed`, `rolled_back`
- `trace`: `observed`

Evidence facts do not directly mutate or derive execution-item status. P3-S04 owns status policy.

## P3-S02 objective

Add the durable Engineering-side storage boundary required to persist normalized evidence and associate it with a stable execution projection without transferring source-of-truth ownership from Runtime, CI, release, deployment or observability systems.

P3-S02 delivers only:

- immutable `ExecutionItem` projection aggregate;
- immutable `EvidenceAttachment` relation value;
- repository ports for execution items, evidence envelopes and attachments;
- SQLite persistence for those records;
- adapter-local migration `000002_execution_evidence`;
- deterministic round-trip, isolation, conflict and attachment-integrity tests.

## ExecutionItem ontology

`ExecutionItem` is a canonical Engineering projection anchor for one externally-owned execution record. It is **not** Workspace Task/Todo, Runtime Run/Attempt, Change, or a new source of execution status.

Required fields:

- workspace ID;
- stable Engineering execution-item ID;
- execution kind;
- source type;
- source-native ID;
- canonical non-secret source locator;
- projection creation time.

Initial execution kinds:

- `run`
- `build`
- `test`
- `release`
- `deployment`
- `observation`

An execution item has no status field in P3-S02. Native source state remains source-owned; normalized evidence is attached separately and P3-S04 later derives an Engineering projection status deterministically.

## Evidence persistence and attachment

`EvidenceEnvelope` remains exactly the P3-S01 subject-agnostic value. S02 persists it without adding an execution subject to the envelope.

`EvidenceAttachment` is a separate immutable relation with:

- workspace ID;
- execution-item ID;
- evidence-envelope ID;
- attachment time.

The repository must enforce at application/adapter level that both records exist in the same workspace before creating the relation. The Engineering SQLite adapter intentionally introduces no foreign keys or cascades, preserving the existing federated-boundary invariant.

Storage operations are additive/immutable:

- create/get ExecutionItem;
- create/get EvidenceEnvelope;
- attach evidence to ExecutionItem;
- list attachment records for one ExecutionItem.

There are no update/delete/status-transition operations in this slice.

## P3-S02 invariants

1. Execution-item/workspace/source identity fields are non-empty after normalization.
2. Execution kind is restricted to the frozen initial vocabulary.
3. Source type is lowercase-normalized and source locator obeys the same canonical, secret-free absolute-URI rule used by Evidence v1.
4. Duplicate writes of the exact same immutable identity are idempotent; conflicting reuse of an ID is rejected.
5. Evidence rehydration must pass the P3-S01 semantic checksum verification before a record is returned.
6. An attachment cannot be created unless both execution item and evidence exist in the requested workspace.
7. Cross-workspace attachment attempts fail closed without revealing a foreign workspace record.
8. Reattaching the same execution-item/evidence pair is idempotent and preserves the first attachment relation.
9. SQLite migration adds no foreign keys or cascade behavior.
10. P3-S02 does not derive or persist execution status.

## P3-S02 hard boundary

P3-S02 does **not** add:

- HTTP/gRPC/MCP ingestion or read endpoints;
- workspace authorization or external subject lookup services;
- producer authentication;
- status derivation or transition policy;
- Runtime Run/Attempt mutation;
- Change acceptance mutation;
- GraphDB projection;
- frontend changes;
- requirement/design/project/sprint/milestone mutation;
- autonomous knowledge/fact publication.

## P3-S02 acceptance

Expected implementation scope:

- `backend/docs/plans/iot-engineering-digital-thread/plan_v4.md`;
- Engineering domain execution/attachment model and tests;
- Engineering repository ports;
- Engineering SQLite migration/repository/tests.

Validation commands:

```bash
cd backend
go test ./internal/modules/engineering/... -count=1
go vet ./internal/modules/engineering/...
go test ./...
go build ./...
```

Canonical repository gates remain mandatory, including deterministic backend checks and race tests. P3-S02 is complete only after the feature PR is scope-clean and green on the exact canonical base.

## Deferred sequence

P3-S03 owns authenticated/authorized evidence ingestion and read transport. It must perform normalization server-side; clients must not become the authority for canonical locator normalization or semantic checksum calculation.

P3-S04 owns deterministic mapping from accumulated normalized evidence facts to Engineering execution-item projection status. It must not mutate Runtime source truth or Workspace DoneGate semantics.
