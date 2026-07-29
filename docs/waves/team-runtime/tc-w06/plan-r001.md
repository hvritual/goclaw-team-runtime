---
schema: goclaw.wave/v1
wave_id: TC-W06
track_id: TEAM-RUNTIME-2026-07
title: Console, CLI, operations, and controlled cutover
revision: 1
plan_status: draft
wave_state: proposed
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

## 权威输入

- `TC-W03`–`TC-W05` final exact Evidence；
- `TC-W02` UI、migration、backup/restore/rollback contracts；
- synthetic legacy inventory 和 operator-owned cutover approval。

## 范围与 non-goals

候选范围：

- `ui/src/team/**`、`ui/tests/**`；
- Gateway knowledge/policy/context/approval projections；
- Team CLI、`plugins/obsidian-goclaw/**`；
- migration/backup/restore tools、`config/**`、`deploy/**` 和 docs/tests。

不包含 HA/Leader、外部 Git/Jira、真实 Pilot release、未批准真实数据迁移或
无关 Runner update。真实 cutover 另需 operator Task。

## 分步计划

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W06-S01` | activation/freeze、synthetic inventory 与 recovery preflight | proposed |
| `TC-W06-S02` | Policy/Knowledge/Context/usage/candidate UI + Team CLI | proposed |
| `TC-W06-S03` | legacy adapters、shadow import、dual-read compare | proposed |
| `TC-W06-S04` | synthetic read cutover、old-writer disable、backup/restore | proposed |
| `TC-W06-S05` | UI/RPC/migration/recovery/credential deterministic Gate | proposed |
| `TC-W06-S06` | UX/security/operations/docs independent review | proposed |

## 验证与证据

- loading/empty/denied/error/conflict/stale/disconnected/checksum mismatch；
- rule provenance、mandatory outcome、citation usage、approval separation；
- RPC project/global auth 和 client `*` rejection；
- inventory/shadow/compare/cutover/rollback idempotency；
- signed monotonic backup epoch、event replay、index rebuild、tamper rejection；
- credential scan 和 Obsidian/Vault secret negative cases。

## 风险与回滚

| 风险 | 缓解 | 回滚 |
|---|---|---|
| UI 把 stale 当 active | typed state + explicit blocking presentation | 关闭 mutation，保留旧 read projection |
| cutover 遗漏记录 | manifest count/hash/relationship compare | 回旧 read snapshot，保留新 audit |
| stale restore 复活旧状态 | signed epoch + event replay | non-executable recovery |

## 退出门禁

- synthetic migration/cutover/restore/rollback 全通过；
- old writer 在 Team mode 被确定性拒绝；
- Web/CLI/Obsidian 都是 projection，不能扩大 scope/写 active/携带 secret；
- independent UX/security/ops-docs P0/P1=0；
- 真实数据未迁移，REL-W01 尚未激活。
