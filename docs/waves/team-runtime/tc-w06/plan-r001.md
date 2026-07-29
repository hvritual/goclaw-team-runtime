---
schema: goclaw.wave/v1
wave_id: TC-W06
track_id: TEAM-RUNTIME-2026-07
title: Console, CLI, operations, and controlled cutover
revision: 1
plan_status: draft
wave_state: proposed
approved_by:
owner: unassigned
reviewers: []
depends_on:
  - TC-W05
created_at: 2026-07-29
updated_at: 2026-07-29
allowed_change_scope:
  - candidate scope only; see TC-W02 migration-and-wave-roadmap.md
product_code_changes_allowed: false
---

# TC-W06 r001 — Proposed operations and cutover

本文件只是路线占位。目标、候选路径、non-goals、Gate 和回滚见
[`TC-W02 migration-and-wave-roadmap.md`](../tc-w02/migration-and-wave-roadmap.md)。

激活前必须新建 approved revision 和 exact Task Freeze；本 draft 不允许
真实知识迁移、运行时 cutover 或发布。

## 目标

交付明确状态的 Console/CLI/Obsidian projection 和可演练的 synthetic
migration、backup、restore、rollback 运维路径，同时保持真实数据与发布未变。

## 权威输入

- `TC-W03`–`TC-W05` final exact Evidence；
- `TC-W02` UI、migration、backup/restore/rollback contracts；
- synthetic legacy inventory 和 operator-owned cutover approval。

## 入口门禁

- [ ] TC-W05 已 `complete`，final Evidence 已索引且 P0/P1=0。
- [ ] UI/RPC 状态、migration manifest 和 recovery checkpoint 合同已冻结。
- [ ] 新 approved revision 已冻结 UI/Gateway/tool/config 精确路径。
- [ ] synthetic dataset、backup/restore environment 和 operator reviewer 已确定。
- [ ] 真实数据、真实 cutover 和 release 仍有独立授权门禁。

## 范围

### 包含

候选范围：

- `ui/src/team/**`、`ui/tests/**`；
- Gateway knowledge/policy/context/approval projections；
- Team CLI、`plugins/obsidian-goclaw/**`；
- migration/backup/restore tools、`config/**`、`deploy/**` 和 docs/tests。

### 不包含

不包含 HA/Leader、外部 Git/Jira、真实 Pilot release、未批准真实数据迁移或
无关 Runner update。真实 cutover 另需 operator Task。

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `TC-ISSUE-002` | UI/CLI/Obsidian 与迁移运维缺少统一 authority projection/cutover contract | `unverified` contract gap | TC-W02 matrix/contracts | 用 synthetic 数据实现状态投影和可恢复运维路径 |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | 规则/知识状态不完整 | 展示 effective/provenance/candidate/usage 和显式状态 | stale 被当 active |
| RPC/API | legacy projection/commands | server-scoped typed projections | 客户端扩大 scope |
| 权限 | UI/CLI 可能自报 project | server derives authority；global 独立角色 | approval 越权 |
| 数据 | 旧源并存 | synthetic shadow/compare/cutover only | 记录遗漏或双写 |
| 部署 | backup/restore 未演练 | monotonic checkpoint/replay/index rebuild | stale restore |
| 文档 | runbook 分散 | migration/recovery/operator guide | 操作顺序漂移 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `TC-W06-S01` | TC-W05 complete | activation/freeze、synthetic inventory 与 recovery preflight | docs/waves | binding/preflight | `proposed` |
| `TC-W06-S02` | S01 | Policy/Knowledge/Context/usage/candidate UI + Team CLI | approved exact scope | UI/RPC state tests | `proposed` |
| `TC-W06-S03` | S02 | legacy adapters、shadow import、dual-read compare | approved exact scope | migration compare | `proposed` |
| `TC-W06-S04` | S03 | synthetic read cutover、old-writer disable、backup/restore | approved exact scope | recovery drill | `proposed` |
| `TC-W06-S05` | S04 | UI/RPC/migration/recovery/credential deterministic Gate | tests/tools | deterministic suite | `proposed` |
| `TC-W06-S06` | S05 | UX/security/operations/docs independent review | evidence/docs | P0/P1=0 | `proposed` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `TC-EVID-W06-001` | UI/RPC deterministic | 全部显式状态、provenance、approval、project/global auth 与 client `"*"` rejection 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W06-002` | migration/recovery deterministic | inventory/shadow/compare/cutover/rollback、epoch/replay/rebuild/tamper/credential 通过 | 待 approved revision 冻结 | `planned` |
| `TC-EVID-W06-003` | independent review | exact candidate UX/security/ops-docs P0/P1=0 | `docs/waves/evidence-index.md` | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| UI 把 stale 当 active | state snapshot/interaction test 失败 | typed state + explicit blocking presentation | 关闭 mutation，只保留服务器从当前 Team Control state 派生的 read-only compatibility projection |
| cutover 遗漏记录 | manifest count/hash/relationship mismatch | compare + mutation quiescence | checkpoint + full event replay；否则 non-executable |
| stale restore 复活旧状态 | epoch 倒退或 event gap | signed epoch + event replay | non-executable recovery |

## 退出门禁

- [ ] synthetic migration/cutover/restore/rollback 全通过。
- [ ] old writer 在 Team mode 被确定性拒绝。
- [ ] Web/CLI/Obsidian 都是 projection，不能扩大 scope/写 active/携带 secret。
- [ ] independent UX/security/ops-docs P0/P1=0。
- [ ] Evidence 已登记，runbook/rollback 可执行。
- [ ] 真实数据未迁移，REL-W01 尚未激活。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-29 | `TC-DEC-006` | UI/CLI/Obsidian 仅为 projection；迁移分 inventory/shadow/compare/cutover | 避免客户端或迁移器成为第二权威 |

## Plan revision

本 r001 是 proposed draft。激活必须创建新 approved revision，冻结 TC-W05
Evidence、synthetic/real boundary、exact scope、operator approval、recovery
fixtures 与 reviewer；本文件不授权真实迁移、cutover 或发布。
