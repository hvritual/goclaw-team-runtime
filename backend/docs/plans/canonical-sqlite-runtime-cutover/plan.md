# Canonical SQLite runtime cutover plan

The approved execution snapshot is [plan_v11.md](plan_v11.md).

- Plan-ID: `canonical-sqlite-runtime-cutover`
- Approved version: `11`
- Active step: `none — v11 integration blocked at clean-candidate offline dependency setup`
- Status: `blocked; C7 and C8 Customer Accepted; C9 Issue-list RED/GREEN and Backend verification passed, but the single frozen offline install timed out before TypeScript/browser gates; no retry or alternate source was attempted; C9 and milestone acceptance remain pending.`
- Milestone: [milestone.md](milestone.md)
- Story map: [story-map.md](story-map.md)
- Parity matrix: [parity-matrix.md](parity-matrix.md)
- Evidence journal: [journal.md](journal.md)

Only the active step is authorized. The milestone goal does not authorize work
from a later step, changes below `server/**`, or changes to an already approved
plan in another plan directory.
