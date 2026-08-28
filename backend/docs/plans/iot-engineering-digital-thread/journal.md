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

## 2026-08-28 — P1-S03 durable local adapter

- Added adapter-local SQLite schema and migrations for EngineeringEntity, SourceBinding, ThreadEdge, Change, Change affected-entity/artifact children, ContextPack, ContextPack target entities, and ContextPack references.
- The schema intentionally contains no database foreign keys or cascade actions. Workspace identity is part of every primary key or lookup boundary.
- Added an embedded migration runner with an adapter-local `engineering_schema_migrations` catalog, WAL mode, a 5-second busy timeout, explicit rollback support, and idempotent re-application.
- Added a durable SQLite Store implementing the Engineering Thread repository ports for entities, source bindings, provenance-bearing typed edges, changes, and frozen context packs.
- Change child collections are replaced inside the same application transaction as the parent mutation; dependent cleanup does not rely on database cascades.
- ContextPack persistence is immutable: an identical `(workspace,id,checksum,created_at)` write is idempotent, while the same identity with a different representation returns `domain.ErrConflict`.
- Added repository contract tests covering workspace isolation, entity/source/edge/change/context round trips, filtered change lookup, not-found behavior, immutable ContextPack conflicts, reopen recovery, rollback/reapply, absence of foreign keys, and concurrent entity writers.
- Compare from the P1-S02 head `0a60e5348c11244a3cf17bc36b9b14164718a876` to P1-S03 implementation head `b8cd8f98b97fa58c2da2f9277c8de2662cb7867c` contains only `backend/internal/modules/engineering/**`; no `server/**` path was changed.
- CI run `33161301114` reached the repository governance gate. `frontend-test` passed, but `governance-policy` failed at the pre-existing canonical-backend legacy-import guard and therefore skipped `canonical-backend`; frontend build also failed.
- The same two failures already exist on CI run `33159973803` for pre-P1-S03 head `0a60e5348c11244a3cf17bc36b9b14164718a876`. This proves those repository-level blockers predate P1-S03 rather than being introduced by the durable-adapter diff.
- Because `canonical-backend` was skipped, repository-authoritative Go 1.26.1 compilation/test/vet evidence is not available for P1-S03. The implementation is complete, but `P1-S03` remains **acceptance-blocked** until the pre-existing governance failure is repaired and the canonical backend checks run successfully.
- Per plan dependency rules, `P1-S04` must not start while `P1-S03` remains acceptance-blocked.

## Verification log

- `P1-S01`: documentation-only; no product-code verification claimed.
- `P1-S02`: isolated complete package `gofmt`, `go test`, and `go vet` passed before commit; repository CI/full backend checks pending.
- `P1-S03`: durable adapter implementation and contract-test sources are committed; diff is confined to Engineering Thread paths and contains no `server/**` changes. CI run `33161301114` cannot execute canonical backend checks because the baseline-wide policy guard already fails on pre-P1-S03 head `0a60e534` (run `33159973803`). Acceptance is blocked, not claimed.
