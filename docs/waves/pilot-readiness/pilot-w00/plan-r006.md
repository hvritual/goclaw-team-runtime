---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Wait for recovered base and frontend gate
revision: 6
supersedes: plan-r005.md
plan_status: approved
wave_state: blocked
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - recovery_code_review
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

# PILOT-W00 r006 — 等待 recovered base 与 FE-W01

本修订完整继承 r005 的 blocked 状态、依赖、范围和现场 Gate，只更正当前
revision 的批准归因。第一轮 code/docs review 是 BLOCK finding 来源，不是
r005 的批准者；当前 revision 由用户指令授权。

重新启动三人试点仍必须新建 Plan revision，并绑定 recovered base、三名
owner、三台设备、真实 Codex OAuth、飞书、浏览器/Obsidian Desktop、
bwrap/WSL2/Lima 和 credential owner Evidence。
