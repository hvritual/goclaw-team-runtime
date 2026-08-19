# Product capability roadmap v36 — S07D complete Project-deletion authority

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `36`
- Task-Revision: `r041`
- Work-Item: `PCR-S07D`
- Exact base: `fbfc1c05b622e08540564b235661b53cc2894aad`
- Predecessor plan: `plan_v35.md`
- Predecessor plan hash: `c2c112bb1b903caad6f87e1de14c8ebdb4d5f4ff89665b47bbc734a79c10237a`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and repeated confirmed
  execution

## Successor trigger and immutable inheritance

v35/r040 activated PCR-S07D from exact S07C closure and froze the complete
governed Retrospective product contract. During assertion-first R40.2 work,
read-only dependency inspection proved that the installed Canonical Workspace
has two Project deletion repositories:

1. `ProjectSurfaceRepository.DeleteProject`, used by the Canonical HTTP Project
   surface; and
2. `projectRepository.DeleteWithDependents`, used by the installed local/proto
   Project service.

Both delete the same canonical `workspace_projects` identity. v35 authorized
the first repository but inadvertently omitted
`backend/internal/modules/workspace/internal/infrastructure/sqlite/project_repository.go`.
Cleaning Retrospective authority from only one installed deletion entry would
violate v35's Project-deletion contract and leave canonical rows behind through
the other entry. No correct implementation can close that gap without the
omitted path.

v35 remains immutable. r040 is `superseded-before-product-commit`: its only
commit is the five-path governance activation `fbfc1c05`; no S07D product
candidate was committed under r040. v36/r041 incorporates every v35 frozen
product contract, acceptance scenario, deterministic gate, installed gate,
review requirement, exclusion, and stop condition unchanged. This successor
changes only the one-path write boundary needed to make Project deletion
complete across both already-installed entries.

## Frozen deletion delta

- Add one shared Retrospective cleanup function operating on the existing
  transaction executor abstraction.
- Both installed Project deletion repositories must invoke that function inside
  their existing Project-deletion transaction before deleting the Project.
- Cleanup removes only current Retrospective heads, immutable revision rows,
  revisioned participant snapshots, pending/completed action-link rows, and
  Retrospective resource-revision authority owned by the deleted Project.
- Cleanup never deletes, updates, re-targets, or unlinks the target Task/Issue.
  The existing Project deletion behavior may clear the target's Project field,
  but the target identity and content survive.
- Retrospective audit and outbox evidence remains under its existing immutable
  retention authority. This successor does not weaken v35's safe evidence or
  migration-down guards.
- Each deletion entry is tested independently. A forced later failure rolls
  back both the Project and every Retrospective cleanup effect.
- Workspace and Project predicates are mandatory on every cleanup statement;
  a same-ID Retrospective in another Project or Workspace is untouched.

No route, response, schema, migration, Retrospective lifecycle, target claim,
Task/Issue creation behavior, UI, feature flag, generated contract, external
service, or public API changes as a result of this successor.

## Exact writable boundary

The exact writable boundary is every path listed by immutable `plan_v35.md`,
plus exactly:

- `backend/internal/modules/workspace/internal/infrastructure/sqlite/project_repository.go`.

No other path is added. In particular every v1-v35 plan, protobuf/generated
path, unrelated Auth/Bootstrap/Issue/Task/Knowledge behavior, app-specific
route, original dirty path, legacy backend tree, and every `server/**` path is
read-only.

## Ordered execution

1. R41.1 — Freeze this successor from exact r040 governance activation, rename
   the isolated branch to `codex/release3-s07d-r041`, register r041, and commit
   only the four governance pointer/register/journal/story-map paths plus this
   immutable plan with one continuous nine-field trailer block.
2. R41.2 — Complete v35 R40.2 assertion-first domain, migration, repository,
   transaction authorization, persisted-read, revision/history, lifecycle,
   governance, audit, outbox, down-guard, and both-entry Project-deletion tests;
   GREEN only the Canonical Retrospective authority.
3. R41.3 — Complete v35 R40.3 target-claim, Task-default, explicit-Issue,
   interruption/concurrency/idempotency, HTTP, composition, Runtime,
   permission, and feature-flag vertical through injected target services.
4. R41.4 — Complete v35 R40.4 strict Core and loaded four-locale shared UI.
5. R41.5 — Run every v35 R40.5 focused, complete, race, frontend, root, build,
   and fresh two-identity installed-acceptance gate; record unrelated aggregate
   NON-PASS honestly.
6. R41.6 — Freeze one exact candidate and run every v35 R40.6 scope, hash,
   trailer, dirty-tree, process, and fresh independent dual-review gate before
   closing PCR-S07D.

## Additional deterministic acceptance

- Seed a published Retrospective with a completed action link and its linked
  Task/Issue. Deleting the owning Project through each installed deletion
  repository removes all Project-owned Retrospective current authority while
  retaining the target and immutable Retrospective audit/outbox evidence.
- Seed same-ID foreign ownership and prove the cleanup does not cross Workspace
  or Project boundaries.
- Force a failure after Retrospective cleanup in each deletion transaction and
  prove exact before/after preservation of the Project, Retrospective authority,
  target, audit, and outbox rows.

All v35 deterministic and installed acceptance remains mandatory. Fresh
independent `SPEC PASS` plus `CODE/SECURITY/QUALITY PASS` is still required.
Release 3 remains incomplete until PCR-S07D closes and a separate aggregate
DoneGate verifies S07A-D together.

## Exclusions and stop conditions

Every v35 exclusion and stop condition is incorporated unchanged. Push, merge,
deployment, generated protobufs, `server/**`, permanent Retrospective deletion,
target deletion, target unlink/re-target, Knowledge integration, realtime
Retrospectives, external services, and Release 3 closure remain excluded from
r041. Any further required path outside the exact v36 boundary stops r041 and
requires another immutable successor.
