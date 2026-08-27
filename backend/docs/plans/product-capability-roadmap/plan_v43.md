# Product capability roadmap v43 — Release 4 S08A governed selective integration

- Plan-ID: PRODUCT-CAPABILITY-ROADMAP-001
- Plan-Version: 43
- Task-ID: PCR-001-S08A-R048
- Task-Revision: r048
- Work-Item: PCR-S08A
- Title: Integrate the bounded Issue-similarity warning candidate into the canonical baseline
- Exact base: bbc1a452c84945cde3fe633a43610a8f1db3ae77
- Integration branch: codex/release4-s08a-integrate
- Predecessor: immutable plan_v42.md / r047 is review-blocked-before-product-code
- Source evidence only: 02359a01, 496013ea, and 76265a27 from codex/release4-s08a-r070
- Frozen policy bundle: CLAUDE.md SHA-256 6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646 / backend/AGENTS.md SHA-256 fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209
- Status: active — R48.1 immutable successor review pending; R48.2-R48.7 pending
- Authority: The Human Customer explicitly confirmed selective integration into codex/multica-six-domain-baseline on 2026-08-27. This successor repairs only the r047 plan-governance block; it grants no broader product, deployment, or source-branch authority.

## Goal and success contract

Implement only PCR-S08A. Before an authenticated member creates an Issue, and after a material title or description edit in the create-Issue dialog, the client may request a ranked, workspace-local list of potential similar Issues. The member can inspect candidates and still create the Issue normally.

The feature is a warning, never a duplicate relation, issue merge, or create-flow denial. The existing Issue-create contract is preserved; at most five candidates are returned; all candidates stay inside the current workspace; and detector failure leaves normal creation available with a localized warning state. S08B duplicate-decision persistence and S09 reminder work remain inactive.

## Dependencies and revalidation

R48 depends on the Release 3 closure at the exact base; the three named source commits resolving locally; a clean isolated worktree on the named integration branch; the current repository-owned Node, Go, pnpm, SQLite, and Windows-race toolchains; and a fresh independent R48.1 SPEC PASS. It does not depend on any external service, deployment credential, external database, or legacy tree.

At R48.1, R48.2 start, R48.5 candidate freeze, and R48.7 acceptance, revalidate the exact base, plan v43 SHA-256, policy-bundle hashes, named source commits, and the source path-set SHA-256 recorded in the mutable task register. Stop if any identity differs.

## Risks and mandatory stop responses

- Source/base divergence or a selective-patch conflict: stop before product commit; do not broaden the selection.
- Transplanted test, static, race, regression, or review failure: stop, record evidence, and leave canonical unchanged.
- A candidate forbidden path or original-dirty-path overlap: stop before review/integration; do not overwrite user work.
- Canonical branch advanced from the exact base: stop and create a new successor plan rather than implicitly rebasing.
- Detector error blocks creation or exposes data: stop; any repair needs its own approved successor plan.

## Invariants

Current workspace identity remains route-driven and server authorization remains authoritative. Candidate records never cross workspace boundaries. The detector never creates or changes an Issue, duplicate relation, database migration, or generated contract. A detector error cannot block ordinary Issue creation. All client cache keys include workspace identity. API validation remains typed and each new client state renders explicit loading, empty, denied, and error projections.

## Scope and non-goals

This is not a branch merge and not adoption of the source branch's roadmap history. R48 may transplant only the 34 product/test paths below from the three named commits, plus plan_v43.md, plan.md, story-map.md, task-register.md, and journal.md. All source backend/docs history and every source path outside this list are excluded.

~~~text
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
~~~

server/**, generated protobufs, migrations, source-branch governance documents, S08B, S09+, deployment, push, and the canonical worktree's pre-existing dirty paths are outside the write boundary. No database foreign key or cascade is introduced.

## TDD and traceability

The Customer approved a narrow TDD transplant exception: only the selected, already-authored source tests may travel with their selected implementation; they are not rewritten as new RED tests. The exception ends at this manifest. R48.3-R48.5 must still directly verify all resulting behavior and regressions.

Every R48 commit must contain the continuous Git trailers Task-ID: PCR-001-S08A-R048, Project-ID: PRODUCT-CAPABILITY-ROADMAP, Task-Revision: r048, Work-Item: PCR-S08A, Plan-ID: PRODUCT-CAPABILITY-ROADMAP-001, Plan-Version: 43, a specific Plan-Step, Issue: PCR-S08A, and the frozen Policy-Bundle value above. No unrelated fix may be mixed into the candidate.

## Ordered execution and deterministic acceptance

### R48.1 — Freeze and independently review this successor plan

Record v43/policy/source-manifest identities in the task register and obtain a fresh independent SPEC PASS. Acceptance: all identities resolve, git diff --check passes for roadmap files, the scope is exact, and no product byte is changed before PASS.

### R48.2 — Selectively transplant S08A implementation and tests

Apply only the manifest paths from the named source commits in chronological order. Preserve the HTTP composition, SQLite repository, typed Core API schema, shared warning component, four locales, Issue create/detail integration, workspace-partitioned cache, and Core export needed by the modal. Acceptance: the diff is manifest-only, warning-only, has zero server/generated paths, and creates no duplicate relation.

### R48.3 — Direct feature verification

Run direct deterministic tests for bootstrap composition, Workspace application/repository/HTTP behavior, Core schemas/similarity, and shared Views modal/detail behavior. Prove isolation, ranking/limit, cache partitioning, unavailable handling, and locale keys. Acceptance: all targeted tests pass with no skips or source-tree writes.

### R48.4 — Canonical regression and static verification

Run pnpm typecheck, pnpm test, cd backend && go test ./..., and cd backend && make check. Run the repository-owned race command when its documented Windows toolchain is available; otherwise record the exact environmental non-pass without substituting an unapproved compiler. Audit final paths, whitespace, forbidden path counts, and original-dirty overlap.

### R48.5 — Freeze exact candidate and revalidate identity

Capture candidate/base/tree identity, revalidate all R48 dependency hashes, and verify the worktree is clean apart from the manifest. Acceptance: exact candidate scope and deterministic gate evidence are indexed in the journal.

### R48.6 — Independent dual review

Use separate fresh reviewers. The specification reviewer validates PCR-S08A, boundaries, warning-only behavior, isolation, and failure mode. Only after SPEC PASS may the code-quality reviewer validate correctness, security, concurrency, test adequacy, API compatibility, and canonical boundaries. Acceptance: both are PASS with no unresolved blocker on the exact candidate.

### R48.7 — Fast-forward reviewed candidate into canonical

Only if codex/multica-six-domain-baseline still points to the exact base, fast-forward it to the reviewed candidate and preserve its unrelated dirty entries. Do not push or deploy. Acceptance: canonical HEAD has the exact reviewed commit; status confirms untouched unrelated dirt; mutable records state the real result.

## Rollback and recovery

If any stop condition occurs, leave the canonical branch untouched and record the failed gate in the journal. Since no migration, external service, or deploy is authorized, there is no data or runtime rollback. Before any cleanup, resolve and verify the exact isolated-worktree paths; do not use reset, checkout, recursive deletion, or overwrite against the canonical worktree. A new successor plan is required for any material scope, contract, dependency, risk, or gate change.

