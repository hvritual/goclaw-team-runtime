# Product capability roadmap — execution plan v3

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `3`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-17`
- Base commit: `ab2b49088b108f771045a090b473a8e235dfa09e`
- Active step: `PCR-S01B-3R`
- Supersedes for future execution: `plan_v2.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Remove the three deterministic verification blockers retained by
`PCR-S01B-3`: the Windows race-loader failure, the Windows-incompatible
`make check` wrapper, and the two existing SQLite attachment concurrency
failures. Preserve S01B product behavior and keep every roadmap capability
flag disabled.

## 2. Scope

### Included

- cross-platform Backend formatting, policy, and generated-output checks;
- a process-local Windows race-toolchain selector with no permanent host
  environment mutation;
- context-aware serialization and bounded SQLite lock acquisition for Space
  attachment writes;
- focused concurrency, cancellation, full-suite, race, and boundary evidence;
- roadmap task and evidence updates.

### Excluded

- permanent PATH changes, package installation, or administrator-level host
  configuration;
- product capabilities, public APIs, migrations, frontend, Desktop, Control
  Plane, generated output, or existing governance behavior;
- weakening, skipping, quarantining, or relabelling either attachment test;
- changes below `server/**` or cleanup of unrelated dirty paths;
- Git commit, push, merge, or pull-request creation.

## 3. Invariants

1. `server/**` remains permanently read-only.
2. Windows and Unix `make check` enforce the same formatting, policy, generated,
   vet, and test semantics.
3. Race is PASS only when the requested race tests execute and pass; loader or
   compiler failures remain failures.
4. Windows compiler selection changes only the child verification process and
   never persists user or system environment changes.
5. Attachment asset, version, Workspace binding, object cleanup, and post-commit
   event semantics remain atomic and Workspace-scoped.
6. Caller cancellation remains authoritative while queued for an attachment
   write; SQLite external contention retains a finite acquisition budget.
7. No test count, assertion, timeout, or response expectation is weakened to
   obtain green evidence.
8. Every roadmap product feature flag remains false.

## 4. Dependencies

- approved `PRODUCT-CAPABILITY-ROADMAP-001 v2` and committed S01B-3 candidate
  `f0d86d9`;
- evidence commit `ab2b490` recording the three blockers;
- Go 1.26.1 automatic toolchain, PowerShell 7, and an existing compatible
  MinGW-w64 compiler available on this Windows host;
- current Canonical SQLite and Space attachment repositories and retained tests.

No network call, credential, external service, package installation, schema
change, or Control Plane dependency is authorized.

## 5. Ordered steps

### PCR-S01B-3R.1 — Freeze and reproduce

Freeze task revision `r008`, preserve the current dirty-worktree exclusions,
and run each existing attachment failure once before production changes.

### PCR-S01B-3R.2 — Cross-platform checks

Add a tested Go Backend-check command for formatting, policy-boundary, and
generated-output cleanliness. Route the corresponding Make targets through it
without changing their acceptance meaning.

### PCR-S01B-3R.3 — Windows race execution

Add a PowerShell race runner that selects a compatible compiler from the
already-installed PATH candidates, adjusts only its child process, and invokes
the supplied package set exactly once. Route the Windows `test-race` target to
it while retaining direct `go test -race` on non-Windows systems.

### PCR-S01B-3R.4 — Attachment contention repair

Use the existing failing tests plus a focused cancellation/serialization test
to drive the smallest Space repository change. Serialize attachment-owned write
transactions before consuming SQLite connections, preserve caller cancellation,
and retain finite retry handling for external SQLite writers.

### PCR-S01B-3R.5 — Integrated evidence

Run the frozen focused, repeated concurrency, full Backend, race, Make, policy,
formatting, module, and scope gates on the same worktree. Record exact outcomes;
do not infer independent review or Customer Acceptance.

## 6. Exact writable paths

- `backend/Makefile`
- `backend/cmd/backend-check/main.go` (new)
- `backend/cmd/backend-check/main_test.go` (new)
- `backend/ci/test-race.ps1` (new)
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository.go`
- `backend/internal/modules/space/internal/infrastructure/sqlite/attachment_repository_test.go`
- `backend/internal/bootstrap/issue_attachment_runtime_test.go`
- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v3.md` (new)
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

Discovery of any other required implementation path stops execution and
requires a new plan version or task revision.

## 7. Acceptance criteria

1. The two indexed attachment tests pass without relaxed assertions or timeouts,
   including ten consecutive focused counts.
2. Concurrent uploads retain all twelve responses, references, and files.
3. Queued attachment writes honor caller cancellation and do not leak a write
   slot, transaction, connection, row, or object.
4. `make fmt-check` and `make check` execute successfully on Windows and retain
   equivalent Unix behavior.
5. The frozen five-package race command executes with a compatible existing
   compiler and passes; `0xc0000139` is not accepted as PASS.
6. Full Backend tests, vet, module verification, formatting, generated-output,
   policy, diff, and `server/**` boundary checks pass.
7. No public contract, migration, feature flag, frontend, generated, Control
   Plane, unrelated dirty, or `server/**` path changes.

## 8. Deterministic verification

From `backend/` unless stated otherwise:

```text
go test ./internal/modules/space/internal/infrastructure/sqlite -run '^TestAttachmentRepositoryRetriesBusyWriteAcquisition$' -count=1
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=1
go test ./cmd/backend-check
go test ./internal/modules/space/internal/infrastructure/sqlite -run 'AttachmentRepository.*(BusyWriteAcquisition|Cancellation|Serial)' -count=10
go test ./internal/bootstrap -run '^TestSQLiteRuntimeConcurrentAttachmentUploadsLoseNoReferencesOrFiles$' -count=10
go test ./... -count=1
go vet ./...
go mod verify
make fmt-check
make check
make test-race RACE_PACKAGES="./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace/internal/interfaces/http ./internal/modules/workspace ./internal/bootstrap"
git diff --check
git diff --name-only -- server
git status --porcelain -- server
```

## 9. Risks and controls

| Risk | Control |
| --- | --- |
| Compiler selection masks a missing dependency | enumerate installed candidates, require a compatible version, print the selected path, run tests once |
| Windows repair breaks Unix CI | keep non-Windows direct Go invocation and use a Go checker with platform-neutral tests |
| Attachment mutex ignores cancellation | use a context-aware slot and a cancellation regression test |
| Serialization hides external DB contention | retain bounded `BEGIN IMMEDIATE` contention handling after slot acquisition |
| Transaction or object cleanup regresses | existing rollback, event, deletion, restart, and full-suite tests remain mandatory |
| Slow tests are made green by timeout inflation | no existing assertion or timeout weakening is authorized |

## 10. Rollback

- Revert the Backend-check command, Make routing, and race runner together to
  restore the prior verification entry points.
- Revert the attachment coordination change as one unit; it adds no schema or
  persisted state.
- Never delete attachment data or run a down migration as rollback.
- Rollback does not modify permanent host configuration, unrelated dirty paths,
  or `server/**`.

## 11. Approval record

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v3` on
2026-08-17. That approval authorizes this plan and `PCR-S01B-3R` only. S01B-4,
independent review, commit, merge, and broader product work remain inactive.
