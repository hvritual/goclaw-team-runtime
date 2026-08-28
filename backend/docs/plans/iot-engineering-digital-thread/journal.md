# IoT Engineering Digital Thread — Journal

Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`

## 2026-08-28 — Activation

- User authorized writing the complete implementation plan to `codex/multica-six-domain-baseline` and starting Phase 1.
- Canonical base commit frozen at `d3bbafb071dc493bd17d5c0387297bbf38da9ecb`.
- Repository governance reread: root `AGENTS.md`, root `CLAUDE.md`, and `backend/AGENTS.md`.
- Hard boundary retained: `server/**` is permanently read-only; implementation may modify only `backend/**` plus non-backend documentation/frontend paths explicitly named by a plan step.
- Existing Multica boundaries retained: Workspace owns team intent; Execution/Runtime owns attempts only; System continues to mean Agent Release / Skill publication and is not reused for IoT engineering systems.
- Phase 1 begins with architecture contract (`P1-S01`) followed by the canonical engineering-thread domain foundation (`P1-S02`).

## 2026-08-28 — P1-S02 foundation slice

- Added the initial Engineering Thread domain vocabulary under `backend/internal/modules/engineering/internal/domain`.
- Added canonical IoT engineering entity types, including the explicit `engineering_system` type so the existing Agent-release `System` context is not overloaded.
- Added typed thread node kinds and a finite relationship vocabulary; generic authoritative `related_to` edges are intentionally unsupported.
- Added provenance-bearing `SourceBinding` and `ThreadEdge` values.
- Added monotonic authority classes: `proposed -> inferred -> observed -> authoritative`; downgrade through the ordinary promotion API is rejected.
- Added table-driven tests for entity validation, provenance, self-edge rejection, generic-relation rejection, and authority promotion.
- Pre-commit isolated verification used the exact candidate Go files: `gofmt`, `go test ./... -count=1`, and `go vet ./...` passed under the available Go 1.23.2 toolchain. This is not a claim that the repository-wide Go 1.26.1 checks have run.

## 2026-08-28 — P1-S02 continuation

- Added `Change` as a first-class domain object, separate from both Workspace work intent and Runtime execution attempts.
- Change starts as `proposed`; acceptance is an explicit transition and records an acceptance timestamp. Accepted changes may later be superseded, while rejected/accepted states cannot be silently rewritten through invalid transitions.
- Change work linkage uses a typed work-item reference and rejects a Runtime `run` node as a work-item identity; Run remains a separate optional execution reference.
- Added immutable `ContextPack` manifests with pinned work-item revision, target engineering entities, revision/checksum-bearing context references, selection-policy version, and deterministic content checksum.
- ContextPack checksum is independent of pack ID, creation time, input target ordering, and context-reference ordering so equivalent frozen context produces the same content identity.
- Added repository port interfaces for EngineeringEntity, SourceBinding, ThreadEdge, Change, and ContextPack. No persistence or transport dependency enters the domain package.
- Added tests for Task/Todo vs Run vs Change separation, invalid work-item links, deterministic ContextPack checksum, checksum mismatch on rehydrate, and mandatory context revision/checksum metadata.
- Re-ran isolated `gofmt`, `go test ./... -count=1`, and `go vet ./...` against the complete P1-S02 candidate package; all passed under Go 1.23.2.
- `P1-S02` domain deliverables are now implemented. Repository-wide Go 1.26.1 checks and CI remain required before the step can be considered accepted and before durable adapter work (`P1-S03`) is treated as dependency-complete.

## Verification log

- `P1-S01`: documentation-only; no product-code verification claimed.
- `P1-S02`: isolated complete package `gofmt`, `go test`, and `go vet` passed before commit; repository CI/full backend checks pending.
