# Product capability roadmap — single-authority closure plan v8

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `8`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-18`
- Base commit: `14908b9e53a73330c0cde3fe8a3f602635906858`
- Active step: `PCR-S01B-8`
- Task revision: `r013`
- Supersedes for future execution: `plan_v7.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Repair the sole v7 independent-review authority block by establishing exactly
one active roadmap task. Preserve all prior immutable snapshots and the verified
product candidate, obtain fresh independent review, and close Release 0 only if
no authority, traceability, implementation, quality, scope, or Release-state
BLOCK remains.

## 2. Verified starting state

- Product candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90`
  remains unchanged and has passed implementation SPEC, code quality,
  deterministic, race, attachment-concurrency, policy, scope, and `server/**`
  review.
- v7 activation `14908b9e53a73330c0cde3fe8a3f602635906858`
  has exact base/parent/ancestry, hashes, trailers, links, and five-path scope.
- v7 independent review found one BLOCK: the task register marks both
  `PCR-001-S01B6-R11` and `PCR-001-S01B7-R12` active, while the plan pointer and
  journal claim one active authority.
- This plan starts from the exact v7 activation commit and does not amend any
  prior immutable plan.

## 3. Included scope

- create immutable v8/r013 authority and activation records;
- mark r011 and r012 `independent-review-blocked` with their exact review reason;
- establish r013 as the only active task and remove stale present-tense claims
  that v6 or v7 remains active;
- mechanically prove exactly one active task across plan pointer, task register,
  story map, and journal;
- verify base, ancestry, policy hashes, trailers, links, exact documentation
  scope, dirty exclusions, focused governance tests, and `server/**` boundary;
- obtain fresh independent read-only review;
- after PASS only, mark r013 and Release 0 complete while Release 1 remains
  inactive.

## 4. Excluded scope

- any product/test code, schema, migration, generated output, API, frontend,
  Desktop, Control Plane, runtime, external-data, or `server/**` change;
- amendment of immutable `plan_v1.md` through `plan_v7.md`;
- push, merge, deployment, Release 1 activation, or capability work;
- deletion, cleanup, backfill, migration, or mutation of retained data;
- staging or absorbing any unrelated dirty path.

## 5. Ordered steps

### PCR-S01B-8.1 — Freeze one authority

Create and commit this plan plus the plan pointer, task register, story map, and
append-only activation journal. r011 and r012 become review-blocked; r013 is the
only active task. The commit contains exactly the Section 6 paths and valid r013
trailers.

### PCR-S01B-8.2 — Verify exact evidence

Resolve the exact v8 base and direct parent, verify immutable prior plans are
unchanged, enumerate active task markers, validate links and hashes, compare the
candidate range with Section 6, confirm dirty exclusions and empty `server/**`,
and re-run focused governance/Bootstrap tests.

### PCR-S01B-8.3 — Independent review

Obtain a fresh independent review of base/ancestry, one-active-task authority,
prior-plan immutability, candidate preservation, exact scope, traceability,
Release 0 readiness, and Release 1 inactivity. Any BLOCK stops closure and
requires a new approved plan/task.

### PCR-S01B-8.4 — Release 0 closure

Only after deterministic evidence and independent PASS, append indexed evidence,
mark r013 and Release 0 complete, keep Release 1 inactive, set no active Release
1 task, and commit the exact closure records.

## 6. Exact writable paths

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v8.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other path may be modified or staged. Any required path outside this list
stops execution and requires another plan/task revision.

## 7. Acceptance criteria

1. This approved immutable plan consistently names existing exact base
   `14908b9e53a73330c0cde3fe8a3f602635906858`.
2. r011 and r012 are review-blocked; r013 is the sole active task before closure.
3. Plan pointer, task register, story map, and journal agree on the same sole
   authority and Release state.
4. Prior immutable plans remain byte-for-byte unchanged and their mismatches or
   blocks remain indexed historical evidence.
5. Activation and closure candidates contain only Section 6 paths and valid
   r013 trailers.
6. Product candidate `4d60e50` remains unchanged; frozen implementation and
   quality PASS evidence remains valid.
7. Policy hashes, Markdown links, diff checks, ancestry, dirty exclusions,
   focused tests, and `server/**` boundaries pass.
8. Fresh independent review reports no BLOCK.
9. Closure records mark Release 0 complete, Release 1 inactive, and no Release 1
   task active.

## 8. Deterministic verification

From the repository root unless stated otherwise:

```text
git rev-parse 14908b9e53a73330c0cde3fe8a3f602635906858^{commit}
git merge-base --is-ancestor 14908b9e53a73330c0cde3fe8a3f602635906858 HEAD
git show -s --format=%P HEAD
git show -s --format=%B HEAD
git diff --check
git diff --cached --check
git diff --name-only 14908b9e53a73330c0cde3fe8a3f602635906858..HEAD
git diff --name-only -- server
git diff --cached --name-only -- server
git status --porcelain -- server
cd backend && go test ./internal/modules/workspace/contract ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace ./internal/bootstrap -count=1
```

Additionally validate every roadmap-relative Markdown link, recompute SHA256 for
`CLAUDE.md`, `backend/AGENTS.md`, and `plan_v8.md`, and mechanically enumerate
every task-register `Status: active` entry. Exactly one must exist before
closure; zero must exist after closure.

## 9. Risks and controls

| Risk | Control |
| --- | --- |
| stale task remains executable | mechanical active-status enumeration plus independent review |
| prior history is rewritten | immutable plans untouched; blocks recorded append-only |
| documentation authority expands into product work | exact five-path allowlist and explicit exclusions |
| Release 1 starts implicitly | closure requires Release 1 inactive and zero active Release 1 tasks |
| unrelated dirty paths enter commits | explicit staging and staged-path audit |

## 10. Rollback

- Before commit, revert only uncommitted v8 documentation paths by explicit
  patch.
- After commit, use a new revert commit if directed; never reset shared history.
- Never amend prior plans, modify product code, mutate data, touch `server/**`,
  or absorb unrelated dirty paths as rollback.

## 11. Approval record

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v8 /
r013` on 2026-08-18 after receiving the exact one-active-task repair, base,
independent-review, closure, Release-state, and exclusion boundaries. This does
not authorize product changes, push, merge, deployment, Release 1, schema
changes, external data handling, or amendment of immutable prior plans.
