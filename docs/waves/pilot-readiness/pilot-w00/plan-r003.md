---
schema: goclaw.wave/v1
wave_id: PILOT-W00
track_id: PILOT-READINESS-2026-07
title: Three-person controlled pilot
revision: 3
supersedes: plan-r002.md
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
  - agent/manager_project_session_test.go
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

# PILOT-W00 r003 — 会话键回归测试范围

本修订完整继承 [`plan-r002`](plan-r002.md)（并由其继承 r001）。唯一变化是
把已存在的 `agent/manager_project_session_test.go` 加入 allowlist，使
agent 写入端切换到权威无碰撞会话键时能够同步更新精确断言和增加回归反例。

生产范围、功能承诺、Issue、Step、平台边界和退出门禁均不改变。该测试必须
证明 agent 与 Gateway history 共用相同 key builder；不能以删除或放宽旧断言
代替验证。

