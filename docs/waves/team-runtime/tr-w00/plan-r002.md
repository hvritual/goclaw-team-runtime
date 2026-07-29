---
schema: goclaw.wave/v1
wave_id: TR-W00
track_id: TEAM-RUNTIME-2026-07
title: Team Control and Runner application boundary acceptance remediation
revision: 2
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - MVP-W00
supersedes:
  - docs/waves/team-runtime/tr-w00/plan-r001.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TR-W00-S06
  - TR-W00-S07
  - TR-W00-S08
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - README.md
  - scripts/build-release.sh
  - scripts/build-apps.sh
  - Makefile
  - cli/application_test.go
  - workstation/localexec.go
  - workstation/localexec_test.go
  - deploy/systemd/**
product_code_changes_allowed: true
---

# TR-W00 r002 — 独立验收修复

TR-W00 r001 的实现提交 `9d6a25276c9eda32360b801aa2c5fde1bd46e863`
已在独立验收前通过 PR #1 合并为
`3a75c7376d73e41f33e2b94eb3bb1ca4c30219fd`。三路独立复核均报告
`P0=0`，但仍有未关闭 P1，因此 r001 不得被描述为已验收完成。本 revision
只做前向修复，不改写或撤销已有 Git 历史。

## 已复现的阻断

### Code

1. source release allowlist 缺少 `cmd/team-control` 与 `cmd/runner`，源码包
   不能重建两个独立应用；
2. `build-apps.sh` 在非空目标目录直接构建并散列所有文件，陈旧文件可能
   进入交付清单。

### Security

1. Runner 把真实 `CODEX_HOME` 暴露给 Codex 进程，但 r001 没有用确定性
   负向 canary 证明模型生成的命令无法读取 OAuth 文件；
2. binary credential Gate 漏扫 Reviewer、Runner device key 和 Codex
   access/refresh token，`grep` 参数也没有失败关闭处理前导连字符。

### Docs / governance

1. Wave README 的 current projection 与 Registry 冲突；
2. 状态机没有定义受治理的 `active -> superseded`；
3. PR #1 在独立验收前合并，违反自身 Draft gate；
4. 根 README 仍把操作者引导到上游单应用源码。

## 修复合同

- source archive 显式包含两个 `cmd` 入口，并在解包后从归档内容编译；
- cross-build 只在任务专用 staging 中生成精确 18 项清单，目标非空时
  fail closed，不散列额外文件；
- binary credential Gate 覆盖 Team/Gateway/Reviewer/Runner/Codex/GitHub
  凭据原值，参数安全，失败时不发布 staging；
- Runner 使用 Codex named permission profile，让工作区可写、网络关闭，
  同时对真实 `CODEX_HOME` 设置 OS sandbox `deny`；每个执行前运行不访问
  模型的负向 sandbox canary，旧版或不支持的 Codex 失败关闭；
- 根 README、Wave current projection、状态机和提前合并偏差前向修复；
- 应用真实入口测试、clean 和 build provenance 的 P2 同步收敛，但不提前
  实现 TC-W01/RN-W01/INT-W01。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `TR-W00-S06` | 登记三路 findings、偏差和 r002 authority | `active` |
| `TR-W00-S07` | 修复构建、凭据、Codex read-deny 与文档入口 | `planned` |
| `TR-W00-S08` | 全量确定性 Gate 与三路独立复核 | `planned` |

## Acceptance

- [ ] r002 Task 从包含本 Plan/Registry/Policy 的 exact activation commit 冻结；
- [ ] source archive 解包后能构建两个 dedicated entrypoint；
- [ ] cross-build 的产物名称和数量与 18 项 expected manifest 精确相等；
- [ ] 所有冻结凭据变量都执行安全原值扫描，命中时不发布候选产物；
- [ ] Codex sandbox canary 在真实模型调用前证明模型命令不能读取
      `CODEX_HOME`，不支持 named permission profile 时失败关闭；
- [ ] 根 README 不再把本 fork 用户引导到上游单应用安装；
- [ ] Wave projection/state machine/提前合并偏差一致且可追溯；
- [ ] Go test/race/vet、UI、Obsidian、cross-build、release Gate 通过；
- [ ] 新 exact review commit 的 code/security/docs 复核均为 P0=0/P1=0。

## 回滚

- named permission profile 或 canary 不受目标 Codex/OS 支持：Runner 拒绝
  执行任务，不降级到只靠 prompt 或旧 `workspace-write`；
- staging/manifest/credential Gate 失败：删除任务专用候选 stage，不覆盖
  已发布目录；
- 文档或治理仍不一致：TR-W00 保持 active，TC-W01 不激活；
- PR #1 的提前合并只记录偏差，不 rebase、不 force-push、不伪造验收时间。
