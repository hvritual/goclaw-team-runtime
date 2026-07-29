---
schema: goclaw.wave/v1
wave_id: RN-W01
track_id: TEAM-RUNTIME-2026-07
title: Runner lifecycle, local execution, and version management
revision: 1
plan_status: approved
wave_state: planned
approved_by:
  - user-directive-2026-07-28-auto-wave
owner: Codex root agent
depends_on:
  - TC-W01
created_at: 2026-07-28
updated_at: 2026-07-28
product_code_changes_allowed: true
---

# RN-W01 r001

在 TC-W01 完成后实现 Runner release pin、兼容性协商、下载校验、原子自更新/
回滚、多项目并发配置、目录限定执行和签名 Evidence 增强。激活前将创建新的
范围精确 revision 和 Task Freeze。
