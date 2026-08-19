# Product capability roadmap

The approved execution snapshot is [plan_v30.md](plan_v30.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `30`
- Active step: `none`
- Status: `Release 3 active; PCR-S07A and PCR-S07B complete-independent-reviewed; no active task`
- Plan base commit: `fa1153164882adb4880d57f7349597900196402f`
- Last closed task candidate: `cd94396093ea73f3f9434fed7410036ae61170ab`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)
- S01B foundation design: [s01b-foundation-design.md](s01b-foundation-design.md)

The Human Customer confirmed execution of the reviewed Release 3 plan on
2026-08-19. `PRODUCT-CAPABILITY-ROADMAP-001 v26 / r031` started from exact
Release 2 closure `80d92b14` and activated only PCR-S07A governed Project
Resources. Its exact 42-path candidate tree `65591332` passed focused and
installed checks, but fresh independent review returned both `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK` for rollback authority guards, transactional
authorization/revision/refresh boundaries, one HTTP permission mapping, secret
logging, and explicit SQLite indexes. r031 is review-blocked and v26 remains
immutable.

The Customer's confirmed continuous Release 3 direction activates
`PRODUCT-CAPABILITY-ROADMAP-001 v27 / r032` from exact governance base
`71afb3c3` only to remediate those S07A findings. The approved connection policy
still saves locally valid, credential-free Resources independently of external
reachability; typed adapter failure projects a safe degraded/unavailable state
and never deletes an external target. S07B, S07C, S07D, Release 3 completion,
push, merge, and deployment remain inactive.

The v27/r032 exact candidate `9fb86ea` (tree `c6a2f08f`) passes its focused,
backend check, exact race, typecheck, production-build, and retries-disabled
installed-Chrome gates, but fresh independent review returns `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK`. Blank lines make eight required commit fields
unparseable as Git trailers, and the blocked down-migration test does not
compare retained row values before and after failure. No further product or
security blocker was found. r032 is review-blocked and immutable v27 remains
unchanged. The standing completion direction activates v28/r033 from exact
base `9fb86ea` only for continuous trailer proof and exact down-migration data
preservation assertions. S07B-D, Release 3 completion, push, merge, and
deployment remain inactive.

Exact v28/r033 candidate `b3828be7b9b272732c5630975e73e35b629ed9f9`
(tree `7c4a45fff414a555688358bd938111f8105c774f`) adds complete
before/after retained-row assertions without changing product behavior. Both
range commits expose all nine required Git trailers; focused Workspace,
backend check, and official changed-package race gates pass. Fresh independent
review returns `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` with no blocking
finding. PCR-S07A is `complete-independent-reviewed`. PCR-S07B remained inactive
until its outline authority boundary was resolved in a successor plan; no push,
merge, deployment, or Release 3 completion was claimed by the S07A closure.

The Human Customer explicitly confirmed the prerequisite minimal outline
authority on 2026-08-19. `PRODUCT-CAPABILITY-ROADMAP-001 v29 / r034` starts
from exact S07A closure `07aef1a5` and activates only PCR-S07B. It freezes the
complete Requirement lifecycle, independent approval, project grants,
revisioned Issue/outline traceability, material-change review-required
projection, singular Canonical authority, and only persistent root-level
outline node create/read with stable IDs. S07C-D, nested hierarchy,
move/reorder/numbering, Issue-outline links, rollups/board, full outline UI,
`project_outline`, generated protobufs, push, merge, deployment, and Release 3
completion remain inactive or excluded.

R34.2-R34.5 install the Requirement domain/migration foundation, singular
Canonical SQLite/application/HTTP authority, transaction-owned deletion
cleanup, strict Core boundary, shared Requirement view, and the runtime
`project_requirements` capability through exact commits `29fc0ed0`,
`1c437313`, `cfffe2ae`, `8167897f`, `9fe63f11`, and `48599860`. Focused
backend/Core/Views/locale/type/lint checks, backend `make check`, the official
seven-package Windows race command, and two production Web builds pass. The
direct old-GCC race probe remains a truthful NON-PASS `0xc0000139` environment
result; the repository-owned wrapper passes with Scoop GCC 15.2.0. Root tests
remain NON-PASS on three pre-existing five-second aggregate-load timeouts while
the two implicated files pass 44/44 in isolation; broad Views lint remains
NON-PASS on pre-existing Knowledge/Skills paths outside r034.

Fresh SQLite installed-Chrome acceptance on the production standalone Web app
and real Canonical HTTP backend passes with independent lead/owner identities:
stable root outline create/link, Issue link, stale-revision 409, independent
approval/freeze, frozen plain-edit denial, material-change old-effective and
review-required behavior without Issue mutation, re-review/refreeze, reload-
persistent v1-v10 history, cross-project 404 without revision advance, and
terminal v11 retirement. Runtime config reports `project_requirements=true`
and `project_outline=false`.

Exact r034 candidate `fa1153164882adb4880d57f7349597900196402f`
(tree `70de766121f06487eeb7c259906b1bbd0118594f`) passes its scope,
continuous-trailer, clean-worktree, and process-cleanup audit. Fresh independent
review nevertheless returns `SPEC BLOCK` and `CODE/SECURITY/QUALITY BLOCK`:
the imported Issue-link down guard misses canonical workspace/project ownership
drift, and explicit live membership/lead/grant/terminal-project mutation-denial
proof is incomplete. r034 and immutable v29 are review-blocked.

The Customer's confirmed continuous Release 3 authority activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v30 / r035` from exact base `fa115316` to add
the two ownership-drift rollback guards/tests and the missing transaction-owned
live-authorization proofs. The S07B product and public contract otherwise stay
unchanged. S07C-D, Release 3 completion, generated protobufs, original dirty
paths, push, merge, deployment, and `server/**` remain inactive or excluded.

R35.2 captures both missing ownership guards as real RED failures against the
r034 down migration, then extends only the imported Issue-link guard so link,
baseline, and retained legacy workspace/project ownership must agree. Both
cases preserve exact retained rows/catalog after blocked rollback, and exact
untouched legacy rollback still succeeds. R35.3 proves current authorization
after membership removal, lead reassignment, editor-grant removal, and both
terminal project states; every denied save leaves baseline/revision,
idempotency, governance, link, grant, and outline effects unchanged. Existing
transaction authorization production code required no change.

Focused migration/repository checks and the complete Workspace package pass.
Backend `make check` passes in 349.6 seconds; the official seven-package race
passes in 409.4 seconds with repository-selected MinGW GCC 15.2.0. No frontend,
HTTP/Core, runtime composition, flag, or installed behavior changed, so the
r034 production-build and two-identity installed acceptance evidence remains
applicable.

Exact r035 candidate `cd94396093ea73f3f9434fed7410036ae61170ab`
has tree `7e6f045ec5a48c4465e7f2fd5261e0d2a3b4b42d`, binary patch hash
`0a9f24812076a842bff24a68c27edb3709974193`, eight authorized paths,
zero `server/**` or generated paths, three complete continuous trailer blocks,
a clean isolated worktree, and zero overlap with the original 25 dirty entries.
Fresh independent review returns `SPEC PASS` and
`CODE/SECURITY/QUALITY PASS` with no blocking finding. PCR-S07B is
`complete-independent-reviewed`; PCR-S07C remains inactive until its own
successor plan is frozen. No push, merge, deployment, or Release 3 completion
is claimed.

The Human Customer's standing direction to complete Release 2 activated
`PRODUCT-CAPABILITY-ROADMAP-001 v20 / r025` from exact Release 1 closure
`0aed3687`. Exact S05A candidate `17f7eb1f` passes its deterministic, installed,
scope, traceability, and fresh independent-review gates; PCR-S05A is
`complete-independent-reviewed`. Release 2 remains active with no active task.
S05B, S06A, and S06B require their next frozen plan versions and remain
inactive. Push, merge, and deployment remain excluded.

The same standing direction activates `PRODUCT-CAPABILITY-ROADMAP-001 v21 /
r026` from exact S05A closure `27def068`. Only PCR-S05B safe preview/import and
version-scoped logical file management is active. S06A and S06B remain inactive;
no push, merge, deployment, or Release 2 completion is authorized or claimed.

Exact S05B candidate `abce08d652c1af263a4e678a31732814d324330d`
passes deterministic, race, production-build, installed-Chrome, scope,
traceability, and fresh independent-review gates. PCR-S05B is
`complete-independent-reviewed`; Release 2 remains active with no active task.
S06A and S06B remain inactive pending their own frozen plan versions. Push,
merge, and deployment remain excluded.

The same standing direction activates `PRODUCT-CAPABILITY-ROADMAP-001 v22 /
r027` from exact S05B closure `78c5b5eb`. Only PCR-S06A authenticated,
revisioned, source-explained Knowledge query is active. S06B proposal/review,
Knowledge realtime, `knowledge_review`, push, merge, deployment, and Release 2
completion remain inactive.

Exact S06A candidate `3d465f1110fed1ce9bef8d076f77d4b553fda421`
passes backend deterministic/race, frontend type/build, installed-browser UI,
scope, traceability, rollback, and fresh independent-review gates. PCR-S06A is
`complete-independent-reviewed`; Release 2 remains active with no active task.
S06B requires its own frozen plan version. Push, merge, and deployment remain
excluded.

The same standing direction activates `PRODUCT-CAPABILITY-ROADMAP-001 v23 /
r028` from exact S06A closure `84ed0e4a`. Only PCR-S06B governed Knowledge
proposal/review/publication is active. Release 3, push, merge, deployment, and
Release 2 completion remain inactive until the exact S06B candidate passes all
frozen gates and fresh independent review.

The v23 candidate passes backend/race/type/build checks, but the frozen
dual-identity browser RED proves `knowledge:candidate_updated` is discarded by
the shared realtime Hub allowlist. Because that boundary lies outside v23
scope, r028 is verification-blocked. The standing completion direction
activates v24/r029 from exact base `ffcdd1c7` only for the Hub event allowlist,
shared event type, exact tests, and real member-to-owner browser journey.

Independent review also proves a raw allowlist change would disclose the full
private candidate projection to ordinary Workspace members. v24/r029 is
design-blocked before product implementation. v25/r030 starts from exact
activation base `33ec500f` and adds only role-aware per-client projection,
composition, exact transition/cancellation coverage, and the same dual-identity
browser acceptance.

Exact S06B candidate `32f273e47e6b9863b4ef462f28eed3e91da654d0`
passes backend deterministic/race, frontend type/build, focused realtime and
transition tests, installed-Chrome dual-identity no-reload acceptance, scope,
traceability, and fresh independent SPEC plus CODE/SECURITY/QUALITY review.
PCR-S06B and Release 2 are `complete-independent-reviewed`; no task is active.
Release 3, push, merge, and deployment remain inactive.

v17/r022 exact candidate `7479e07adcc1c703a52cb348af7919df2cc68553`
passes deterministic, performance, production-build, installed-Chrome, scope,
process-cleanup, and fresh independent-review gates. PCR-S03A is
`complete-independent-reviewed`. Release 1 remains active with no active task;
S03B and S04 require a new approved plan version.

v18/r023 exact candidate `c9e905bc7675d253991c0a816bcd19985a49c10b`
passes deterministic, performance, production-build, installed-Chrome, scope,
process-cleanup, commit-traceability, and fresh independent-review gates.
PCR-S03B is `complete-independent-reviewed`. Release 1 remains active with no
active task; S04 requires a new approved plan version and remains inactive.

The Human Customer's standing direction to complete Release 1 activates
`PRODUCT-CAPABILITY-ROADMAP-001 v19 / r024` from exact S03B closure `b864479`.
Only PCR-S04 atomic revisioned Pin reorder is active. Existing Issue and
Project search remain installed. No Release 2 story, push, merge, deployment,
or Release 1 completion is authorized or claimed.

v19/r024 exact candidate `b4fa3a3c60d24953c9f37c82cf0d70cda0fd7b11`
passes backend check/race, root typecheck/test, production-build,
fresh-database installed-Chrome, scope/process-cleanup, traceability, and fresh
independent-review gates. PCR-S04 and Release 1 are
`complete-independent-reviewed`; no task is active. Release 2 remains inactive,
and no push, merge, or deployment is claimed.

Candidate `5062e84` passed every frozen deterministic gate, but its independent
review returned `SPEC BLOCK` for authority mapping, canonical request hashing,
secret-free envelopes, outbox lease/tuple safety, and empty-only down migration.
The Human Customer explicitly approved `PRODUCT-CAPABILITY-ROADMAP-001 v5 /
r010` on 2026-08-17. Candidate `ee02403` passed the frozen deterministic gates,
but independent re-review returned `SPEC BLOCK`: deprecated raw envelope inputs
are silently ignored, Basic authorization material is not rejected, and
unknown-policy or unversioned outbox rows can be rewritten by claim/replay
before policy validation. The Human Customer explicitly approved
`PRODUCT-CAPABILITY-ROADMAP-001 v6 / r011` on 2026-08-17. v6 is retained as
historical review-blocked authority; it is not an active execution step.

Candidate `4d60e50` passed the complete v6 implementation and deterministic
gates, and independent review passed implementation SPEC and code quality. The
review still blocked closure because immutable `plan_v6.md` names a nonexistent
base object while Git, the task register, and the activation journal name the
actual base. The Human Customer explicitly approved documentation-only
`PRODUCT-CAPABILITY-ROADMAP-001 v7 / r012` on 2026-08-17. v7 starts from exact
candidate `4d60e50`, preserves v6 unchanged, and requires a fresh independent
authority/traceability PASS before Release 0 closure. Release 1 remains
inactive.

Independent v7 review passed base, ancestry, immutable-history, product,
five-path scope, hashes, links, tests, dirty exclusions, and `server/**`, but
blocked closure because r011 and r012 were both marked active in the task
register. The Human Customer explicitly approved documentation-only
`PRODUCT-CAPABILITY-ROADMAP-001 v8 / r013` on 2026-08-18. v8 marked r011 and
r012 review-blocked and established r013 as the sole pre-closure authority.
Fresh independent review returned `SPEC PASS`; r013 and Release 0 are complete.
Release 1 remained inactive with no active task at that closure.

The Human Customer directed continued approved execution through Release 1 on
2026-08-18. `PRODUCT-CAPABILITY-ROADMAP-001 v9 / r014` starts from exact Release
0 closure `6289963` and activates only `PCR-S02A — Manage tasks`. S02B and all
later stories remain inactive. Product changes are limited to the exact v9
scope, RED/GREEN order, deterministic evidence, and independent review gate.

S02A implementation candidate `9065802` passed backend checks, real race,
focused/full Core checks, frontend typechecks, exact installed browser
acceptance, and fresh independent review. Closure remained blocked because the
frozen root commands referenced nonexistent package `@multica/mobile`, while
direct Views execution exposed four deterministic verification-baseline
failures. The Human Customer approved the follow-up scope on 2026-08-18.
`PRODUCT-CAPABILITY-ROADMAP-001 v10 / r015` starts from exact `9065802`, marks
r014 verification-blocked, and authorizes only the root-script, dead-locale-key,
and Windows boundary-test repairs required to rerun the frozen gates. S02B and
later stories remain inactive.

v10 candidate `2aac82e` repaired the stale root filters and all four captured
Views failures: focused Views passed 131/131 and root typecheck passed across
the current graph. Root test then reached Desktop and exposed a transformed
shebang syntax failure plus an incomplete ignored Electron installation. The
Human Customer's continued follow-up authority activates v11/r016 from exact
`2aac82e` for only the Desktop script line and cache-only dependency recovery.
S02B and later stories remain inactive.

v11 focused Desktop checks passed 30/30 and 4/4 after the package-module and
cache-only Electron repairs. Root test then executed the Desktop suite and
showed only three real-Git integration cases timing out at the default five
seconds under full-workspace parallel load; the isolated file remains 30/30.
Continued follow-up authority activates v12/r017 from exact `3c030b4` solely to
assign those three tests a 15-second parallel-load budget without changing
assertions or product code.

v12 root execution passed 422/423 Desktop tests; one multi-step real-Git case
still exceeded the frozen 15-second ceiling under full-workspace load. v13/r018
starts from exact `8f94d69` and authorizes only raising the same three
external-process test ceilings to 60 seconds. Assertions, concurrency, and
retry behavior remain unchanged.

v13 candidate `bc1eed7` passed isolated Desktop, root typecheck/test, backend
full/race, exact clean-candidate Task Playwright acceptance, scope and dirty
blob checks, and fresh independent review. r018 and PCR-S02A are complete.
Release 1 remains active, but no task is active; PCR-S02B remains inactive until
a new versioned plan is activated.

The Human Customer's standing direction to complete Release 1 activates
`PRODUCT-CAPABILITY-ROADMAP-001 v14 / r019` from exact S02A closure `3262232`.
Only PCR-S02B Task-to-Issue promotion is active. S03A and all later stories
remain inactive.

v14 candidate `36b18b4` passed deterministic and installed-browser gates, but
fresh independent review blocked closure because replay reloaded mutable live
rows, Core callers could not reuse an idempotency key, and the nested promotion
Issue schema remained loose. Continued Customer authority activates v15/r020
from exact `36b18b4` solely for those three S02B remediations. S03A and later
stories remain inactive.

v15/r020 implemented those three remediations at exact candidate `e97c92b` and
passed focused backend/Core/Views checks plus forced root frontend typecheck and
tests. Backend full verification then reproduced an earlier attachment
concurrency failure on both `36b18b4` and `e97c92b`: Kratos's implicit
one-second HTTP request deadline cancels serialized atomic attachment writes
before the repository's bounded lock budget. No-waiver v16/r021 starts from
exact `e97c92b` and authorizes only an explicit 30-second Canonical Runtime HTTP
budget, its transport contract test, and repeated complete S02B verification.
S03A and later stories remain inactive.

v16 candidate `5ea1a47` passed the explicit deadline and unchanged 12-writer
acceptance, forced root typecheck/test, backend full/race gates, exact installed
Chrome Task-promotion acceptance, scope/process cleanup, and fresh independent
review. r021, r020, and PCR-S02B are closed. Release 1 remains active with no
active task; S03A still requires a new versioned plan.

The Human Customer's standing direction to complete Release 1 activates
`PRODUCT-CAPABILITY-ROADMAP-001 v17 / r022` from exact S02B closure `2439e9c`.
Only PCR-S03A repository-backed Issue search is active. The plan freezes
Unicode normalization, deterministic ranking/pagination, closed-state and
Workspace isolation, cancellation, an installed shared client surface, and a
feature signal that cannot prematurely enable S03B. S03B and S04 remain
inactive.

The Canonical cutover plan remains independent. The Customer confirmed C9
passed before PCR-S01A resumed; full Canonical milestone acceptance is not
inferred. This roadmap never authorizes a change below `server/**`.
