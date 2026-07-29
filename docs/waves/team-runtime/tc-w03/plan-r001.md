---
schema: goclaw.wave/v1
wave_id: TC-W03
track_id: TEAM-RUNTIME-2026-07
title: Knowledge authority and storage convergence
revision: 1
plan_status: draft
wave_state: proposed
approved_by:
owner: unassigned
reviewers: []
depends_on:
  - TC-W02
created_at: 2026-07-29
updated_at: 2026-07-29
allowed_change_scope:
  - candidate scope only; see TC-W02 migration-and-wave-roadmap.md
product_code_changes_allowed: false
---

# TC-W03 r001 — Proposed knowledge authority convergence

本文件只是后继路线占位，不是 implementation 授权。目标、候选路径、
non-goals、Gate 和回滚见
[`TC-W02 migration-and-wave-roadmap.md`](../tc-w02/migration-and-wave-roadmap.md)。

激活前必须基于 TC-W02 final Evidence 创建新的 approved revision，冻结
精确 Task；本 draft 不允许产品代码变更或真实数据迁移。

## 目标

在不迁移真实数据的前提下，建立 Team Control 唯一的 Knowledge lifecycle/
approval 权威、唯一可迁移的 content-addressed 正文边界和可重建索引合同。

## 权威输入

- `TC-W02` final P0/P1=0 Evidence；
- `TC-ISSUE-002`；
- `TC-W02 target-contracts.md` Knowledge/authority/object/index 合同；
- `TC-W02 migration-and-wave-roadmap.md` inventory/shadow/cutover 合同。

## 入口门禁

- [ ] TC-W02 已 `complete`，final Evidence 已索引且 P0/P1=0。
- [ ] TC-ISSUE-002 的知识权威部分已具备可验证 acceptance。
- [ ] 新 approved revision 已冻结精确路径、contract、base 和回滚。
- [ ] synthetic inventory、migration fixture 和证据位置已确定。
- [ ] 独立 architecture/security/data reviewer 已分配。

## 范围

### 包含

候选范围：

- `teamcontrol/**`；
- `memory/catalog/**`；
- `gateway/memory_catalog.go`、`gateway/team_control.go`、`gateway/team_guard.go`；
- `cli/commands/memory_catalog.go`；
- 独立 migration tool、直接相关 tests 和 docs。

### 不包含

不包含 MCP、Runner Evidence、UI 全量改造、真实数据迁移/切换、HA 或外部
同步。新 approved revision 必须把上述 glob 收敛为精确文件/contract。

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `TC-ISSUE-002` | KnowledgeSource 与 Memory Catalog 形成独立知识真相源 | `unverified` contract gap | TC-W02 current-state matrix/final Evidence | 用 synthetic fixtures 收敛 authority/content/index，不宣称真实缺陷 |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | 多入口展示/编辑 | 本 Wave 不改 UI，只冻结 projection contract | 旧 UI 误写 active |
| RPC/API | Team Control 与 Catalog 分离 | candidate/approval/authority 统一入口 | legacy caller 行为变化 |
| 权限 | project 与 `"*"` shared 混合 | server-derived project/global authority | global 越权 |
| 数据 | 两套正文/状态边界 | 单一 content store；index 可重建 | 错误归类或双写 |
| 部署 | 无迁移 checkpoint contract | shadow-only adapter 与 rollback gate | 提前 cutover |
| 文档 | 责任边界分散 | authoritative schema/migration runbook | 文档与实现漂移 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `TC-W03-S01` | TC-W02 complete | 创建 approved revision 并冻结 exact Task | docs/waves | binding validator | `proposed` |
| `TC-W03-S02` | S01 | KnowledgeRecord/Revision/Candidate/Approval/Audit schema 与授权 | approved exact scope | auth/state tables | `proposed` |
| `TC-W03-S03` | S02 | 单一 content store、可重建 index adapter、legacy shadow import | approved exact scope | checksum/rebuild fixture | `proposed` |
| `TC-W03-S04` | S03 | migration/atomic/race/rollback deterministic Gate | tests/tools | deterministic suite | `proposed` |
| `TC-W03-S05` | S04 | architecture/security/data-docs independent review | evidence/docs | P0/P1=0 | `proposed` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `TC-EVID-W03-001` | auth/state deterministic | project/global、single-active CAS、职责分离通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W03-002` | storage/migration deterministic | tamper、rebuild、`"*"` quarantine、idempotency、race/rollback 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W03-003` | independent review | exact candidate 三路 P0/P1=0 | `docs/waves/evidence-index.md` | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| 双 active/双写 | CAS 或 writer-negative test 失败 | 单写者、CAS、compat flag 禁止新 active | 回到 shadow-off base |
| 错误 global 分类 | entitlement/approval matrix 差异 | 客户端 `*` 拒绝、global 独立审批 | quarantine，不进入 Context |
| 正文或索引成为第二权威 | index 无法由 event/object 重建 | object checksum + Team Control state；index 可重建 | 丢弃 shadow/index |

## 退出门禁

- [ ] Team Control 是 knowledge lifecycle/approval 的唯一逻辑权威。
- [ ] 正文只有一个 content-addressed boundary，index 可从事件重建。
- [ ] 旧 Catalog active writer 可显式关闭，真实库仍未迁移。
- [ ] migration dry-run/rollback 和三路独立复核 P0/P1=0。
- [ ] Evidence 已登记，未解决风险已进入后继 Wave。
- [ ] TC-W04 仍未激活。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-29 | `TC-DEC-003` | Team Control 管理知识身份/lifecycle；正文单一、索引可重建 | 消除第二权威源；真实迁移留待独立授权 |

## Plan revision

本 r001 是未批准的 proposed draft，可在激活前被新 approved revision
supersede；激活版本必须记录 exact TC-W02 Evidence、范围、base、reviewer 和
重新审批，不得把本 draft 直接解释为产品代码授权。
