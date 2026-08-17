# Product capability roadmap — authority closure plan v7

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Version: `7`
- Status: `approved-for-execution`
- Approved by: `Human Customer, 2026-08-17`
- Base commit: `4d60e50d9c03a68b2723b427506ea7db64d90d90`
- Active step: `PCR-S01B-7`
- Task revision: `r012`
- Supersedes for future execution: `plan_v6.md`
- Product contract: unchanged `PCR-CONTRACT-v1` and
  [s01b-foundation-design.md](s01b-foundation-design.md)

## 1. Objective

Repair the sole v6 independent-review authority/traceability block without
rewriting the immutable v6 snapshot or changing the verified product candidate.
Re-establish an internally consistent approved base, obtain a fresh independent
review, and close Release 0 only if that review reports no BLOCK.

## 2. Verified starting state

- Product candidate `4d60e50d9c03a68b2723b427506ea7db64d90d90`
  passed the v6 implementation, deterministic, race, attachment-concurrency,
  policy, scope, and `server/**` gates.
- Independent review passed implementation SPEC and code quality.
- The only review BLOCK is that immutable `plan_v6.md` declares nonexistent
  base `f93eca77c3450109b7328441812d63710f179521`, while the actual Git object,
  v6 task register, plan pointer, and activation journal use
  `f93eca764bb464245ef096429701aa0a856f0c56`.
- This plan does not amend or reinterpret `plan_v6.md`. It establishes new
  execution authority from the already verified candidate at this plan's exact
  base commit.

## 3. Included scope

- create the immutable v7 authority snapshot and r012 task record;
- preserve the v6 mismatch as indexed historical evidence;
- verify the exact v7 base, ancestry, candidate trailers, documentation scope,
  policy hashes, dirty-path exclusions, and empty `server/**` scope;
- obtain a fresh independent read-only SPEC/quality/traceability review;
- after PASS only, synchronize roadmap records and mark Release 0 complete;
- keep Release 1 inactive.

## 4. Excluded scope

- any product-code, test-code, schema, migration, generated, API, frontend,
  Desktop, Control Plane, runtime, external-data, or `server/**` change;
- amendment of any immutable `plan_v1.md` through `plan_v6.md` snapshot;
- push, merge, deployment, Release 1 activation, or capability work;
- deletion, cleanup, quarantine, backfill, or rewrite of retained data;
- absorbing any unrelated dirty worktree path.

## 5. Ordered steps

### PCR-S01B-7.1 — Freeze authority

Create and commit this immutable plan plus its plan pointer, task, story-map,
and append-only activation journal records. The commit must contain only the
Section 6 paths and machine-readable r012 trailers.

### PCR-S01B-7.2 — Verify exact evidence

Prove the v7 base resolves exactly, `HEAD` descends from it only through the v7
activation record, the product candidate remains unchanged, all frozen policy
hashes resolve, unrelated dirty paths remain excluded, and `server/**` remains
empty. Re-run focused governance tests as non-mutating confirmation.

### PCR-S01B-7.3 — Independent review

Obtain a fresh independent read-only review against this exact v7 authority.
The reviewer must distinguish product implementation, code quality,
authority/traceability, scope, and Release-state boundaries. Any BLOCK stops
closure and requires a new approved plan/task.

### PCR-S01B-7.4 — Release 0 closure

Only after deterministic evidence and independent PASS, append the indexed
review evidence, mark r012 and Release 0 complete, retain Release 1 as inactive,
and commit the exact closure records.

## 6. Exact writable paths

- `backend/docs/plans/product-capability-roadmap/plan.md`
- `backend/docs/plans/product-capability-roadmap/plan_v7.md`
- `backend/docs/plans/product-capability-roadmap/story-map.md`
- `backend/docs/plans/product-capability-roadmap/task-register.md`
- `backend/docs/plans/product-capability-roadmap/journal.md`

No other path may be modified or staged. Any required path outside this list
stops execution and requires another plan/task revision.

## 7. Acceptance criteria

1. This approved immutable plan names the existing exact base object
   `4d60e50d9c03a68b2723b427506ea7db64d90d90` consistently.
2. v6 remains byte-for-byte unchanged and its base mismatch is preserved as
   historical evidence rather than silently corrected.
3. The v7 activation and closure candidates contain only Section 6 paths and
   have valid r012 traceability trailers.
4. Product candidate `4d60e50d` remains unchanged and all v6 product/code gates
   remain independently accepted.
5. Policy hashes, relative links, Markdown diff checks, ancestry, dirty-path
   exclusions, and `server/**` boundaries pass.
6. Fresh independent review reports no BLOCK across implementation, quality,
   authority/traceability, scope, and Release-state dimensions.
7. Closure records mark Release 0 complete and Release 1 inactive.

## 8. Deterministic verification

From the repository root unless stated otherwise:

```text
git rev-parse 4d60e50d9c03a68b2723b427506ea7db64d90d90^{commit}
git merge-base --is-ancestor 4d60e50d9c03a68b2723b427506ea7db64d90d90 HEAD
git show -s --format=%B 4d60e50d9c03a68b2723b427506ea7db64d90d90
git diff --check
git diff --cached --check
git diff --name-only 4d60e50d9c03a68b2723b427506ea7db64d90d90..HEAD
git diff --name-only -- server
git diff --cached --name-only -- server
git status --porcelain -- server
cd backend && go test ./internal/modules/workspace/contract ./internal/modules/workspace/internal/application ./internal/modules/workspace/internal/infrastructure/sqlite ./internal/modules/workspace ./internal/bootstrap -count=1
```

Validate all roadmap-relative Markdown links and re-compute the frozen SHA256
values for `CLAUDE.md`, `backend/AGENTS.md`, and `plan_v7.md`. Compare changed
paths exactly with Section 6 before each commit and independent review.

## 9. Risks and controls

| Risk | Control |
| --- | --- |
| v6 history is silently rewritten | immutable snapshots remain untouched; append-only evidence records the mismatch |
| documentation plan expands product authority | exact five-path allowlist and explicit no-product-code boundary |
| passing implementation is mistaken for closure | independent v7 traceability PASS remains mandatory |
| Release 1 starts implicitly | plan, task, story map, and journal keep Release 1 inactive |
| unrelated dirty paths enter a commit | explicit staging and staged-name audit before every commit |

## 10. Rollback

- Before commit, revert only uncommitted v7 documentation paths by explicit
  patch.
- After commit, use a new revert commit if directed; never reset shared history.
- Never amend v6, change product code, mutate data, touch `server/**`, or absorb
  unrelated dirty paths as rollback.

## 11. Approval record

The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v7 /
r012` on 2026-08-17 after receiving the exact documentation-only authority
repair, base, independent-review, Release-state, and exclusion boundaries. This
does not authorize product changes, push, merge, deployment, Release 1, schema
changes, external data handling, or amendment of immutable prior plans.
