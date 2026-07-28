---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Wait for recovered base and frontend gate
revision: 5
supersedes: plan-r004.md
plan_status: approved
wave_state: blocked
approved_by:
  - user-directive-2026-07-28
  - recovery_code_review
  - recovery_docs_review
owner: Codex root agent
reviewers:
  - recovery_docs_review
depends_on:
  - FE-W00
  - MVP-W00
  - FE-W01
created_at: 2026-07-27
updated_at: 2026-07-28
steps:
  - PILOT-W00-S06
  - PILOT-W00-S07
allowed_change_scope:
  - docs/**
product_code_changes_allowed: false
---

# PILOT-W00 r005 — 等待 recovered base 与 FE-W01

本修订继承 [`plan-r004`](plan-r004.md) 的暂停决定，并关闭其机器字段缺口：

- `depends_on` 显式加入 `MVP-W00` 和 `FE-W01`；
- Registry 与本计划统一为 `docs/**`、`product_code_changes_allowed=false`；
- 即使依赖完成，本 revision 也不授权产品变更或实机执行；
- 重新启动三人试点必须另建 Plan revision，绑定 recovered base、三名 owner、
  三台设备和本次仍未完成的 Evidence。

真实 Codex OAuth、飞书、浏览器、bwrap、WSL2、Lima、credential owner、
三 Runner 并发和冷备恢复 Gate 继续保持未完成。
