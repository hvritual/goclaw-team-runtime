---
schema: goclaw.wave/v1
wave_id: TC-W01
track_id: TEAM-RUNTIME-2026-07
title: Team Control path boundary and RBAC provenance remediation
revision: 4
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
  - TR-W00
supersedes:
  - docs/waves/team-runtime/tc-w01/plan-r003.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TC-W01-S09
  - TC-W01-S10
  - TC-W01-S11
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - teamcontrol/service.go
  - teamcontrol/service_test.go
  - teamcontrol/validation.go
  - teamcontrol/controlplane_test.go
product_code_changes_allowed: true
---

# TC-W01 r004 — 路径边界与 RBAC provenance 前向修复

r003 candidate exact `e879b0e2...` 的 code/security/docs review 均 BLOCK。
r004 不改写 r002/r003 历史，显式授权必要的 RBAC 文件和路径校验边界，并在
新的 Task Freeze 后做前向实现。

## 已复现问题

### 路径规范化

- `/tmp/../dev/zero`、`file:///tmp/../dev/zero` 和 percent-encoded
  `..` 可在实际打开时落入 `/dev`、`/proc`、`/sys`；
- Windows 将 `COM¹`、`LPT²`、`COM³` 视为 DOS device name，现有 ASCII
  `1..9` 判断未覆盖；
- `source_kind` 和 URL parse error 仍可能回显不受信任内容，Registry URI
  没有显式长度上限。

### RBAC provenance

r002 exact `6aab01f...` 修改了 `teamcontrol/service.go` 的 read-action
集合，但 r002 Plan/Task scope 未包含该文件，且此前 Evidence 未登记该偏差。
r004 将该文件纳入 exact scope；冻结后以等价的明确 read-action helper
前向实现并由授权矩阵回归证明语义。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W01-S09` | 登记 exact review 4、激活 r004、建立 Policy manifest | `active` |
| `TC-W01-S10` | 冻结 exact Task，前向修复路径与 RBAC provenance | `planned` |
| `TC-W01-S11` | 全量 Gate 与三路 exact-commit final review | `planned` |

## Acceptance

- [ ] r004 activation commit 已包含本 Plan、Registry 和 Policy manifest；
- [ ] r004 Task base 精确等于 activation 的远端 commit/tree；
- [ ] `teamcontrol/service.go` 在冻结后由授权 commit 前向实现等价 RBAC；
- [ ] raw、`file:`、percent-encoded 路径在词法折叠 `.`/`..` 后统一校验；
- [ ] Windows DOS device 覆盖 ASCII `1..9` 和 superscript `¹²³`；
- [ ] Registry URI 有显式长度上限，parse/source-kind 错误不回显输入；
- [ ] 两项目 read/write RBAC、Registry CRUD、Context/usage 回归继续通过；
- [ ] 全仓 Go test/vet、关键 race、UI test/build 通过；
- [ ] final exact commit 的 code/security/docs 均 P0=0/P1=0。

## 回滚

- 边界不确定或输入超限：拒绝，不尝试猜测性修复；
- r002 scope 偏差只读保留，由 r004 前向授权，不 rebase/force-push；
- 任一 P1 未关闭：TC-W01 保持 active，RN-W01 不激活。
