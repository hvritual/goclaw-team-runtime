# Canonical SQLite runtime cutover plan

The approved execution snapshot is [plan_v10.md](plan_v10.md).

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Approved version: `10`
- Active step: `M1-S7-C9-INTEGRATE-RED`
- Status: `active; C7 and C8 Customer Accepted; v10 review correction retains one P1 evidence gap (incomplete HTTP :8080 trace coverage). The approved plan already permits its E2E/trace-only repair; no Customer milestone acceptance may be requested or inferred.`
- Milestone: [milestone.md](milestone.md)
- Story map: [story-map.md](story-map.md)
- Parity matrix: [parity-matrix.md](parity-matrix.md)
- Evidence journal: [journal.md](journal.md)

Only the active step is authorized. The milestone goal does not authorize work
from a later step, changes below `server/**`, or changes to an already approved
plan in another plan directory.
