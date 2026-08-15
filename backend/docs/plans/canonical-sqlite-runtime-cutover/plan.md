# Canonical SQLite runtime cutover plan

The approved execution snapshot is [plan_v10.md](plan_v10.md).

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Approved version: `10`
- Active step: `none — C9 blocked pending approved plan_v11`
- Status: `blocked; C7 and C8 Customer Accepted; the v10 trace repair exposed an authenticated clean-candidate product failure (two GET /api/issues responses returned 500 during the C9 detail journey). Product repair requires a new approved plan; no Customer milestone acceptance may be requested or inferred.`
- Milestone: [milestone.md](milestone.md)
- Story map: [story-map.md](story-map.md)
- Parity matrix: [parity-matrix.md](parity-matrix.md)
- Evidence journal: [journal.md](journal.md)

Only the active step is authorized. The milestone goal does not authorize work
from a later step, changes below `server/**`, or changes to an already approved
plan in another plan directory.
