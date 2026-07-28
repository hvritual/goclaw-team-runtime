---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Wait for authoritative recovery base
revision: 10
supersedes:
  - plan-r009.md
plan_status: approved
wave_state: blocked
approved_by:
  - user-directive-2026-07-28
  - recovery_docs_review
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

# FE-W01 r010 — 等待权威恢复基线

本修订保留 [`plan-r009`](plan-r009.md) 的历史实现、外部阻断和验收合同。
唯一变化是将 Wave 明确暂停为 `blocked`，并机器化依赖 `MVP-W00`。

在 `MVP-W00` 完成前：

- 不允许继续产品代码、synthetic runtime 或 Browser 执行；
- 不把历史不可解析 commit 绑定到新 Task；
- `FE-EVID-W01-011/015/018` 保持原状态；
- 恢复完成后必须创建新 revision，把仍有效的范围、Task base、Plan SHA 和
  Evidence 重新绑定到 recovered Git base。

本修订不声称 r009 已完成，也不降低 credential owner、ptrace 或真实
Playwright Gate。
