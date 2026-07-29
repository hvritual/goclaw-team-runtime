---
schema: goclaw.wave/v1
wave_id: RN-W01
track_id: TEAM-RUNTIME-2026-07
title: Runner device authentication and acceptance remediation
revision: 5
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
  - user-directive-2026-07-29-finish-current-task-then-pause
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TC-W01
supersedes:
  - docs/waves/team-runtime/rn-w01/plan-r004.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - RN-W01-S07A
  - RN-W01-S07B
  - RN-W01-S08
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - README.md
  - teamcontrol/**
  - workstation/**
  - gateway/workstation.go
  - gateway/workstation_test.go
  - gateway/development.go
  - gateway/development_wave_test.go
  - gateway/team_control.go
  - gateway/team_runtime_test.go
  - gateway/server.go
  - gateway/principal.go
  - gateway/team_guard.go
  - cli/runner.go
  - cli/runner_test.go
  - cli/runner_security_test.go
  - cli/system.go
  - cli/dev.go
  - cli/dev_enqueue_test.go
  - config/**
  - deploy/**
  - scripts/**
product_code_changes_allowed: true
---

# RN-W01 r005 — 设备身份与验收修复

r005 完整继承
[`plan-r002.md`](plan-r002.md)、
[`plan-r003.md`](plan-r003.md) 和
[`plan-r004.md`](plan-r004.md)。r004 exact candidate
`d6a166ceb1f445e7098855841d20bf3903f0d3d5` 的独立 code/security/docs
验收确认以下 P1：

- 长期 Runner work loop 仍持有成员 `GOCLAW_USER_TOKEN`，与 r002 的
  Runner 不接收 Team Token 合同冲突；
- claim 未重校验 repository-scoped current policy，通配 capability 可满足
  profile/version/release pin；
- active release 未绑定实际运行 executable，且 Runner 启动与 release
  activation 存在锁竞态；
- Windows owner SID 未验证，Windows workstation 冻结交叉测试无法编译；
- 真实 PolicyBundle 写入面拒绝 `runner.*`，legacy zero-size release 仍可
  update，构建/状态文档与产物不一致。

其中 Runner device authentication 需要 `gateway/server.go`、
`gateway/principal.go`、`gateway/team_guard.go` 与 `cli/system.go`，超出
r004 exact scope。r005 只前向增加这些认证边界、对应测试与 README；其他
修复仍在继承 scope 内。它不增加远程 artifact fetch、网络 sandbox、Job
Object、installer、自动重启或后续 INT/REL Wave。

## Acceptance

- [ ] `runner work` 在 `GOCLAW_USER_TOKEN` 存在时失败关闭；
- [ ] device credential 只能调用其自身
  `ping/claim/heartbeat/complete/fail`，不能调用成员或管理接口；
- [ ] claim 在授予 lease 前按任务 repository 重算 policy hash、profile 与
  version/release pin；
- [ ] wildcard 不能满足 lifecycle/profile 安全 capability；
- [ ] active release 的 version/path/size/SHA 与 `os.Executable()` 一致；
- [ ] Runner start 与 activate 使用同一 mutation/process lock 顺序；
- [ ] Windows owner + DACL + reparse 检查和冻结 cross-test 通过；
- [ ] `runner.*` policy 可通过真实 put/resolve 往返；
- [ ] legacy zero-size release 可读但不可 stage/update；
- [ ] deterministic Gate 与 code/security/docs exact review 均
  `P0=0/P1=0`；
- [ ] 完成后 RN-W01 标记 complete 并暂停，不激活 INT-W01/REL-W01，不
  自动 merge Draft PR。

## Deferred P2

Windows delegated 使用 `taskkill /T /F` best effort，尚未使用
kill-on-close Job Object。文档必须披露该边界；高保证任务继续使用
Linux/WSL2/Lima strict。本项不阻止 r005，但不得被描述为强进程隔离。
