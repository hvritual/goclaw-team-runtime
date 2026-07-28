---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Recovered baseline complete and handed to MVP browser gate
revision: 6
supersedes: plan-r005.md
plan_status: approved
wave_state: complete
approved_by:
  - user-directive-2026-07-28
owner: Codex root agent
reviewers:
  - recovery_code_review
  - recovery_security_review
  - recovery_docs_review
depends_on: []
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - MVP-W00-S09
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
product_code_changes_allowed: false
---

# MVP-W00 r006 — 恢复基线完成并移交 MVP 浏览器门禁

r005 的确定性 Gate 与三路 final review 已通过。review-writeback commit
`bf36ed343ca213d1df0a32ffa0e5184063b1fd58` 连续构建两次，第二次明确为
`Verified identical existing release`；annotated tag
`v0.8.0-pilot.1-recovered.1` 指向同一 commit，release manifest 的
commit/tree 与 tag target 完全一致。

## 最终恢复身份

| 对象 | 值 |
|---|---|
| 原始 source archive SHA-256 | `cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f` |
| import commit/tree | `e4783a4f2bc7a6ce8df1405787c44ed636b195d3` / `38f798c2a652eaf99d5ad1ca145e50c176ee4c58` |
| recovered tag | `v0.8.0-pilot.1-recovered.1` |
| tag target commit/tree | `bf36ed343ca213d1df0a32ffa0e5184063b1fd58` / `bce6dc94a2b7bfd57edb3848e4b6833786f62ac9` |
| final manifest SHA-256 | `a12e7e7497b8e217212f8bc04124f1c5371792364b1712b91d0e13e76e7353d9` |
| final source archive SHA-256 | `924c6bbddde355c035016bef24120ae0b6fe24644f3b64941f0aefd44b5b1197` |

## 完成门禁

- [x] import 611/611，内容、执行位、extra 均为 0；
- [x] Policy、五个历史前缀、archive negative、Go、Web、Obsidian 全通过；
- [x] code/security/docs 三路 final review P0=0/P1=0；
- [x] final clean commit 双构建逐字节一致；
- [x] manifest commit/tree 等于 recovered tag target；
- [x] final checksum 和 clean worktree 通过；
- [x] 旧原版命名 ignored rebuild 已隔离，不作为交付入口。

## 移交

Recovery 只证明源码、构建、发布与治理基线可追溯，不证明真实 Codex OAuth、
飞书、ptrace、浏览器、bwrap、WSL2/Lima 或三台电脑 Gate。后继唯一 active
Wave 为 `FE-W01 r012`，从 recovered tag 后的治理 transition commit 冻结
MVP 浏览器 Task。
