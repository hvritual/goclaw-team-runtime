# Product capability roadmap v42 — Release 4 S08A selective integration

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `42`
- Task-ID: `PCR-001-S08A-R047`
- Task-Revision: `r047`
- Work-Item: `PCR-S08A`
- Title: `Integrate the bounded Issue-similarity warning candidate into the canonical baseline`
- Exact base: `bbc1a452c84945cde3fe633a43610a8f1db3ae77`
- Integration branch: `codex/release4-s08a-integrate`
- Source evidence only: `02359a01`, `496013ea`, and `76265a27` from
  `codex/release4-s08a-r070`
- Frozen policy bundle: `CLAUDE.md` SHA-256
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646` /
  `backend/AGENTS.md` SHA-256
  `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`
- Status: `active — R47.1 plan frozen; R47.2-R47.6 pending`
- Authority: The Human Customer explicitly confirmed this selective integration
  into `codex/multica-six-domain-baseline` on 2026-08-27.

## Goal and product contract

Implement only PCR-S08A. Before an authenticated member creates an Issue, and
after a material title or description edit in the create-Issue dialog, the
client may request a ranked, workspace-local list of potential similar Issues.
The member can inspect the candidates and still create the Issue normally.

The candidate is a warning, not a duplicate relation or merge decision. It
must remain scoped to the current workspace, preserve the existing Issue create
contract, return no more than five ranked candidates, and tolerate detector
unavailability by retaining the ordinary create flow with a localized warning
state. S08B duplicate-decision persistence and S09 reminder work stay
inactive.

## Dependencies, invariants, and rollback

Dependencies are the Release 3 closure at the exact base, the three named
source commits remaining locally resolvable, a clean isolated worktree, and
the existing repository-owned Node, Go, pnpm, SQLite, and Windows-race
toolchains. No network service, deployment credential, external database, or
legacy tree is a dependency.

The current workspace identity remains route-driven and server authorization
remains authoritative. Candidate records never cross workspace boundaries; the
detector never creates or changes an Issue, duplicate relation, migration, or
generated contract; a detector error is not allowed to block ordinary Issue
creation; and all client cache keys include workspace identity. API validation
remains typed and every new state has an explicit loading, empty, and error
projection.

If a selective patch conflicts, a test/regression/review gate fails, the scope
audit detects a forbidden path, or the canonical branch has advanced from the
exact base, stop R47 before integration. Discard only the uncommitted candidate
changes in the isolated worktree using a separately verified recovery action;
do not reset, overwrite, or modify the canonical worktree. No data migration
or external side effect exists to roll back.

## Source-selection and write boundary

This is **not** a branch merge and not an adoption of the source branch's
roadmap history. R47 may transplant only the product and test bytes reached by
the three named source commits, plus the five canonical roadmap records below.
All source `backend/docs/**` history and every source path outside this list is
excluded.

Authorized product and test paths:

```text
backend/internal/bootstrap/issue_similarity_runtime_test.go
backend/internal/bootstrap/runtime.go
backend/internal/bootstrap/runtime_test.go
backend/internal/modules/workspace/contract/issue_similarity.go
backend/internal/modules/workspace/internal/application/issue_similarity.go
backend/internal/modules/workspace/internal/application/issue_similarity_test.go
backend/internal/modules/workspace/internal/infrastructure/sqlite/issue_similarity_repository.go
backend/internal/modules/workspace/internal/infrastructure/sqlite/issue_similarity_repository_test.go
backend/internal/modules/workspace/internal/interfaces/http/issue_similarity.go
backend/internal/modules/workspace/internal/interfaces/http/issue_similarity_test.go
backend/internal/modules/workspace/issue_similarity_composition_test.go
backend/internal/modules/workspace/issue_similarity_extension.go
backend/internal/modules/workspace/sqlite_workspace_chain.go
packages/core/api/client.ts
packages/core/api/issue-similarity-schema.test.ts
packages/core/api/schemas.ts
packages/core/issues/similarity.test.ts
packages/core/issues/similarity.ts
packages/core/package.json
packages/core/types/api.ts
packages/views/issues/components/issue-detail.test.tsx
packages/views/issues/components/issue-detail.tsx
packages/views/issues/components/issue-similarity-warning.test.tsx
packages/views/issues/components/issue-similarity-warning.tsx
packages/views/modals/create-issue.test.tsx
packages/views/modals/create-issue.tsx
packages/views/locales/en/issues.json
packages/views/locales/en/modals.json
packages/views/locales/ja/issues.json
packages/views/locales/ja/modals.json
packages/views/locales/ko/issues.json
packages/views/locales/ko/modals.json
packages/views/locales/zh-Hans/issues.json
packages/views/locales/zh-Hans/modals.json
```

The only authorized governance paths are this immutable snapshot,
`plan.md`, `story-map.md`, `task-register.md`, and `journal.md`. `server/**`,
generated protobufs, migrations, source-branch governance documents, S08B,
S09+, deployment, push, and the base worktree's pre-existing dirty paths are
outside the write boundary. No database foreign key or cascade is introduced.

## Execution and acceptance plan

### R47.1 — Freeze and independently review the integration plan

Create this immutable snapshot and point the canonical roadmap to it. An
independent plan/spec reviewer must confirm the scope, source selection,
exclusions, and test contract before production bytes are changed.

Acceptance: `git diff --check` passes for the roadmap records; all named source
commits and the exact base resolve; no source roadmap document is copied.

### R47.2 — Selectively transplant the S08A feature and tests

Apply only the authorized product/test paths from the named source commits in
chronological order. Preserve the public API shape, HTTP composition, SQLite
repository boundary, Core API schema, shared warning component, four locales,
and Issue create/detail integration. Partition the client similarity cache by
workspace and preserve the Core package export required by the modal.

The Customer approved a narrow TDD transplant exception for this step: the
already-authored source tests may be moved with their implementation rather
than recreated as new RED tests. This exception applies only to the paths in
this plan; all behavior remains test-verified in R47.3-R47.4.

Acceptance: the resulting diff contains only R47-authorized paths, has no
`server/**` or generated path, and the feature is warning-only with no duplicate
relation persistence.

### R47.3 — Run direct S08A verification

Run the source-owned deterministic tests for the backend composition,
application/repository/HTTP behavior, Core schema and similarity calculation,
and shared Views modal/detail warning behavior. Verify workspace isolation,
ranking/limit, cache partitioning, unavailable handling, and locale keys.

Acceptance: all targeted tests pass without test skips or source-tree writes.

### R47.4 — Run canonical regression and static gates

Run `pnpm typecheck`, `pnpm test`, `cd backend && go test ./...`, and
`cd backend && make check`. Run the repository-owned race command if its
documented Windows toolchain is available; otherwise record the precise
environmental non-pass and do not substitute an unapproved compiler. Inspect
the final path list, whitespace errors, generated-path count, and `server/**`
count.

Acceptance: required deterministic checks pass, or any non-pass is classified
with reproducible evidence and escalated before integration. No original dirty
path may overlap the candidate.

### R47.5 — Independent dual review

Use separate fresh reviewers. The specification reviewer checks the plan and
implemented behavior against PCR-S08A, boundary, warning-only, isolation and
failure-mode requirements. Only after SPEC PASS may the code-quality reviewer
check correctness, security, concurrency, test adequacy, API compatibility,
and canonical boundaries.

Acceptance: both reviews are PASS with no unresolved blocker on the exact
candidate.

### R47.6 — Integrate the reviewed candidate

Commit the reviewed candidate with the required roadmap trailers, fast-forward
only `codex/multica-six-domain-baseline` if it still points at the exact base,
and recheck that its pre-existing unrelated changes remain untouched. Do not
push or deploy. Update this plan's mutable records only to state the actual
result; this immutable file is never amended.

Acceptance: canonical HEAD contains the exact reviewed candidate, retains the
unrelated dirty entries without overlap, and Release 4 records PCR-S08A's real
post-review status. If any gate blocks, leave the canonical branch unchanged.
