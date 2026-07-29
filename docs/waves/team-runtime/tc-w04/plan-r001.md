---
schema: goclaw.wave/v1
wave_id: TC-W04
track_id: TEAM-RUNTIME-2026-07
title: Policy resolver and Context Compiler v2
revision: 1
plan_status: draft
wave_state: proposed
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

## 权威输入

- `TC-W02` final Policy/Context contracts；
- `TC-W03` final exact Evidence；
- frozen Task、Repository、member、budget、Knowledge/Skill identity。

## 范围与 non-goals

候选范围：

- `teamcontrol/**`、`orchestratorlite/**`；
- `gateway/team_control.go`、`gateway/development.go`；
- `workstation/types.go`、`workstation/queue.go`；
- `workstation/service_test.go`、`gateway/team_runtime_test.go`；
- 直接相关 docs/tests。

只允许最小 ExecutionPack ContextManifest binding；不实现 MCP、Runner
feedback、UI 全量或真实迁移。approved revision 必须冻结精确路径。

## 分步计划

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W04-S01` | 激活/freeze exact Policy/Context Task | proposed |
| `TC-W04-S02` | global defaults + team/project/repository/component resolver | proposed |
| `TC-W04-S03` | global mandatory set、provenance、canonical resolved hash | proposed |
| `TC-W04-S04` | ContextManifest/Bundle v2 与 ExecutionPack server binding | proposed |
| `TC-W04-S05` | golden/negative/race/full deterministic Gate | proposed |
| `TC-W04-S06` | architecture/security/docs independent review | proposed |

## 验证与证据

- Policy 冲突矩阵和 global authority/epoch；
- RFC 8785/domain-separated golden vectors；
- task/member/repository/budget/knowledge/Skill/expiry/visibility/checksum；
- secret schema、unknown field、tamper、cross-project negative cases；
- ExecutionPack exact manifest hash server binding；
- 三路 final P0=0/P1=0。

## 风险与回滚

| 风险 | 缓解 | 回滚 |
|---|---|---|
| mandatory 被下层覆盖 | final validation + global set epoch | 保留 v1 resolver，不生成 v2 Context |
| cross-runtime hash 漂移 | shared golden vectors | 禁止 mixed compiler versions |
| v1 Bundle 被误当 executable | schema namespace + explicit denial | v1 只读历史 |

## 退出门禁

- 相同 material 生成稳定 resolved policy/manifest ID/hash；
- mandatory、secret、scope、checksum 任一错误失败关闭；
- ExecutionPack 只含 server-built manifest binding；
- MCP/feedback 尚未实现，TC-W05 未激活；
- deterministic 和三路 independent review P0/P1=0。
