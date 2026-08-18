# Product capability roadmap

The approved execution snapshot is [plan_v21.md](plan_v21.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `21`
- Active step: `PCR-S05B Skill import and logical files`
- Status: `Release 2 active; PCR-S05B r026 active`
- Plan base commit: `27def068f07cb56ef7e58471fbfacb397b11b639`
- Last closed task candidate: `17f7eb1f2db22bd64e426614b84b945325b1f90f`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)
- S01B foundation design: [s01b-foundation-design.md](s01b-foundation-design.md)

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
