---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Three-person controlled pilot
revision: 2
supersedes: plan-r001.md
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-27
  - pilot_frontend_impl
  - Codex root agent
owner: Codex root agent
reviewers:
  - runner_xplat_impl
  - pilot_governance_impl
  - pilot_frontend_impl
depends_on:
  - FE-W00
created_at: 2026-07-27
updated_at: 2026-07-27
steps:
  - PILOT-W00-S01
  - PILOT-W00-S02
  - PILOT-W00-S03
  - PILOT-W00-S04
  - PILOT-W00-S05
  - PILOT-W00-S06
  - PILOT-W00-S07
allowed_change_scope:
  - AGENTS.md
  - README.md
  - CHANGELOG.md
  - CHANGE-HANDOFF.md
  - Makefile
  - .goreleaser.yaml
  - config.json.example
  - config/**
  - cli/**
  - gateway/**
  - internal/start/**
  - orchestratorlite/**
  - workstation/**
  - teamcontrol/**
  - ouroboros/**
  - agent/manager.go
  - session/**
  - ui/package.json
  - ui/package-lock.json
  - ui/src/team/**
  - ui/tests/**
  - scripts/**
  - deploy/**
  - docs/**
product_code_changes_allowed: true
---

# PILOT-W00 r002 — 无碰撞项目会话键

本修订完整继承 [`plan-r001`](plan-r001.md) 的目标、范围、步骤、门禁、风险和
退出条件，只增加一个在 S05 实现期间经确定性反例确认的 S0 隔离修复。

## 新事实

旧的 `project:<project>:<topic>` 会话键在文件层把 `:` 替换为 `_`。因此，
例如 `(project=alpha, topic=beta_gamma)` 与
`(project=alpha_beta, topic=gamma)` 会映射到相同文件。这会令项目聊天历史
在合法标识组合下发生碰撞，违反三人试点的项目隔离门禁。

## 变更

- 登记 `PILOT-ISSUE-013`；
- 在 `session` 提供一个带版本、按段无歧义编码的权威项目会话键；
- `agent` 写入路径与 Gateway `chat.history` 读取路径必须共用该函数；
- 对旧键执行“无歧义才迁移”：目标不存在且旧键只可能对应当前项目/Topic
  时原子迁移；无法证明唯一时拒绝读取并要求管理员显式处理；
- 增加碰撞反例、旧数据迁移、跨项目拒绝和并发读取测试；
- 允许修改 `agent/manager.go`，其余 allowlist 与 r001 相同。

## 验证

`PILOT-EVID-005` 只有在以下条件同时通过时才可标记 verified：

1. 上述碰撞对生成不同的新键和文件路径；
2. agent 与 history RPC 使用完全相同的 key builder；
3. 未经授权的项目仍在访问存储前被拒绝；
4. 模糊旧键失败关闭，不静默猜测归属；
5. UI transport/state tests、session/agent/gateway Go tests 与 Browser 三上下文
   场景通过。

## 决策

`PILOT-DEC-005`：聊天历史隔离使用版本化的无歧义键，并只做可证明安全的
旧键迁移。兼容性不能优先于跨项目数据隔离。

