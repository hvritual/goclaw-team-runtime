---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Three-person controlled pilot
revision: 4
supersedes: plan-r003.md
plan_status: approved
wave_state: blocked
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - runner_xplat_impl
  - pilot_governance_impl
  - pilot_frontend_impl
depends_on:
  - FE-W00
created_at: 2026-07-27
updated_at: 2026-07-28
steps:
  - PILOT-W00-S01
  - PILOT-W00-S02
  - PILOT-W00-S03
  - PILOT-W00-S04
  - PILOT-W00-S05
  - PILOT-W00-S06
  - PILOT-W00-S07
allowed_change_scope:
  - docs/**
product_code_changes_allowed: false
---

# PILOT-W00 r004 — 等待权威恢复基线

本修订继承 [`plan-r003`](plan-r003.md) 的目标、已完成实现和未完成实机
Gate。唯一实质变化是：

- `PILOT-W00` 暂停为 `blocked`；
- 在 `MVP-W00` 完成并生成唯一恢复 base commit 前，不接受新的产品改动、
  Task freeze、enqueue 或试点完成声明；
- 恢复完成后必须创建新的 Plan revision，把历史 Wave/Evidence 重新绑定到
  新 Git base，而不是继续引用无法解析的旧 commit。

暂停不会把真实 Codex OAuth、飞书、浏览器、bwrap、WSL2 或 Lima Gate
改写为通过。
