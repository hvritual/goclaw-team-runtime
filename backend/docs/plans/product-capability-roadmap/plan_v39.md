# Product capability roadmap v39 — Release 3 aggregate DoneGate

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `39`
- Task-Revision: `r044`
- Work-Item: `PCR-RELEASE-3-DONEGATE`
- Exact base: `8150a0e53defe1562c5ea5b41de34bbdba3a178e`
- Release 3 base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Predecessor plan: `plan_v38.md`
- Predecessor plan hash: `3d82ae02163a19f2ba2912db433402111b95699dbb1c204e6016abfadbe3c54d`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 completion
  direction, repeated confirmed execution, and confirmed prerequisite minimal
  outline authority

## Purpose and frozen inputs

PCR-S07A-D have each passed their own frozen deterministic, installed,
exact-candidate, and fresh independent dual-review gates. Their closure commits
are now immutable aggregate inputs:

| Story | Plan/task | Exact reviewed candidate | Candidate tree | Governance closure |
| --- | --- | --- | --- | --- |
| PCR-S07A | v28/r033 | `b3828be7b9b272732c5630975e73e35b629ed9f9` | `7c4a45fff414a555688358bd938111f8105c774f` | `07aef1a577db78598c92c70312a33989e6177d64` |
| PCR-S07B | v30/r035 | `cd94396093ea73f3f9434fed7410036ae61170ab` | `7e6f045ec5a48c4465e7f2fd5261e0d2a3b4b42d` | `f5695de83d55e277c8eeb9db7461b81137dc93ad` |
| PCR-S07C | v34/r039 | `47ee4189cb5571ec38ae39480c758d4decad22bd` | `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca` | `1d515efcca0919eed1e8a811c53d015efa89dfa3` |
| PCR-S07D | v38/r043 | `64091302b703a4590bdbe88d154f65fec9d6b37c` | `e696d67ad72aad52bc53e4a6bfe3211aac2f89d7` | `8150a0e53defe1562c5ea5b41de34bbdba3a178e` |

This successor authorizes only the aggregate proof needed to decide Release 3.
It does not reopen, reinterpret, or amend any story contract, earlier plan,
candidate, blocker, or retained NON-PASS. A story-specific defect discovered by
this gate cannot be repaired inside r044; it requires a new immutable successor
and a newly scoped product task.

## Exact writable boundary

r044 may change only:

- `backend/docs/plans/product-capability-roadmap/plan_v39.md` at R44.1 only;
- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`; and
- `backend/docs/plans/product-capability-roadmap/capability-matrix.md`, limited
  to evidence-backed Release 3 reconciliation and aggregate-verification text.

No backend, frontend, test, migration, locale, configuration, dependency,
generated, legacy, app-specific, original dirty-worktree, or `server/**` path
is writable. `plan_v39.md` is immutable after its activation commit. Product
behavior and bytes must remain identical to exact base `8150a0e5` throughout
r044.

## Ordered execution

1. R44.1 — From clean exact base `8150a0e5`, create
   `codex/release3-donegate-r044`, freeze this plan, register the aggregate task,
   and activate only `PCR-RELEASE-3-DONEGATE` with one complete continuous
   ordered nine-field trailer block.
2. R44.2 — Verify the four frozen story candidate/tree/closure tuples, all
   v26-v38 plan hashes, continuous trailer lineage from Release 3 base, zero
   active story tasks, Release 3 path scope, original dirty-tree isolation, and
   zero `server/**` or generated paths. Reconcile only the Project Resources,
   Requirements/coverage/outline-prerequisite, and Retrospectives rows in the
   capability matrix to that proven evidence, then commit the documentation-only
   result with r044 trailers.
3. R44.3 — On the unchanged product tree, freshly run the complete Workspace
   graph, backend `make check`, official full `make test-race`, Core and Views
   full tests, Core/Views and forced-root typechecks, package lint evidence, the
   forced root aggregate test, and production Web build. Known aggregate or
   environment NON-PASS results must be retained and isolated; none may be
   waived, hidden, retried with reduced scope, or renamed PASS.
4. R44.4 — Against a new database and one real Canonical backend plus production
   Web build, use two authenticated identities and one Project to prove all four
   Release 3 verticals coexist: Resource create/reorder/archive/restore;
   Requirement draft/review/approval/freeze/material-change/re-review plus the
   minimal authorized root outline and linked/implemented/accepted coverage;
   Retrospective participant/facilitator authority, publish/supersede/archive,
   default Task and explicit Issue action targets, stable replay, immutable
   provenance, and restart persistence. Verify loaded feature flags, strict
   workspace/project isolation, visible links/history, no Next error overlay,
   and no console error. Record exact IDs and outcomes, then close every owned
   process, listener, browser tab, database, binary, log, cache, and external
   evidence directory.
5. R44.5 — Freeze one exact documentation-only r044 candidate. Verify its base,
   tree, patch hash, exact allowed paths, policy/plan hashes, clean worktree,
   zero original-dirty overlap, zero product-byte drift, closed processes, and
   complete r044 trailers. Obtain fresh independent `SPEC PASS` and
   `CODE/SECURITY/QUALITY PASS` over the aggregate candidate and all ordered
   evidence.
6. R44.6 — Only after both fresh PASS decisions, record the exact candidate and
   evidence in the four mutable governance records, mark r044 complete, mark
   Release 3 complete with zero active tasks, and leave Release 4 inactive.

## Deterministic acceptance

- Exact base is the S07D governance closure above; its parent candidate and all
  earlier story candidate/tree/closure tuples resolve without drift.
- Every plan from v26 through v39 hashes to its registered value. Both governing
  policy hashes remain
  `6bd6e9f4207b6657b4463564db750a9e4329d5896e74a21fa8839aa940af3646`
  and
  `fc24a977573ea9e36da00d46e8492f7062235a30af4c38aa690e37bc3c5d5209`.
- The complete Release 3 range from `80d92b14` contains no `server/**` or
  generated path, no original dirty-path overlap, and no commit with a missing,
  duplicate, discontinuous, reordered, or mismatched nine-field trailer block.
- r044 changes only the six paths listed above and changes zero product bytes.
  The capability matrix distinguishes the historical static baseline from its
  Release 3 evidence reconciliation and makes no claim beyond passed runtime
  evidence.
- The unchanged complete Workspace/backend/race/Core/Views/type/build gates and
  the fresh installed aggregate journey pass all story-owned checks. A retained
  broad NON-PASS is non-blocking only when exact isolation proves it is
  pre-existing or unrelated and fresh independent review agrees; a Release 3
  regression, weaker retry, skipped mandatory gate, or hidden diagnostic blocks
  closure.
- Fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` are both
  mandatory. No primary-agent conclusion substitutes for either decision.

## Exclusions and stop conditions

Release 4/S08+, S10 expansion beyond the already closed minimal prerequisite,
generated protobufs, automatic Knowledge integration, realtime
Retrospectives, permanent Retrospective delete, target re-link/re-target,
external services, original dirty paths, legacy backend writes, push, merge,
deployment, and every `server/**` path remain inactive or excluded.

Stop r044 without claiming Release 3 complete on any frozen-tuple/hash drift,
product-byte change, out-of-bound path, capability-matrix overclaim, story-owned
test or installed failure, dirty overlap, unclosed owned process/listener,
missing or malformed trailer, or either independent-review BLOCK. Repair then
requires a new immutable plan/task; this plan may not absorb product changes.
