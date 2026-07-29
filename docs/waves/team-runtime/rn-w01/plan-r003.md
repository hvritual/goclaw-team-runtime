---
schema: goclaw.wave/v1
wave_id: RN-W01
track_id: TEAM-RUNTIME-2026-07
title: Runner lifecycle and trusted execution-profile selection
revision: 3
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
  - docs/waves/team-runtime/rn-w01/plan-r002.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - RN-W01-S03A
  - RN-W01-S03B
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

# RN-W01 r003 — 受信 Execution Profile 选择

r002 正确冻结了双 profile、安全边界和生命周期目标，但遗漏团队模式唯一
允许的 `dev.task.enqueue` 受信构建路径。只修改 `runner.enqueue` 没有意义，
因为 Team mode 已失败关闭该旁路。r003 不改写 r002 历史，只扩展必要的
Gateway/CLI exact scope。

## 新增授权

- `gateway/development.go`：从 resolved Team policy 读取允许 profile，服务端
  重建 ExecutionPack，添加匹配 capability；不信任 client execution pack；
- `gateway/development_wave_test.go`、`gateway/team_runtime_test.go`：默认
  strict、delegated opt-in、policy deny 与 capability mismatch 回归；
- `cli/dev.go`、`cli/dev_enqueue_test.go`：向受信 enqueue 请求显式传递
  `--execution-profile`，默认 strict。

其余定位、双 profile 合同、Team Control 职责、更新状态机、Acceptance、
风险与回滚完整继承
[`plan-r002.md`](plan-r002.md)。远程 release fetch 在完整网络策略前仍
失败关闭。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `RN-W01-S03A` | 登记 r002 scope gap、激活 r003、生成 manifest | `active` |
| `RN-W01-S03B` | 冻结 r003 exact Task | `planned` |
| `RN-W01-S04` | 双 profile、Team policy、doctor、directory Gate | `planned` |
| `RN-W01-S05` | 本地 artifact stage/verify/activate/rollback | `planned` |
| `RN-W01-S06` | migration、多项目并发与 lifecycle Evidence | `planned` |
| `RN-W01-S07` | 三平台 CLI/deploy/操作文档 | `planned` |
| `RN-W01-S08` | 全量 Gate 与三路 exact final review | `planned` |

## Additional acceptance

- [ ] Team mode 只有 `dev.task.enqueue` 能选择 profile；
- [ ] client 传入的 `execution_pack` 继续被忽略；
- [ ] resolved policy 未声明时仅允许 strict；
- [ ] `codex-delegated` 要求 policy allow 与 runner capability 同时成立；
- [ ] commit/file scope、Task tuple 与 r003 manifest 全部可复核。
