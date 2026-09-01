# IoT Engineering Digital Thread

Plan-ID: `IOT-ENGINEERING-DIGITAL-THREAD-001`

Current approved version: [plan_v4.md](./plan_v4.md)

Status: **Approved / Phase 2 certified; Phase 3 P3-S01 active**

Original base commit: `d3bbafb071dc493bd17d5c0387297bbf38da9ecb`

Phase 2 certified canonical: `f590d56fe5cc019489ac61ac378f1fffba55ee50`

The user explicitly authorized Phase 1 on 2026-08-28, Phase 1 wrap-up plus Phase 2 execution on 2026-08-28, and direct entry into the next phase on 2026-09-01.

Validated through: **Phase 2 E2E Exit Certification** — CI run `33519792124` on the final canonical product.

Active product step: `P3-S01 — Normalized Evidence Envelope`.

P3-S01 extends the Engineering Thread with provider-neutral immutable evidence projections while preserving Runtime as the authority for Run/Attempt/Lease/Retry and Runner execution Evidence. It must not create a second execution truth source or mutate Change acceptance, DoneGate, or governed Knowledge publication.

Current integration governance warning: GitHub currently reports no active repository ruleset for the canonical branch. Phase 3 implementation may proceed on its isolated branch, but canonical integration requires the independent-review/required-CI protection gate to be re-established or explicitly enforced before merge.

Product-code changes must reference Plan-ID `IOT-ENGINEERING-DIGITAL-THREAD-001`, version `4`, step `P3-S01` until that step is accepted.

Progress and verification evidence are append-only in [journal.md](./journal.md).
