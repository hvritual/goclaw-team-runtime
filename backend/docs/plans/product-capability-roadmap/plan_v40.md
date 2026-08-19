# Product capability roadmap v40 — Release 3 historical-trailer audit correction

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `40`
- Task-Revision: `r045`
- Work-Item: `PCR-RELEASE-3-DONEGATE`
- Exact base: `a1f6a934370ddbc2d645035767c80a24edf2ad4c`
- Release 3 base: `80d92b14b1f5a1525fbce0c60ce992e28c7f0e8b`
- Predecessor plan: `plan_v39.md`
- Predecessor plan hash: `ba58180a641ba2bbcbead3673c9f19c987a85e001a581137e3850eb36158f909`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 completion
  direction, repeated confirmed execution, and confirmed prerequisite minimal
  outline authority

## Successor trigger and exact correction

R44.2 resolved all four frozen story candidate/tree/closure tuples, matched every
registered v26-v39 plan hash, and found a Release 3 range of 131 paths: 114
product paths plus 17 roadmap paths, zero `server/**`, zero generated paths, and
zero r044 product drift. The optimized single-read trailer audit then correctly
found one historical exception to v39's over-strict uniform-nine-field rule.

Release 3 activation `71afb3c33a4d82431a8016cb195a97e5a36d8646`
has one continuous ordered eight-field trailer block: `Task-ID`, `Project-ID`,
`Task-Revision`, `Work-Item`, `Plan-ID`, `Plan-Version`, `Plan-Step`, and
`Policy-Bundle`. It has no `Issue` field. This matches `backend/AGENTS.md`, which
requires `Issue` only "when present", and matches the v26/r031 frozen authority,
which did not register an Issue value. Every later Release 3 commit through the
r044 activation has one continuous ordered nine-field block including `Issue`.

v39 instead required every commit in the complete historical range to carry all
nine fields. That assertion cannot pass and cannot be waived or repaired by
rewriting immutable history. r044 therefore stops as
`audit-blocked-before-capability-matrix-write`; it changes no capability matrix
or product byte and makes no aggregate PASS claim. v40 corrects only the
historical audit predicate. It does not excuse a missing field that was present
in a commit's frozen authority, amend v26-v39, or relax any current r045 trailer.

## Immutable inheritance and writable boundary

Every v39 frozen story tuple, gate, combined installed scenario, exact writable
path, exclusion, cleanup requirement, independent-review requirement, and stop
condition is incorporated unchanged, except for this single deterministic
lineage rule:

- commit `71afb3c3` must contain exactly its frozen continuous ordered
  eight-field block and no `Issue` line; and
- every subsequent Release 3/r044/r045 commit must contain exactly one complete
  continuous ordered nine-field block, including `Issue`, matching its frozen
  plan/task/policy values.

r045 may change only the same six roadmap paths authorized by v39, with
`plan_v40.md` replacing `plan_v39.md` as the one-time immutable activation file:

- `backend/docs/plans/product-capability-roadmap/plan_v40.md` at R45.1 only;
- `backend/docs/plans/product-capability-roadmap/plan.md`;
- `backend/docs/plans/product-capability-roadmap/task-register.md`;
- `backend/docs/plans/product-capability-roadmap/story-map.md`;
- `backend/docs/plans/product-capability-roadmap/journal.md`; and
- `backend/docs/plans/product-capability-roadmap/capability-matrix.md`, limited
  to evidence-backed Release 3 reconciliation and aggregate-verification text.

All product/test/migration/locale/config/dependency/generated/original-dirty/
legacy/`server/**` paths remain read-only. Product behavior and bytes must stay
identical to exact S07D closure `8150a0e5`.

## Ordered execution

1. R45.1 — Preserve branch `codex/release3-donegate-r044` at exact blocked
   activation `a1f6a934`; create `codex/release3-donegate-r045`, freeze this
   successor, mark r044 audit-blocked, and activate only r045 with one current
   nine-field trailer block.
2. R45.2 — Rerun the complete v39 R44.2 audit with the exact historical rule
   above. Verify the live original dirty worktree directly, reconcile only the
   three Release 3 capability-matrix rows after PASS, and commit that
   documentation-only evidence with r045 trailers.
3. R45.3 — Execute v39 R44.3 complete deterministic gates unchanged.
4. R45.4 — Execute v39 R44.4 fresh two-identity combined installed acceptance
   unchanged and remove every owned artifact/process/listener afterward.
5. R45.5 — Execute v39 R44.5 exact candidate freeze and fresh independent dual
   review unchanged, using v40/r045 hashes and trailers.
6. R45.6 — Only after both fresh PASS decisions, execute v39 R44.6 closure and
   mark Release 3 complete with zero active tasks; leave Release 4 inactive.

## Acceptance and stop conditions

- The corrected full lineage audit passes exactly 42 commits through r044
  activation plus all new r045 commits: one frozen eight-field historical block
  and otherwise nine-field blocks. No duplicate, discontinuity, order drift,
  value drift, or unregistered omission is allowed.
- All v39 deterministic, installed, scope, product-byte, matrix, dirty-tree,
  process, and independent-review acceptance remains mandatory without waiver.
- The earlier dynamic-revision script error, invalid universal-predecessor-hash
  assumption, slow per-commit audit timeout, and the final valid r044 trailer
  BLOCK remain disclosed tooling/governance evidence; none is a product PASS.
- Any other trailer exception, product-byte change, out-of-bound path, story
  regression, installed failure, capability-matrix overclaim, dirty overlap,
  unclosed process/listener, or independent-review BLOCK stops r045 and requires
  another immutable successor.

Release 4/S08+, S10 expansion, generated protobufs, external services, original
dirty paths, push, merge, deployment, legacy backend writes, and `server/**`
remain inactive or excluded.
