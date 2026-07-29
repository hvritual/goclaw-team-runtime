---
schema: goclaw.wave/v1
wave_id: TC-W04
track_id: TEAM-RUNTIME-2026-07
title: Policy resolver and Context Compiler v2
revision: 1
plan_status: draft
wave_state: proposed
approved_by:
owner: unassigned
reviewers: []
depends_on:
  - TC-W03
created_at: 2026-07-29
updated_at: 2026-07-29
allowed_change_scope:
  - candidate scope only; see TC-W02 migration-and-wave-roadmap.md
product_code_changes_allowed: false
---

# TC-W04 r001 — Proposed resolver and compiler v2

本文件只是路线占位。目标、候选路径、non-goals、Gate 和回滚见
[`TC-W02 migration-and-wave-roadmap.md`](../tc-w02/migration-and-wave-roadmap.md)。

激活前必须新建 approved revision 和 exact Task Freeze；本 draft 不允许
产品代码变更。

## 目标

产出可重现、带来源和 hash 的 resolved policy，以及只含无秘密 opaque refs、
并被 ExecutionPack 精确绑定的 immutable Context Manifest/Bundle v2。

## 权威输入

- `TC-W02` final Policy/Context contracts；
- `TC-W03` final exact Evidence；
- frozen Task、Repository、member、budget、Knowledge/Skill identity。

## 入口门禁

- [ ] TC-W03 已 `complete`，final Evidence 已索引且 P0/P1=0。
- [ ] global authority、knowledge revision/content boundary 已冻结。
- [ ] 新 approved revision 已冻结 resolver/compiler/Workstation 精确路径。
- [ ] canonical/golden fixtures 与 cross-runtime verification 已确定。
- [ ] rollback 能禁止 v2 executable 并保留 v1 history read-only。

## 范围

### 包含

候选范围：

- `teamcontrol/**`、`orchestratorlite/**`；
- `gateway/team_control.go`、`gateway/development.go`；
- `workstation/types.go`、`workstation/queue.go`；
- `workstation/service_test.go`、`gateway/team_runtime_test.go`；
- 直接相关 docs/tests。

### 不包含

只允许最小 ExecutionPack ContextManifest binding；不实现 MCP、Runner
feedback、UI 全量或真实迁移。approved revision 必须冻结精确路径。

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `TC-ISSUE-002` | 当前 Policy/Context 缺少完整 precedence、mandatory final gate 和执行身份绑定 | `unverified` contract gap | TC-W02 matrix/contracts | 用确定性 fixtures 实现并验证 resolver/compiler contract |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | 展示局部 policy/context | 本 Wave 仅提供 provenance projection contract | effective rule 误展示 |
| RPC/API | compile 接受不完整身份 | server derives full frozen inputs | 客户端依赖旧字段 |
| 权限 | project/member 检查有限 | mandatory/global authority 与 project scope final validation | 下层绕过 |
| 数据 | ContextBundle v1 file state | schema-namespaced immutable v2 | v1/v2 混用 |
| 部署 | compiler version 未冻结 | version/golden-vector gate | mixed runtime hash 漂移 |
| 文档 | precedence 分散 | canonical contract 和 examples | 实现/合同漂移 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `TC-W04-S01` | TC-W03 complete | 激活/freeze exact Policy/Context Task | docs/waves | binding validator | `proposed` |
| `TC-W04-S02` | S01 | global defaults + team/project/repository/component resolver | approved exact scope | conflict matrix | `proposed` |
| `TC-W04-S03` | S02 | global mandatory set、provenance、canonical resolved hash | approved exact scope | golden/negative tests | `proposed` |
| `TC-W04-S04` | S03 | ContextManifest/Bundle v2 与 ExecutionPack server binding | approved exact scope | identity/tamper tests | `proposed` |
| `TC-W04-S05` | S04 | golden/negative/race/full deterministic Gate | tests | deterministic suite | `proposed` |
| `TC-W04-S06` | S05 | architecture/security/docs independent review | evidence/docs | P0/P1=0 | `proposed` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `TC-EVID-W04-001` | policy deterministic | 冲突矩阵、mandatory authority/epoch、JCS vectors 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W04-002` | context deterministic | identity/budget/expiry/checksum/secret/tamper/cross-project 与 pack binding 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W04-003` | independent review | exact candidate 三路 P0/P1=0 | `docs/waves/evidence-index.md` | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| mandatory 被下层覆盖 | conflict matrix 产生 override | final validation + global set epoch | 保留 v1 resolver，不生成 v2 Context |
| cross-runtime hash 漂移 | golden ID/hash 不一致 | shared golden vectors | 禁止 mixed compiler versions |
| v1 Bundle 被误当 executable | schema/audience negative test 失败 | schema namespace + explicit denial | v1 只读历史 |

## 退出门禁

- [ ] 相同 material 生成稳定 resolved policy/manifest ID/hash。
- [ ] mandatory、secret、scope、checksum 任一错误失败关闭。
- [ ] ExecutionPack 只含 server-built manifest binding。
- [ ] deterministic 和三路 independent review P0/P1=0。
- [ ] Evidence 已登记，回滚可执行。
- [ ] MCP/feedback 尚未实现，TC-W05 未激活。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-29 | `TC-DEC-004` | defaults 可覆盖；mandatory 集合独立并在 merge 后 final validate | 保留项目灵活性且不允许下层绕过约束 |

## Plan revision

本 r001 是 proposed draft。任何激活版必须以新 approved revision 冻结 exact
scope、TC-W03 evidence、canonical vectors、回滚和 reviewer；不得原地把本
draft 改为执行授权。
