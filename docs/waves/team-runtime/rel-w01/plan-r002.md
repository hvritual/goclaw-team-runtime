---
schema: goclaw.wave/v1
wave_id: REL-W01
track_id: TEAM-RUNTIME-2026-07
title: Cross-platform packaging, operations, and pilot release
revision: 2
plan_status: approved
wave_state: planned
approved_by:
  - user-directive-2026-07-29-team-control-authority-replan
owner: Codex root agent
depends_on:
  - TC-W06
supersedes:
  - docs/waves/team-runtime/rel-w01/plan-r001.md
created_at: 2026-07-29
updated_at: 2026-07-29
product_code_changes_allowed: false
---

# REL-W01 r002 — Team Control convergence successor route

r002 不改写 r001 的发行目标，只把已被替代的 INT-W01 依赖前向调整为
`TC-W06`。TC-W03–TC-W06 依次关闭知识权威、Policy/Context、MCP/Evidence
和运维 cutover 后，REL-W01 才能创建新的范围精确 revision。

本 revision 只是路线调整，`product_code_changes_allowed=false`。激活前
必须再创建 approved plan revision，冻结 release Task，并重新确认真实
Pilot、凭据、跨平台和回滚 Gate。
