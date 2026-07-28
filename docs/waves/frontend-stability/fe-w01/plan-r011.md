---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Wait for authoritative recovery base
revision: 11
supersedes:
  - plan-r010.md
plan_status: approved
wave_state: blocked
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - recovery_docs_review
depends_on:
  - FE-W00
  - MVP-W00
created_at: 2026-07-26
updated_at: 2026-07-28
steps:
  - FE-W01-S08
  - FE-W01-S10
  - FE-W01-S11
allowed_change_scope:
  - docs/waves/**
product_code_changes_allowed: false
---

# FE-W01 r011 — 等待恢复基线并更正批准归因

本修订完整继承 r010 的暂停合同，只移除把 BLOCK reviewer 当作批准者的错误
归因。用户指令是当前 revision 的批准来源，`recovery_docs_review` 仍是独立
reviewer。

在 `MVP-W00` 完成前，FE-W01 继续 `blocked`、docs-only，不能启动 runtime、
浏览器或任何产品变更。恢复完成后必须创建新 revision，绑定 recovered
base、credential owner 和本地 Playwright Gate。
