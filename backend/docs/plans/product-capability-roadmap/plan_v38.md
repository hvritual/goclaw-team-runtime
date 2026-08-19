# Product capability roadmap v38 — S07D exact Core test-path correction

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Plan-Version: `38`
- Task-Revision: `r043`
- Work-Item: `PCR-S07D`
- Exact base: `28e7a56c99855d21ed39494d2726318be5d64cd0`
- Predecessor plan: `plan_v37.md`
- Predecessor plan hash: `9ef4d4aa9dca5bf44142b30dc4f6edfff31694b16cf83656a5e294df1b92050c`
- Scope-blocked predecessor candidate: `f3a77a6c418332e1f0ea2c65073c4be87c41140d`
- Scope-blocked predecessor tree: `92d136706f61e4404012ae28aa4d0928c57a1bda`
- Status: `approved-active`
- Authority: the Human Customer's confirmed continuous Release 3 direction,
  confirmed prerequisite minimal outline authority, and repeated confirmed
  execution

## Successor trigger and immutable inheritance

R42.6 compared the exact 45-path S07D product candidate with the inherited
v35-v37 writable boundary. Forty-four paths matched byte-for-byte. The sole
mismatch is a filename-extension error in immutable v35: the plan lists
`packages/core/implementation-knowledge/mutations.test.ts`, while the existing
repository test file and the candidate path are
`packages/core/implementation-knowledge/mutations.test.tsx`. No `.ts` file
exists at either the exact base or the blocked candidate.

The candidate has zero `server/**` paths, zero generated paths, and no product
scope beyond the frozen Retrospective story. Its focused, complete, race,
frontend, production-build, and installed evidence remains useful diagnostic
evidence, but an allowed-path mismatch cannot be waived or renamed PASS.
Therefore r042 is `scope-blocked-before-independent-review`; candidate
`f3a77a6c` is never an accepted or reviewable release candidate.

Immutable v35-v37 are not amended. The blocked r042 branch remains preserved
for provenance. r043 starts from the last valid governance-only activation
`28e7a56c`, before every r042 product commit, and incorporates all v35-v37
product contracts, acceptance scenarios, ordered outputs, exclusions, and stop
conditions unchanged. The same product patch must be replayed only after this
successor is committed, with r043 trailers and fresh exact-candidate gates.

## Exact writable boundary

The exact product writable boundary is every path authorized by immutable v37,
with exactly one spelling correction:

- remove the nonexistent
  `packages/core/implementation-knowledge/mutations.test.ts`; and
- authorize the existing
  `packages/core/implementation-knowledge/mutations.test.tsx`.

This is a one-for-one path correction. It adds no production behavior, route,
schema, migration, permission, feature flag, public contract, dependency, or
test obligation. The resulting S07D product candidate must contain exactly 45
paths and must otherwise match the blocked r042 product patch. Every v1-v37
plan, protobuf/generated path, unrelated Auth/Bootstrap/Issue/Task/Knowledge
behavior, app-specific route, original dirty path, legacy backend tree, and
every `server/**` path remains read-only.

## Ordered execution

1. R43.1 — Freeze this successor from exact valid governance activation
   `28e7a56c`, create `codex/release3-s07d-r043`, register r043 and the r042
   scope block, and commit only the four governance pointer/register/journal/
   story-map paths plus this immutable plan with one continuous nine-field
   trailer block.
2. R43.2 — Replay the byte-identical R42.2 Canonical authority patch after
   authorization and commit it with r043 trailers.
3. R43.3 — Replay the byte-identical R42.3 target/HTTP/composition/Runtime patch
   and its bounded snapshot follow-up after authorization, preserving their
   separate commits and r043 trailers.
4. R43.4 — Replay the byte-identical R42.4 strict Core/loaded Views/four-locale
   patch, including the correctly authorized `.tsx` test path, and commit it
   with r043 trailers.
5. R43.5 — Run every inherited focused, complete Workspace, backend check,
   official race, Core, Views, typecheck, exact lint, root, production-build,
   and fresh two-identity installed-acceptance gate on the r043 candidate.
   Preserve every aggregate or environment NON-PASS without waiver.
6. R43.6 — Freeze one exact r043 candidate; verify path count, byte-equivalence,
   policy/plan hashes, continuous trailers, clean/isolated worktrees, zero
   `server/**` and generated paths, and closed owned processes; then obtain
   fresh independent `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` before
   closing PCR-S07D.

## Deterministic acceptance

- The r043 product diff has exactly the same 45 product paths and bytes as the
  blocked r042 product patch, with the real `.tsx` mutation test explicitly in
  scope and the nonexistent `.ts` path absent.
- The complete inherited Retrospective contract and all v37 deterministic and
  installed acceptance remain mandatory; no earlier r042 result substitutes
  for a fresh r043 candidate gate.
- Every r043 commit after activation contains one complete, continuous,
  ordered nine-field trailer block naming plan v38/r043 and its exact plan hash.
- Only fresh independent dual PASS may close r043/PCR-S07D. Release 3 remains
  incomplete until a later separate aggregate DoneGate verifies S07A-D.

## Exclusions and stop conditions

Every v35-v37 exclusion and stop condition is incorporated unchanged. Push,
merge, deployment, generated protobufs, `server/**`, permanent Retrospective
deletion, target deletion/unlink/re-target, automatic Knowledge integration,
realtime Retrospectives, external services, and Release 3 closure remain
excluded from r043. Any byte drift from the blocked product patch, any further
required path outside the corrected exact boundary, any missing/duplicate
trailer, hidden failure, unclosed owned process, dirty-tree overlap, or either
independent-review BLOCK stops r043 and requires another immutable successor.
