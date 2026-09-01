# IoT Engineering Digital Thread — Implementation Plan v4

- Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`
- Version: `4`
- Status: **Approved / executing**
- Parent plans: `plan_v1.md`, `plan_v2.md`, `plan_v3.md`
- Canonical branch: `codex/multica-six-domain-baseline`
- Certified Phase 2 canonical: `f590d56fe5cc019489ac61ac378f1fffba55ee50`
- Activation: explicit user instruction on 2026-09-01 to enter the next phase immediately

## 1. Purpose

Phase 3 establishes requirement-to-runtime traceability beyond source context by normalizing immutable evidence, rebuilding facts from that evidence, and connecting accepted changes to build, release, deployment and runtime feedback.

Target chain:

`Requirement -> Task -> Run -> Change -> PR -> Build -> Release -> Deployment -> Observation/Incident -> Fact -> KnowledgeSuggestion`.

The federated source-of-truth model remains mandatory. Runtime owns execution attempts and Runner execution evidence; GitHub owns repository/commit/PR identity; CI owns build/test evidence; release systems own release artifacts; deployment systems own deployment state; observability systems own observations/incidents; Engineering owns canonical cross-source evidence identity, provenance-bearing projections and thread relationships only.

`Evidence != Fact != Knowledge`. Evidence is immutable source material, Fact is a rebuildable claim derived from evidence, and governed Knowledge remains a reviewed organizational conclusion.

## 2. Phase 2 closure baseline

Phase 2 is closed on canonical product SHA `f590d56fe5cc019489ac61ac378f1fffba55ee50` after final E2E certification run `33519792124` passed governance, deterministic `make check`, canonical race tests, frontend aggregate and `CI / required`.

The Phase 2 Exit harness was validation-only and was not merged into canonical.

No Phase 3 work may reinterpret a successful Run as Change acceptance or Knowledge publication.

## 3. Phase 3 execution sequence

Execution order is strict:

`P3-S01 -> P3-S02 -> P3-S03 -> P3-S04 -> P3-S05 -> P3-S06 -> P3-EXIT`.

Only `P3-S01` is active in this plan revision. Later steps are dependency-staged and may not be implemented until their predecessor is accepted.

### P3-S01 — Normalized Evidence Envelope

#### Goal

Introduce one provider-neutral, immutable evidence envelope that can represent evidence originating from Runtime, GitHub, CI, release, deployment and observability sources without moving authority away from those systems.

#### Required invariants

- Existing Runtime `EvidenceRef` and `AttachEvidence` remain the authority for Runner execution evidence and are not replaced or weakened.
- Engineering stores a normalized projection/reference, never source-owned mutable lifecycle state.
- Evidence identity is immutable. Replaying the same identity with the same canonical content is idempotent; reusing an identity for different content is a conflict.
- Every envelope carries workspace scope, evidence kind, subject identity, source provenance, immutable source revision and/or digest, producer identity, observation/capture timestamps and a deterministic content checksum.
- Source locator/revision/digest and subject identity are explicit; opaque free-form evidence that cannot be traced back to a source is rejected.
- Secret values, credentials and tokens are forbidden from evidence metadata. Evidence bytes remain owned by Space or the external source; Engineering stores references and checksums only.
- Evidence may support later Fact derivation, but P3-S01 does not create Facts or governed Knowledge.
- Importing or recording evidence never mutates Workspace Task status, Runtime Run lifecycle, Change acceptance, DoneGate or Knowledge publication.

#### Initial evidence kinds

- `execution`
- `source_change`
- `build`
- `test`
- `release`
- `deployment`
- `observation`
- `incident`

The vocabulary is finite and versioned. Unknown kinds fail closed.

#### Deliverables

- Engineering domain `EvidenceEnvelope` with deterministic validation and checksum semantics.
- Typed subject reference and provenance/source reference contracts that do not import Runtime/GitHub/CI implementation packages.
- Repository port for immutable `Put/Get/List` evidence projection.
- SQLite migration and durable implementation with workspace isolation, no foreign keys/cascades and deterministic ordering.
- Public Engineering contract/application read/write surface guarded by existing workspace authorization; owner/admin may record normalized external evidence, members may read, outsiders are denied.
- Runtime integration is intentionally adapter-shaped: P3-S01 defines the mapping seam but does not change Runtime `EvidenceRef` or its event log.
- Tests for invalid kinds, missing provenance, weak/non-immutable source identity, checksum mismatch, idempotent replay, workspace isolation, cross-workspace subject protection and no lifecycle side effects.

#### Allowed paths

- `backend/internal/modules/engineering/**`
- `backend/internal/bootstrap/**` only for composition/integration tests if required
- `backend/docs/plans/iot-engineering-digital-thread/**`
- `docs/architecture/**` only if a boundary clarification is required

Explicitly forbidden for P3-S01:

- `server/**`
- mutation of Runtime execution semantics
- Workspace DoneGate semantics
- existing System Agent Release / Skill publication ownership
- autonomous Change acceptance or governed Knowledge publication
- build/release/deployment/observability network adapters; those start in later Phase 3 steps

#### Acceptance

- `cd backend && go test ./internal/modules/engineering/... -count=1`
- `cd backend && go vet ./internal/modules/engineering/...`
- repository Go 1.26.1 deterministic `make check`
- canonical `make test-race`
- frontend aggregate and `CI / required`
- compare against Phase 2 canonical contains no `server/**` changes
- CI evidence must be appended to `journal.md` before P3-S01 is accepted

### P3-S02 — Fact projection

Goal: derive rebuildable, provenance-bearing Facts from normalized Evidence without treating derived claims as source authority.

Staged deliverables include deterministic projector contracts, source-evidence sets, derivation version/checksum, rebuild tests and conflict/freshness semantics.

### P3-S03 — Build/test/release source adapters

Goal: ingest authoritative CI build/test and release evidence through provider-neutral read adapters and project the `Change -> PR -> Build -> Release` thread.

### P3-S04 — Deployment projection

Goal: connect immutable releases to deployment evidence and environments without transferring deployment lifecycle ownership into Engineering.

### P3-S05 — Runtime observation and incident feedback

Goal: normalize observations/incidents and trace them backward through Deployment -> Release -> Change -> Task/Requirement.

### P3-S06 — KnowledgeSuggestion and stale-knowledge detection

Goal: use accepted Changes, Facts and runtime feedback to produce reviewable KnowledgeSuggestions and stale architecture/runbook warnings. No autonomous governed Knowledge publication is permitted.

## 4. Phase 3 exit criteria

A canonical E2E scenario must prove forward traceability:

`Requirement -> Task -> Run -> Change -> PR -> Build -> Release -> Deployment`

and backward runtime traceability:

`Incident/Observation -> Deployment -> Release -> Change -> Task -> Requirement`.

Every transition must retain source provenance and immutable revisions/checksums. Facts must be rebuildable from evidence. KnowledgeSuggestions must remain proposals until governed review.

## 5. Governance and integration gate

Product implementation must reference Plan-ID `IOT-ENGINEERING-DIGITAL-THREAD-001`, version `4`, and the explicitly active step.

The repository currently reports no active remote ruleset for the canonical branch. This governance drift does not change product architecture, but it must be remediated or explicitly re-established before any Phase 3 product integration into canonical so independent review and required CI remain enforceable.

No direct canonical mutation is authorized by this plan.
