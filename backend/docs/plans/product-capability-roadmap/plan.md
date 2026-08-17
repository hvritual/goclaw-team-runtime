# Product capability roadmap

The approved execution snapshot is [plan_v11.md](plan_v11.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `11`
- Active step: `PCR-S02A Desktop verification repair`
- Status: `Release 1 active; PCR-S02A/r016 Desktop verification repair authorized`
- Plan base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Active task base commit: `2aac82e742932ddcda2ab6fa1387a9960a530974`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)
- S01B foundation design: [s01b-foundation-design.md](s01b-foundation-design.md)

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

The Canonical cutover plan remains independent. The Customer confirmed C9
passed before PCR-S01A resumed; full Canonical milestone acceptance is not
inferred. This roadmap never authorizes a change below `server/**`.
