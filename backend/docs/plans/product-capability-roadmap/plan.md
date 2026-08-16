# Product capability roadmap

The approved execution snapshot is [plan_v1.md](plan_v1.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `1`
- Active step: `PCR-S01A — Canonical capability authorization and accurate feature flags`
- Status: `approved-for-execution; PCR-S01A active`
- Plan base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Active task base commit: `144997ab5fcd04544f8ffa40a1a75fc79fdb5904`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)

The user explicitly approved implementation of v1 on 2026-08-16. That approval
activates only the step named above. Later steps require the current step to
close, a frozen task, base and policy revalidation, and an updated active-step
pointer.

The Canonical cutover plan remains independent. The Customer confirmed C9
passed before PCR-S01A resumed; full Canonical milestone acceptance is not
inferred. This roadmap never authorizes a change below `server/**`.
