# Product capability roadmap

The approved execution snapshot is [plan_v4.md](plan_v4.md).

- Plan-ID: `PRODUCT-CAPABILITY-ROADMAP-001`
- Approved version: `4`
- Active step: `none; PCR-S01B-4 independent-review-blocked`
- Status: `Release 0 incomplete`
- Plan base commit: `45213820fade7f61294d2287e063bf19fbd015ee`
- Active task base commit: `ab2b49088b108f771045a090b473a8e235dfa09e`
- Capability baseline: [capability-matrix.md](capability-matrix.md)
- Frozen product contracts: [contract-freeze_v1.md](contract-freeze_v1.md)
- Ordered delivery stories: [story-map.md](story-map.md)
- Task register: [task-register.md](task-register.md)
- Evidence journal: [journal.md](journal.md)
- S01B foundation design: [s01b-foundation-design.md](s01b-foundation-design.md)

The Human Customer accepted the v3/r008 repair, then explicitly approved the
displayed v4/r009 follow-up actions and directed continuous progress to complete
Release 0 on 2026-08-17. Candidate `5062e84` passed every frozen deterministic
gate, including the repaired Windows race checks. The independent read-only
review returned `SPEC BLOCK` for unresolved frozen-contract violations in
authority mapping, canonical request hashing, secret-free envelopes, outbox
lease/tuple safety, and empty-only down migration. Release 0 is not complete.
No product repair may begin until a new plan version and task are approved;
Release 1 remains inactive.

The Canonical cutover plan remains independent. The Customer confirmed C9
passed before PCR-S01A resumed; full Canonical milestone acceptance is not
inferred. This roadmap never authorizes a change below `server/**`.
