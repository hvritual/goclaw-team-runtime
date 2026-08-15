# Canonical SQLite runtime cutover plan

The approved execution snapshot is [plan_v10.md](plan_v10.md).

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Approved version: `10`
- Active step: `M1-S7-C9-INTEGRATE-GREEN`
- Status: `active; C7 and C8 Customer Accepted; the v10 E2E-only trace repair has RED/GREEN evidence. A new clean candidate must still complete deterministic, Chrome, restart, rollback and independent review gates; no Customer milestone acceptance may be requested or inferred.`
- Milestone: [milestone.md](milestone.md)
- Story map: [story-map.md](story-map.md)
- Parity matrix: [parity-matrix.md](parity-matrix.md)
- Evidence journal: [journal.md](journal.md)

Only the active step is authorized. The milestone goal does not authorize work
from a later step, changes below `server/**`, or changes to an already approved
plan in another plan directory.
