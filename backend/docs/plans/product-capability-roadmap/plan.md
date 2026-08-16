# Product capability roadmap

The approved execution snapshot is [plan_v1.md](plan_v1.md).
The proposed successor is [plan_v2.md](plan_v2.md); it is not executable until
the Human Customer explicitly approves version 2.

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `1`
- Active step: `PCR-S01B — Revision, audit, idempotency, and outbox`
- Status: `approved-for-execution; PCR-S01B design task active`
- Plan base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Active task base commit: `cc61297be42ca5acf1fc47d9ba9d70939f406588`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)
- S01B foundation design: [s01b-foundation-design.md](s01b-foundation-design.md)

The user explicitly accepted S01A and its three documented gate waivers, then
confirmed activation of S01B on 2026-08-16. S01B begins with a documentation-only
contract and migration-design task. Product code remains unauthorized until a
new plan version resolves the SQLite migration-policy conflict and is approved.
Later steps still require the current step to close, a frozen task, base and
policy revalidation, and an updated active-step pointer.

The Canonical cutover plan remains independent. The Customer confirmed C9
passed before PCR-S01A resumed; full Canonical milestone acceptance is not
inferred. This roadmap never authorizes a change below `server/**`.
