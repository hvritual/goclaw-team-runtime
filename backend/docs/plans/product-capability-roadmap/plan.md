# Product capability roadmap

The approved execution snapshot is [plan_v45.md](plan_v45.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `45`
- Active step: `PCR-001-S08A-R050 / R50.2`
- Status: `Release 3 complete-independent-reviewed; Release 4 S08A r049 review-blocked and r050 bounded-input correction active under independent plan PASS`
- Plan base commit: `f412279976c74df575cb8038c64f5474cfdda25d`
- Last closed task candidate: `daab0777b110ec6b21645ffe68771263d4619ec5`
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

The Human Customer's confirmed continuous Release 3 direction and confirmed
execution activate `PRODUCT-CAPABILITY-ROADMAP-001 v31 / r036` from exact S07B
closure `f5695de8` only for PCR-S07C. The frozen coverage authority is a
read-derived current/effective projection over the four traceable Requirement
sections, revision-relative Issue-link intervals, current Issue status, and the
latest Issue acceptance conclusion. Exact item stages are `unlinked`, `linked`,
`implemented`, and `accepted`; multi-Issue aggregation advances only when every
linked Issue satisfies the next stage.

v31 adds only the strict authenticated coverage GET contract, bounded
consistent SQLite read path, strict Core parsing and invalidation, shared
current/effective coverage view, four-locale labels, deterministic tests, and
fresh production installed acceptance. It adds no storage cache, migration,
permission, flag, generated contract, or Issue mutation behavior.
`project_requirements` remains true and `project_outline` remains false.
PCR-S07D, Release 3 completion, S10, original dirty paths, push, merge,
deployment, generated protobufs, and `server/**` remain inactive or excluded.

The v31 implementation passes its focused coverage suites, complete Workspace,
Core 628/628, Views 1683/1683, changed-file lint, locale, and Core/Views
typechecks. Two fresh backend `make check` executions nevertheless remain
NON-PASS on the same pre-existing Auth restart-persistence test. The exact test,
the complete Auth package, and ten consecutive exact executions pass alone. A
detached diagnostic reproduction exposes the aggregate-only hidden error as
`context deadline exceeded`: Kratos v3's incidental one-second default server
timeout expires under full package parallel load. Because the one required Auth
test path is outside v31's frozen list, r036 stops without closure and v31 stays
immutable.

The Customer's standing continuous Release 3 authority activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v32 / r037` from exact base `5d769058`. It
retains the entire S07C product contract and adds only a finite ten-second
timeout to the two in-process test-server constructor sites in the one affected
Auth test. Auth production code and behavior remain unchanged. r037 must rerun
the full backend/race/frontend/build/installed gates and obtain fresh
independent dual PASS before S07C closes. PCR-S07D, Release 3 completion, push,
merge, deployment, generated protobufs, original dirty paths, and `server/**`
remain inactive or excluded.

Commit `2f4e8015` applies the exact v32 finite test-server timeout and a fresh
complete backend `make check` passes in 383.6 seconds. The next official
seven-package race uses repository-selected Scoop GCC 15.2.0; six Workspace
packages pass, but Bootstrap exposes a real test-harness data race between the
Governance outbox reading the injected `Now` closure and the onboarding test
advancing its shared mutable clock. The official race remains NON-PASS after
439 seconds. A focused single race run passes due to scheduling, while the
focused ten-execution run reproduces the same detector stack and fails after
39.1 seconds.

Because the necessary Bootstrap test path is outside v32, r037 stops and v32
remains immutable. The Customer's standing direction activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v33 / r038` from exact base `2f4e8015`. It adds
one test-local `RWMutex` around the injected clock reads and sole time advance,
without suppressing the outbox or changing Bootstrap/Auth/Workspace production
concurrency. r038 must pass repeated focused and official race gates plus every
remaining S07C deterministic, installed, scope, and independent-review gate.
PCR-S07D, Release 3 completion, push, merge, deployment, generated protobufs,
original dirty paths, and `server/**` remain inactive or excluded.

Commit `48c95172` applies only the v33 test-local clock synchronization. The
focused Bootstrap race passes ten consecutive executions, and the unchanged
official seven-package race passes all seven packages in 299.5 seconds with
repository-selected Scoop GCC 15.2.0. A preceding official invocation remains
NON-PASS because the Windows wrapper received a transient null
`gcc -dumpfullversion` result before any test; it is not hidden or renamed.

Fresh focused S07C suites, the complete Workspace tree, backend `make check`,
Core 628/628, Views 1683/1683, Core/Views typecheck, exact changed-file lint,
root forced typecheck, and the production Web build pass. Root forced tests
remain NON-PASS only on two pre-existing Team Control five-second load
timeouts; that exact file passes 10/10 immediately in isolation. Broad Views
lint remains NON-PASS on the recorded unrelated Knowledge/Skills paths.

Fresh SQLite installed acceptance through real Canonical HTTP and production
Web proves `unlinked -> linked -> implemented -> accepted`, mixed-Issue
fail-closed aggregation, later conditional revocation, independent approval,
current/effective divergence, revision-relative unlink, deleted-Issue
exclusion, restart persistence, retirement, and reload visibility. A direct
fresh-database membership fixture was required because the Canonical
invitation route is uninstalled; it is disclosed and not counted as product
behavior. The browser recorded no framework overlay. Production `next start`
could not proxy the WebSocket upgrade and logged 403 reconnect diagnostics, so
the plan-authorized reload path supplied refresh proof; the uninstalled
invitation route also logged 404. All task-owned processes and runtime files
were removed and the three dedicated ports are closed. Exact r038 candidate
`8116b79f` has tree `d328692c`, binary patch hash `943514fc`, 32 authorized
paths, zero `server/**` or generated paths, eight continuous nine-trailer
commits, a clean isolated worktree, and zero overlap with the original 25 dirty
entries.

Fresh independent review of that exact candidate returns `SPEC BLOCK` and
`CODE/SECURITY/QUALITY BLOCK`. Persisted valid-JSON Requirement content is not
fully domain-validated on read; an active Issue link whose persisted ownership
drifts from the authorized baseline can project a foreign Issue; the shared UI
turns coverage query failure into the empty state; and the frozen constant
query graph lacks an executable query-count assertion. r038 is review-blocked
and immutable v33 remains unchanged.

The Customer's confirmed continuous Release 3 direction activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v34 / r039` from exact base `8116b79f` to close
those four findings with assertion-first failure-closed validation, same-query
link-ownership checking, a localized UI error state, and a real-driver bounded
query proof. Coverage stages, public schema, storage, permissions, flags, and
every unrelated behavior stay frozen. PCR-S07D, Release 3 completion, push,
merge, deployment, generated protobufs, original dirty paths, and `server/**`
remain inactive or excluded.

R39.2-R39.3 make the complete domain invariant authoritative on every persisted
Requirement read, reject per-snapshot active-link ownership drift, prove the
current-plus-effective repository graph remains exactly eight queries for one
and one hundred items, and render a localized safe coverage error instead of
the successful empty state. Assertion-first cases cover `{}`, invalid and
duplicate keys, oversized content, a real foreign Workspace/Project/Issue, and
real Canonical HTTP composition.

The first targeted installed run proved the coverage endpoint failed closed but
visual inspection exposed the same foreign Issue through the sibling baseline
link projection. That observable RED stayed inside v34's frozen ownership
finding and exact repository/test/composition paths. The existing single
baseline-link query now returns its stored Workspace/Project identity and
compares both with the authorized baseline before returning any link. A new
repository assertion reproduced the exact leaked identifier/title before the
repair; the repository and real HTTP composition are GREEN afterward without a
route, schema, migration, permission, flag, or query-count change.

On the repaired tree, complete Workspace and backend `make check` pass; the
official Windows race passes all packages with Scoop GCC 15.2.0. The unchanged
frontend tree retains full Core 628/628 and Views 1684/1684 PASS, Core/Views
typecheck and changed-file lint PASS, root forced typecheck 6/6 PASS, and a
17-page production Web build PASS. Broad Views lint remains NON-PASS only on
the recorded 16 errors and two warnings in unrelated Knowledge/Skills/editor/
search paths. Root forced tests remain NON-PASS on one pre-existing Team
Control aggregate-load timeout while that exact file passes 10/10.

A second fresh SQLite runtime then passes 38 targeted installed checks through
real Canonical HTTP and the production Web build: valid linked coverage renders;
corrupt content makes both raw typed failure and the safe localized UI state;
a real foreign Workspace/Project/Issue makes both baseline and coverage reads
return typed 500 with no partial or foreign detail; restoring either fixture
returns the page to valid linked coverage. The in-app browser again cannot
reach host loopback, so installed Chrome supplies the allowed fallback. The
known WebSocket-upgrade 403 and uninstalled-invitations 404 remain disclosed,
and the three dedicated ports are closed.

Exact r039 candidate `47ee4189cb5571ec38ae39480c758d4decad22bd`
has tree `d0b7d56b65964e1559e3bbe33aa734f70e2f8eca`, binary patch hash
`6d58ab21a2b39a9328c457e00d0a8ea6ffef7e1493e11af8cdc29e5b178ada34`,
15 authorized paths, zero `server/**` or generated paths, four complete
continuous nine-trailer commits, a clean isolated worktree, and zero overlap
with the original 25 dirty entries. v31-v34 and policy hashes match, all owned
processes are stopped, and ports 3018/38139/39139 are closed. Fresh independent
review returns `SPEC PASS` and `CODE/SECURITY/QUALITY PASS` with no blocking
finding. PCR-S07C is `complete-independent-reviewed`; PCR-S07D remains inactive
until its own immutable successor plan is activated. No push, merge,
deployment, generated-protobuf change, or Release 3 completion is claimed.

The Customer's confirmed continuous Release 3 direction and confirmed
execution activate only `PRODUCT-CAPABILITY-ROADMAP-001 v35 / r040` from exact
S07C closure `1d515efc` for PCR-S07D. The frozen Retrospective authority is
Workspace-owned, versioned draft/publication/supersede/archive content with
revisioned participant roles, current facilitator/Project-lead authorization,
structured action items, default Task and explicit Issue targets, resumable
idempotent cross-contract creation, and immutable provenance links. The
Retrospective repository never writes Task or Issue tables; published snapshots
are immutable and every mutation revalidates current membership/Project
authority after `BEGIN IMMEDIATE`.

v35 completes the strict HTTP/Core/shared-Views surface, four locales, the
`project_retrospectives` installed flag, deterministic migration/governance/
interruption/concurrency evidence, production Web build, and fresh
two-identity installed acceptance. It adds no automatic Knowledge proposal,
public Issue/proto change, generated code, realtime Retrospective projection,
target unlink/re-target, or permanent delete. Release 3 completion remains
inactive until S07D passes its own exact candidate and fresh independent dual
review, followed by a separate aggregate DoneGate. Activation claims no
implementation or PASS.

Assertion-first R40.2 dependency inspection then proved that the installed
Canonical Workspace has two Project deletion repositories but v35 authorized
only the HTTP-surface repository path. Cleaning Retrospective authority from
one deletion entry while leaving it behind through the installed local/proto
entry would violate the frozen Project-deletion contract. No S07D product
candidate was committed under r040.

The same confirmed continuous authority activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v36 / r041` from exact governance activation
`fbfc1c05`. v36 incorporates every v35 contract, gate, exclusion, and stop
condition unchanged and adds exactly
`backend/internal/modules/workspace/internal/infrastructure/sqlite/project_repository.go`
to the writable boundary so both installed Project-deletion transactions can
reuse one scoped Retrospective cleanup. Target Tasks/Issues and immutable
Retrospective audit/outbox evidence remain retained. Release 3 completion,
push, merge, deployment, generated code, and `server/**` remain excluded.

The complete Workspace graph then exposed exactly two stale migration-count
assertions after installing migration 000020. One assertion path was already
authorized; `sqlite_workspace_services_test.go` was not. Exact repository-wide
search found no third count-19 dependency, and no S07D product candidate was
committed under r041.

Continued confirmed authority therefore activates only
`PRODUCT-CAPABILITY-ROADMAP-001 v37 / r042` from exact governance activation
`24d3043b`. It inherits every v35-v36 product requirement and adds only
`backend/internal/modules/workspace/sqlite_workspace_services_test.go` so the
two installed integration assertions can state the correct catalog count 20.
No production behavior, route, schema, or exclusion changes.

R42.6 exact scope audit then found that immutable v35 spelled one existing Core
test path as `mutations.test.ts` although the repository and 45-path candidate
use `mutations.test.tsx`. Zero `server/**`, generated, or additional product
paths changed, but an allowed-path mismatch cannot be waived. r042 and candidate
`f3a77a6c` are therefore scope-blocked before independent review and retained
only as provenance. The same confirmed authority activates
`PRODUCT-CAPABILITY-ROADMAP-001 v38 / r043` from the last valid governance-only
activation `28e7a56c`; v38 makes only that one-for-one path correction and
requires the byte-identical product patch, fresh gates, and fresh independent
dual review after authorization.

Exact r043 candidate `64091302b703a4590bdbe88d154f65fec9d6b37c`
has tree `e696d67ad72aad52bc53e4a6bfe3211aac2f89d7`. Its 45-path
product patch is byte-identical to blocked r042 at SHA-256
`a6ebeda199614944fb254d8a5c184cb2d34dc7790c511bae6cd657f6772bcfef`;
the complete diff from exact base `28e7a56c` contains five governance plus 45
product paths and hashes to
`b0ac56383905ace9581f5c575e2e7321116253cc42b03a18dcc1661d1add53fe`.
It contains zero `server/**` and generated paths, all five r043 commits carry
one continuous ordered nine-field trailer block, the isolated candidate is
clean, and it overlaps none of the original worktree's 25 dirty entries.

Fresh r043 focused and complete Workspace checks, backend `make check`, the
full redirected official race rerun, Core 635/635, Views 1688/1688,
Core/Views and forced-root typechecks, exact changed-file lint, production Web
build, and production-Web two-identity installed lifecycle/restart/archive
acceptance pass. The first race run failed for disclosed C-drive temporary-disk
pressure; broad Views lint retains 16 unrelated errors and two warnings, and
forced root tests retain three aggregate five-second load timeouts while the
two exact files pass 44/44. None is converted to PASS or waived.

Fresh independent read-only review of the exact candidate returns `SPEC PASS`
and `CODE/SECURITY/QUALITY PASS`. PCR-S07D is
`complete-independent-reviewed`; all four Release 3 stories are now closed,
but Release 3 itself remains active until a separately frozen aggregate
DoneGate passes. Push, merge, deployment, generated protobufs, original dirty
paths, and `server/**` remain excluded.

The same confirmed continuous completion direction activates governance-only
`PRODUCT-CAPABILITY-ROADMAP-001 v39 / r044` from exact S07D closure
`8150a0e5`. It freezes the four reviewed story candidate/tree/closure tuples,
permits only six roadmap-document paths, and authorizes no product/test/schema/
runtime/frontend byte. r044 must freshly audit the complete Release 3 lineage,
reconcile only the three Release 3 capability-matrix rows, rerun complete
deterministic and aggregate installed coexistence gates, close all owned
artifacts, and obtain fresh independent `SPEC PASS` plus
`CODE/SECURITY/QUALITY PASS`. Activation claims no aggregate PASS or Release 3
completion; Release 4, push, merge, deployment, generated protobufs, and
`server/**` remain inactive or excluded.

R44.2 proves the four frozen tuples and all registered v26-v39 hashes, finds 114
product plus 17 roadmap paths with zero `server/**`, generated, or r044 product
drift, then stops on v39's incorrect uniform-nine-field history assertion.
Initial Release 3 activation `71afb3c3` has the exact continuous eight fields
required by v26/r031; `Issue` was not present in that frozen authority, and
backend policy requires it only when present. All later range commits have nine
fields. Immutable v39 cannot be relaxed, so r044 is
`audit-blocked-before-capability-matrix-write` without product or matrix change.
The same confirmed authority activates v40/r045 from `a1f6a934`, preserving
every v39 gate while correcting only that historical audit predicate. No
aggregate PASS or Release 3 completion is claimed by the successor activation.

R45.2 then passes the policy-correct audit across 43 commits: exact S07A-D
candidate/tree/closure tuples, registered v26-v40 hashes, one frozen historical
eight-field block followed by 42 nine-field blocks, 114 product plus 17 roadmap
paths, zero `server/**`/generated/original-dirty overlap, and zero r045 product
drift. Exact Release 3 closure tree is `2caa262b`; its full and product-only
binary patch SHA-256 values are respectively
`34d5b356e382f037fdddac2371bb467945c50fee74ab9ea5f5fd8d9d08fc56a2`
and `0a9e59e2aab7b23d3c6dad4b69c54df3f0c1ea702b7d362838b3c7dcc7668aa0`
under the recorded UTF-8 line-normalized method. Only the three Release 3
capability-matrix rows are reconciled; complete deterministic and installed
aggregate gates remain pending, so Release 3 is not yet complete.

R45.3 deterministic execution passes the separate complete Workspace graph in
123.4 seconds, backend `make check` in 557.8 seconds, and unchanged full
repository `make test-race` in 546 seconds with GCC 15.2.0 and F-drive temporary
caches. Core passes 103 files/635 tests; Views passes 169 files/1688 tests;
Core/Views typechecks, Core broad lint, exact 23 Core plus 11 Views Release 3
file lint, forced root typecheck 6/6 with zero cache, and the production Web
build with 17/17 static pages all pass. The build's generated `next-env.d.ts`
line is restored to its exact pre-build hash and the candidate is clean.

Retained NON-PASS evidence is not relabeled: broad Views lint reports 16 errors
and two warnings in unrelated editor/Knowledge/search/Skills paths; forced root
tests finish 4/5 tasks and Views 1686/1688 because Login and Team Control each
hit the five-second aggregate timeout, while those exact two files pass 44/44
and the standalone complete Views run passes 1688/1688. Existing jsdom canvas,
navigation, React `act`, and i18n warnings remain visible. The created
747,659,709-byte F-drive gate cache has no process/listener but remains outside
the repository after two host-policy deletion rejections; no alternate deletion
mechanism is used. Aggregate installed acceptance and independent review remain
pending, so Release 3 is not yet complete.

R45.4 combined installed acceptance passes on one fresh Canonical SQLite
database, one Project, two independently authenticated identities, the real
HTTP backend, and the production Next 16.2.6 Web build. Project
`d48a22fa-ce08-4638-8fce-4be579106181` retains Resources
`80bd9b8b-f8e6-488f-b588-d1c4a34fc490` and
`2e25dd29-7e4d-42f7-b6fe-2b7f792fb33b` after create, reorder, archive, and
restore at set revision 5. Requirement baseline
`f0ee857a-1bf6-479c-b8b5-fa50634aa614` reaches frozen revision 13 with 13
immutable history entries, independent approval/re-approval, one authorized
root-outline link, four Issue links, and observed 4/4 linked, implemented, then
accepted current/effective coverage. Full `project_outline` stays false while
the confirmed root prerequisite remains installed under
`project_requirements=true`.

The same Project denies ordinary-member facilitator self-appointment with 403,
then permits the current Project lead to publish, supersede, target, and archive
Retrospective `b2769a21-e682-4b83-aa9c-c052bdf0e308`. Its revision-4 history
retains default Task `5334d009-43d3-4ae3-8142-76df99a08b08` and explicit Issue
`7048a2af-986e-4417-8203-91f1515d4442`, both immutably linked to source revision
3; repeated Task-target requests with the same key are byte-identical.
Resources, all 13 Requirement revisions, accepted coverage, all four
Retrospective revisions, both links, and both targets survive a real backend
restart. Unknown Project reads return 404 across all three verticals; a foreign
Workspace header returns 404 without disclosing the Project ID.

A clean authenticated in-app Browser tab renders both Resources, the archived
Retrospective with Task/Issue links and four-entry history, frozen Requirement
v13, four accepted current/effective coverage items, the root-outline link, and
all 13 Requirement history entries. It has zero Next overlay and zero console
error. Known production WebSocket reconnect and uninstalled-invitations 404
warnings remain disclosed. All browser tabs, owned processes, and ports
3021/38142/39142 are closed. The validated 353,616,921-byte acceptance directory
and the prior 747,659,709-byte gate cache remain under the task-owned F-drive
root because exact PowerShell removal requests were rejected by host policy
before execution; no alternate deletion mechanism is used. Exact-candidate
audit and fresh independent dual review remain pending, so Release 3 is not yet
complete.

R45.5 freezes exact documentation-only candidate `a63bf58a` with tree
`aba1d1c1` and r045 binary patch SHA-256
`f9322809dba511553c127c490705c6438943ad6466c631ff504ae2bc23b9260b`.
Its six roadmap paths, zero product drift, zero `server/**`/generated/original-
dirty overlap, clean worktree, and closed processes/ports pass independent
recalculation. Fresh review returns `CODE/SECURITY/QUALITY PASS` but
`SPEC BLOCK`: historical commit `9fb86ea0` separates each of its nine correct
v27/r032 fields with a blank line, so it is not the continuous nine-field block
claimed by v40; additionally, v39's required external-directory removal did
not execute because host policy rejected the exact requests. r045 stops before
closure and its audit PASS is not retained.

The confirmed continuous completion direction activates governance-only
`PRODUCT-CAPABILITY-ROADMAP-001 v41 / r046` from exact blocked candidate
`a63bf58a`. It freezes three exact historical trailer shapes, permits the
host-policy-retained F-drive directory only as a fully inventoried, zero-live-
resource disposition subject to fresh independent acceptance, and leaves all
product bytes plus `capability-matrix.md` read-only. No Release 3 completion or
review PASS is claimed by successor activation.

R46.2 raw-message-aware audit then passes exact activation candidate
`b9719f73` (tree `d1820775`, r046 patch SHA-256
`1e3044468c1c2ee34ec80078c2e9da52e575e9beb597ae04f5585c4a2cd8a337`).
Across 47 Release 3 commits it proves one continuous eight-field message, one
exact blank-separated nine-field message, and 45 continuous nine-field
messages with registered task/plan/policy values. The range has 135 paths: 114
product plus 21 roadmap; product patch SHA-256 remains
`0a9e59e2aab7b23d3c6dad4b69c54df3f0c1ea702b7d362838b3c7dcc7668aa0`,
with zero `server/**`, generated, product-drift, or original-dirty overlap.
Ports and processes remain closed. The exact retained root now has 11,837
files, 527 directories, and 1,101,304,235 bytes and is explicitly not deleted.
Exact final candidate and fresh independent dual review remain pending, so
Release 3 is not complete.

R46.3 freezes exact candidate `daab0777b110ec6b21645ffe68771263d4619ec5`
with tree `b012a2c1aa7cc6dae4e016206585e51102e2a69b` and r046 binary
patch SHA-256
`528aa873dea5d477be4ddbdef956b643d393204f6811bd19da08c02f5689d482`.
Its strict audit passes 48 commits as one continuous eight-field, one exact
blank-separated nine-field, and 46 continuous nine-field messages; five r046
roadmap paths, 114 unchanged product paths, all plan/policy hashes, zero
forbidden/generated/drift/dirty overlap, clean candidate, and closed ports/
processes pass. The retained external root is independently inventoried at
11,837 files, 527 directories, and 1,101,307,455 bytes and remains explicitly
not deleted under v41's bounded terminal disposition.

Fresh independent review of that exact candidate returns `SPEC PASS` and
`CODE/SECURITY/QUALITY PASS`. It confirms the unique `9fb86ea0` historical
shape against v27 authority, accepts the exact host-policy-retained disposition
without creating a general waiver, and confirms unchanged R45.3/R45.4 product
evidence plus all retained NON-PASS disclosures. R46.4 closes r046 and Release
3 as `complete-independent-reviewed` with zero active tasks. Release 4 remains
inactive; no push, merge, deployment, generated protobuf, original-dirty, or
`server/**` action is claimed.

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
