# Product capability roadmap v37 — S07D migration-count integration assertion

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `37`
- Task-Revision: `r042`
- Work-Item: `PCR-S07D`
- Exact base: `24d3043b322ae27add8023507330f3b9d55b1d95`
- Predecessor plan: `plan_v36.md`
- Predecessor plan hash: `e652d38ebbc1d697d822554fe689aa842183ff6721f6e0e5335c50d05abb3c5d`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and repeated confirmed
  execution

## Successor trigger and immutable inheritance

v36/r041 corrected the only missing production dependency for both installed
Project deletion repositories. Assertion-first R41.2 then ran the complete
Workspace package graph against additive migration
`000020_project_retrospectives.up.sql`. Every Workspace subpackage passed except
the root package, where exactly two existing migration-count assertions still
expected 19 instead of the installed and catalogued count 20:

- `backend/internal/modules/workspace/sqlite_persistence_test.go`, already in
  the v35-v36 boundary; and
- `backend/internal/modules/workspace/sqlite_workspace_services_test.go`,
  inadvertently absent from that boundary.

An exact repository-wide search found no other count-19 assertion or required
path. The failed integration assertion cannot truthfully pass while retaining
the frozen additive migration, and changing production behavior to hide the
twentieth migration would violate the S07D persistence contract.

v36 remains immutable. r041 is `superseded-before-product-commit`: its only
commit is governance successor activation `24d3043b`; all S07D product work
remains uncommitted. v37/r042 incorporates every v35 and v36 product contract,
acceptance scenario, ordered output, deterministic/installed gate, exclusion,
and stop condition unchanged. It changes only the one test-path boundary needed
to update the installed migration-count integration assertion from 19 to 20.

## Exact writable boundary

The exact writable boundary is every path authorized by immutable v36, plus
exactly:

- `backend/internal/modules/workspace/sqlite_workspace_services_test.go`.

No production behavior is added by this successor. Every v1-v36 plan,
protobuf/generated path, unrelated Auth/Bootstrap/Issue/Task/Knowledge
behavior, app-specific route, original dirty path, legacy backend tree, and
every `server/**` path remains read-only.

## Ordered execution

1. R42.1 — Freeze this successor from exact r041 governance activation, rename
   the isolated branch to `codex/release3-s07d-r042`, register r042, and commit
   only the four governance pointer/register/journal/story-map paths plus this
   immutable plan with one continuous nine-field trailer block.
2. R42.2 — Complete inherited R41.2 and update only the two exact migration
   count assertions from 19 to 20; rerun the complete Workspace graph before
   committing the Canonical Retrospective authority.
3. R42.3-R42.6 — Execute inherited v36 R41.3-R41.6 unchanged: target/HTTP/
   composition/Runtime, strict Core/loaded Views, complete and fresh installed
   verification, then exact candidate audit and fresh independent dual review.

## Deterministic acceptance

- `TestWorkspaceGovernanceMigrationUpgradesRetainedVersionEightDatabase` and
  `TestSqliteMigrationsAreOrderedAtomicAndRepeatable` each require exactly 20
  catalogued Workspace migrations after S07D installation.
- A second migration run remains idempotent and no duplicate catalog row or
  schema object appears.
- The full `go test ./internal/modules/workspace/... -count=1` graph passes with
  the same test scope that exposed the stale assertions. No assertion is
  skipped, weakened, or scoped out.

All inherited v35-v36 acceptance remains mandatory. Release 3 remains
incomplete until PCR-S07D closes with fresh independent `SPEC PASS` plus
`CODE/SECURITY/QUALITY PASS` and a separate aggregate DoneGate verifies S07A-D.

## Exclusions and stop conditions

Every v35-v36 exclusion and stop condition is incorporated unchanged. Push,
merge, deployment, generated protobufs, `server/**`, permanent Retrospective
deletion, target deletion/unlink/re-target, Knowledge integration, realtime
Retrospectives, external services, and Release 3 closure remain excluded from
r042. Any further required path outside the exact v37 boundary stops r042 and
requires another immutable successor.
