---
schema: goclaw.wave/v1
wave_id: RN-W01
track_id: TEAM-RUNTIME-2026-07
title: Runner lifecycle with complete release artifact identity
revision: 4
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
  - user-decision-1A-plus-2A
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TC-W01
supersedes:
  - docs/waves/team-runtime/rn-w01/plan-r003.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - RN-W01-S03C
  - RN-W01-S03D
  - RN-W01-S04
  - RN-W01-S05
  - RN-W01-S06
  - RN-W01-S07
  - RN-W01-S08
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - teamcontrol/**
  - workstation/**
  - gateway/workstation.go
  - gateway/workstation_test.go
  - gateway/development.go
  - gateway/development_wave_test.go
  - gateway/team_control.go
  - gateway/team_runtime_test.go
  - cli/runner.go
  - cli/runner_test.go
  - cli/dev.go
  - cli/dev_enqueue_test.go
  - config/**
  - deploy/**
  - scripts/**
product_code_changes_allowed: true
---

# RN-W01 r004 — 完整 Release Artifact Identity

r003 授权了受信 profile 选择，但实现 release stage-from-control 时确认
`RunnerRelease` 的中央合同缺少 `size_bytes`，且 r003 scope 没有允许
Gateway 对该字段做 secret-safe 投影。只在本地记录大小会使中央 release pin
无法完整绑定 artifact identity。

r004 完整继承 [`plan-r002.md`](plan-r002.md) 与
[`plan-r003.md`](plan-r003.md)，只新增：

- `RunnerRelease.size_bytes`：新 release 必须提供正数；旧 state 的零值可读，
  但不得用于 stage/update；
- `gateway/team_control.go`：put/get/list 投影该非秘密字段；
- `runner release stage-from-control`：只接受 `approved` release，要求本机
  artifact 与中央 ID/version/OS/arch/protocol/size/SHA 全匹配；
- 远程 URI 不自动 fetch；operator 明确给出本地 artifact。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `RN-W01-S03C` | 激活 r004 与完整 identity manifest | `active` |
| `RN-W01-S03D` | 冻结 r004 exact Task | `planned` |
| `RN-W01-S04`–`S08` | 实现、Gate、三路 review | `planned` |

## Additional acceptance

- [ ] 新 RunnerRelease 的 `size_bytes > 0`；
- [ ] legacy zero-size record 只读兼容，stage-from-control 失败关闭；
- [ ] immutable release ID 的 size 不能变更；
- [ ] Gateway/CLI 不根据 URI 自动联网或回显凭据；
- [ ] r002/r003 全部 Acceptance 保持。
