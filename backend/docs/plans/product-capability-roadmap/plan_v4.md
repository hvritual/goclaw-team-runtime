# Product capability roadmap — execution plan v4

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `4`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-17`
- Base commit: `ab2b49088b108f771045a090b473a8e235dfa09e`
- Active step: `PCR-S01B-4`
- Task revision: `r009`
- Supersedes for future execution: `plan_v3.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Close Release 0 — Authority and safety foundation. Commit the already accepted
S01B-3R repair, verify the complete S00/S01A/S01B release candidate, obtain an
independent read-only review of S01B, and record the resulting release state
without activating any Release 1 capability.

## 2. Scope

### Included

- the exact S01B-3R files accepted under v3/r008;
- one explicitly scoped candidate commit for that accepted repair and v4/r009
  activation records;
- deterministic Release 0 verification on the same committed candidate;
- independent read-only review of transaction atomicity, Workspace isolation,
  redaction, lease safety, migration policy, and candidate scope;
- roadmap story-map, task-register, journal, and current-plan closure records;
- one evidence-only commit after all acceptance gates pass.

### Excluded

- new product behavior, public API, schema, migration, capability flag, frontend,
  Desktop, Control Plane, generated output, deployment, push, merge, or pull
  request;
- changes below `server/**` or cleanup of unrelated dirty paths;
- repair of any newly discovered defect without a new plan version and frozen
  task revision;
- Release 1 or later roadmap activation.

## 3. Invariants

1. `server/**` remains permanently read-only and absent from every candidate.
2. Existing UI, local-runtime, code-to-product, and other unrelated dirty paths
   remain unmodified and unstaged.
3. S01B revision, idempotency, audit, and outbox state commits atomically with
   the governed domain mutation; rollback commits none of them.
4. Outbox delivery occurs only after commit, remains Workspace-scoped and
   replayable, and uses stable event identity.
5. Payloads, errors, audit projections, and operator diagnostics expose no
   credential or governed payload content beyond their frozen contracts.
6. Lease claim, acknowledgement, retry, dead-letter, and replay transitions are
   token guarded and bounded; empty polling remains read-only.
7. The accepted S01B-3R repair retains caller cancellation, finite external
   SQLite contention handling, and attachment atomicity.
8. Every Release 1+ feature flag remains false.
9. The primary assignee cannot self-certify the independent-review gate.

## 4. Dependencies

- accepted Release 0 records through S01B-3R task `r008`;
- committed S01B candidate through `ab2b490` and the accepted, uncommitted r008
  repair in the current worktree;
- unchanged frozen policy hashes for `CLAUDE.md`, `backend/AGENTS.md`, and plan
  snapshots v1-v3;
- the Human Customer statement on 2026-08-17: `批准后续动作，按目标持续推进完成
  Release 0 — Authority and safety foundation`;
- an independent read-only reviewer distinct from the implementing assignee.

No network call, credential, external service, package installation, database
write outside tests, or runtime deployment is authorized.

## 5. Ordered steps

### PCR-S01B-4.1 — Freeze and commit the candidate

Freeze task `PCR-001-S01B4-R9`, revalidate base, policy hashes, exact paths, and
dirty exclusions, then explicitly stage only the accepted r008 files and v4/r009
activation records. Audit the index before committing with required trailers.

### PCR-S01B-4.2 — Integrated deterministic evidence

Run the v3 repair gates plus focused S01B contract, repository, HTTP, runtime,
full Backend, race, module, formatting, policy, generated-output, diff, and
`server/**` boundary checks on the same committed candidate.

### PCR-S01B-4.3 — Independent review

After deterministic checks pass, provide the reviewer the frozen S01B contract,
candidate range, acceptance criteria, and exact review dimensions. The reviewer
must remain read-only and report blocking findings with file and line evidence.
Any blocking finding stops closure and requires a new plan version before code
repair.

### PCR-S01B-4.4 — Release 0 closure

When deterministic evidence and independent review pass, synchronize
`story-map.md`, `task-register.md`, `journal.md`, and `plan.md`; record the
Customer's instruction to complete Release 0 as the closure authority; verify
the evidence-only diff; and commit the closure records with required trailers.

## 6. Exact writable paths

Accepted r008 candidate and its checks:

- `backend/Makefile`
- `backend/cmd/backend-check/main.go`
- `backend/cmd/backend-check/main_test.go`
- `backend/ci/test-race.ps1`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository.go`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository_test.go`

Roadmap authority and evidence:

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v3.md`
- `backend/docs/plans/product-capability-roadmap/plan_v4.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other path may be staged or modified. Discovery of a required path outside
this list stops execution and requires a new plan version or task revision.

## 7. Acceptance criteria

1. S00 authority inventory, task/base-drift gate, and dirty-path preservation
   remain accepted and unchanged.
2. S01A capability-specific member/agent denials and missing-provider
   fail-closed behavior remain accepted and pass their focused tests.
3. Concurrent revision writes produce one winner and one conflict.
4. Idempotent replay returns the original result without a second mutation.
5. Rollback commits no audit row or outbox event.
6. A committed mutation is delivered only after commit, retains stable identity,
   and remains replayable through bounded failure handling and restart.
7. Workspace isolation, actor authorization, payload/audit redaction, lease
   safety, migration policy, diagnostics, and readiness behavior pass focused
   and full-suite verification.
8. Every v3 repair gate passes on the same candidate with no loader waiver,
   weakened assertion, skipped test, or permanent host mutation.
9. The candidate contains only exact allowed paths; `server/**` and unrelated
   dirty paths remain excluded.
10. An independent reviewer reports no blocking finding across the frozen review
    dimensions.
11. Closure records mark Release 0 complete and keep Release 1 inactive.

## 8. Deterministic verification

From `backend/` unless stated otherwise:

```text
go test ./internal/modules/workspace/internal/application -count=1
go test ./internal/modules/workspace/internal/infrastructure/sqlite -count=1
go test ./internal/modules/workspace/internal/interfaces/http -count=1
go test ./internal/modules/workspace -count=1
go test ./internal/bootstrap -count=1
go test ./internal/modules/space/internal/infrastructure/sqlite -run 'AttachmentRepository.*(BusyWriteAcquisition|Cancellation|Serial)' -count=10
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=10
go test ./cmd/backend-check -count=1
go test ./... -count=1
go vet ./...
go mod verify
make fmt-check
make check
make test-race RACE_PACKAGES="./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace/internal/interfaces/http ./internal/modules/workspace ./internal/bootstrap"
git diff --check
git diff --cached --check
git diff --name-only -- server
git diff --cached --name-only -- server
git status --porcelain -- server
```

The index and each commit are additionally compared with the exact writable
path allowlist before the commit is created or accepted.

## 9. Independent review contract

The reviewer examines the committed S01B implementation range beginning with
`3876791` plus the accepted r008 repair commit and reports:

- `PASS` or `BLOCK` for transaction atomicity and rollback semantics;
- `PASS` or `BLOCK` for Workspace/actor isolation and authorization;
- `PASS` or `BLOCK` for idempotency, revision conflicts, and replay identity;
- `PASS` or `BLOCK` for payload, audit, error, and diagnostics redaction;
- `PASS` or `BLOCK` for lease, retry, dead-letter, restart, and readiness safety;
- `PASS` or `BLOCK` for migration and `server/**` policy compliance;
- `PASS` or `BLOCK` for exact candidate scope and test coverage.

A concern without a blocking contract violation is recorded as acceptance debt;
any `BLOCK` prevents Release 0 closure.

## 10. Risks and controls

| Risk | Control |
| --- | --- |
| Accepted repair is mixed with unrelated changes | explicit path staging and cached-diff audit |
| Historical waiver is mistaken for current evidence | rerun all S01B and r008 gates on one committed candidate |
| Assignee self-approves the release | independent read-only reviewer provides a separate verdict |
| Review discovers a real defect | stop; record evidence; create v5/new task before repair |
| Documentation claims a broader release | mark Release 0 only and leave Release 1 inactive |
| Windows tools mutate the host | process-local compiler selection and no installation |

## 11. Rollback

- Before the first commit, unstage only the explicitly staged r009 paths; never
  discard the accepted worktree or unrelated user changes.
- After the first commit, use a new revert commit for the exact candidate if the
  Customer directs rollback; never reset or rewrite shared history.
- Documentation closure can be reverted independently from product code.
- Never delete attachment data, run a down migration, modify permanent host
  configuration, or touch `server/**` as rollback.

## 12. Approval record

The Human Customer explicitly approved all displayed v4/r009 follow-up actions,
including the independent read-only review, and directed continuous progress to
complete Release 0 on 2026-08-17. This approval authorizes this plan only; it
does not authorize push, merge, deployment, Release 1 activation, or repair of a
newly discovered defect outside a new approved plan.
