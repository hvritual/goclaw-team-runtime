---
schema: goclaw.wave/v1
wave_id: MC-W01
track_id: MULTICA-BASELINE-2026-07
title: Multica baseline adoption and six-domain verification
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-confirmation-2026-07-29-multica-plan-a-g3
owner: Codex root agent
reviewers:
  - independent_architecture_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - MVP-W00
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - MC-W01-S01
  - MC-W01-S02
  - MC-W01-S03
  - MC-W01-S04
  - MC-W01-S05
allowed_change_scope:
  - docs/waves/**
  - all tracked repository paths for one exact Multica upstream tree replacement
  - Multica-native six-domain documentation, tests, and confirmed gap fixes
product_code_changes_allowed: true
---

# MC-W01 r001 — Multica 基线采用与六域验收

## 目标

把目标工作线转换为完整、干净、可追溯的 Multica 上游基线，保留现有
GoClaw 源码与计划于本地 backup 分支；第一阶段只盘点、验证并在证据确认
缺口后补齐 Workspace、Member、Project、Issue、Task、Skill 六域。

不变量：

- Multica 是唯一产品与数据权威；
- 目标工作树不保留 GoClaw 产品代码、配置、品牌或旧 Wave 文档；
- 不裁剪 Multica 的 Agent、Runtime、Realtime、共享 API 等原生依赖；
- 不改写 `main`、`origin` 或远端，不部署、不访问真实数据或凭据；
- backup 在独立验收前不得删除。

## 权威输入

- 用户确认：采用完整 Multica 基线的方案 A，并明确授权 backup 后替换整个
  tracked tree；
- 关联 Issue：`MC-ISSUE-001`；
- 关联决策：`MC-DEC-001`；
- 前置证据：`MVP-EVID-001`–`006`；
- 当前事实：GoClaw HEAD `3150c9c8ac4439add888ba8f0c232f26be201fe7`
  工作树干净；Multica `main` 候选
  `beb3e9be65023f63bd5dfdbb0231ed41aa9f1cb8`，执行前必须重新冻结。

## 入口门禁

- [x] `MVP-W00` 已 complete。
- [x] 用户确认目标、G3 替换范围与 backup 边界。
- [x] 允许范围、非目标、停止与恢复路径已冻结。
- [x] 当前工作树、分支、远端和候选上游 commit 已只读盘点。
- [ ] activation commit 已包含 Plan、Registry 与 Policy manifest。
- [ ] Task 从 activation exact commit/tree 冻结。
- [ ] `BACKUP-VERIFIED` 已通过。

## 范围

### 包含

- docs-only 路线转换、Task freeze 与 backup Evidence；
- 创建本地 `codex/backup-goclaw-pre-multica-20260729`；
- 在 `codex/multica-six-domain-baseline` 做一次 exact upstream tree replacement；
- 六域数据模型、状态机、API、权限、事件、UI、CLI、测试、依赖源码映射；
- 仅对确定性测试证实的六域缺口做后续小步修复。

### 不包含

- GoClaw `teamcontrol`、Gateway、Runner、Obsidian 或旧数据迁入目标树；
- 删除 Multica 其他域或原生支撑依赖；
- 改写/推送 `main`、remote、PR、部署、真实 Agent smoke、真实凭据/数据；
- 商业托管、品牌修改或许可证解释。

## 问题与事实

| Issue ID | 表面症状 | 当前状态 | 证据 | 本 Wave 责任 |
|---|---|---|---|---|
| `MC-ISSUE-001` | 目标产品尚未以干净 Multica tree 承载六域 | `reproduced` | Git tree/branch 与上游源码盘点 | backup、tree adoption、六域 baseline |

## 影响分析

| 影响面 | 当前契约 | 计划变化 | 兼容/迁移风险 |
|---|---|---|---|
| UI | GoClaw UI | 完整采用 Multica Web/Desktop/Mobile | 全树替换，仅 backup 恢复 |
| API | GoClaw RPC | 完整采用 Multica REST/WS | 不做双写或兼容 sidecar |
| 权限 | GoClaw TeamControl | Multica workspace membership | 六域差距后续单独登记 |
| 数据 | file/SQLite 混合 | Multica PostgreSQL/sqlc | 本 Wave 不迁移真实数据 |
| 部署 | GoClaw release | 不部署 | 无运行环境影响 |
| 文档 | GoClaw Waves | backup 保留；目标树只留 Multica 相关材料 | 通过 Git parent/backup 追溯 |

## 分步计划

| Step ID | 前置 | 计划动作 | 允许文件/模块 | 验证 | 状态 |
|---|---|---|---|---|---|
| `MC-W01-S01` | 用户确认 | 激活 Plan/Registry/Policy | `docs/waves/**` | JSON、hash、links、clean diff | `active` |
| `MC-W01-S02` | activation commit | freeze Task、创建并验证 backup | `docs/waves/**`、Git refs | exact SHA/tree、checkout 可解析 | `planned` |
| `MC-W01-S03` | BACKUP-VERIFIED | 在目标分支替换为 frozen Multica tree | all tracked paths | candidate tree == upstream tree | `planned` |
| `MC-W01-S04` | S03 PASS | 形成六域 baseline 与确定性验证 | Multica docs/tests | domain matrix、narrow/full checks | `planned` |
| `MC-W01-S05` | S04 PASS | 三路独立复核 | Evidence only | P0=0/P1=0 | `planned` |

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 保存位置 | 状态 |
|---|---|---|---|---|
| `MC-EVID-W01-001` | backup | exact ref/tree/checkout 可恢复 | backup branch + journal | `planned` |
| `MC-EVID-W01-002` | tree adoption | target tree 与 frozen upstream 相同 | target commit | `planned` |
| `MC-EVID-W01-003` | six-domain baseline | 六域合同/源码/测试映射完整 | target Multica docs | `planned` |
| `MC-EVID-W01-004` | deterministic | Go/TS/必要 E2E 通过 | target evidence | `planned` |
| `MC-EVID-W01-005` | independent review | architecture/security/docs P0/P1=0 | target evidence | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 缓解 | 回滚 |
|---|---|---|---|
| 覆盖用户工作 | dirty tree/未知 worktree | 替换前双检、禁止 broad delete | 停止；不改 refs |
| backup 不可恢复 | ref/tree/checkout 不一致 | BACKUP-VERIFIED 硬门禁 | 保持原分支 |
| 上游漂移 | main SHA 变化 | 冻结 exact SHA/tree | 停止并记录新 tuple |
| 目标树混入 GoClaw | tree diff/关键字命中 | exact tree comparison | 切回 backup |
| 依赖验证过宽 | 调用真实 Agent/凭据 | synthetic/fake executable | 停止相关测试 |

## 退出门禁

- [ ] backup 可恢复且未删除；
- [ ] 目标 tree 与 frozen Multica tree 的初始采用提交相同；
- [ ] 六域 baseline、确定性验证和缺口记录完整；
- [ ] 默认测试未访问真实 Agent、凭据或数据；
- [ ] architecture/security/docs P0=0/P1=0；
- [ ] 证据与最终 exact identifiers 已登记；
- [ ] 未改写或推送 `main`、`origin`、远端。

## 决策记录

| 日期 | Decision ID | 决策 | 原因与影响 |
|---|---|---|---|
| 2026-07-29 | `MC-DEC-001` | 完整 Multica tree；六域先行；旧树只留 backup | 保留原生依赖并消除 GoClaw tree 污染 |

## Plan revision

r001 获批后不可原地改写。任何目标 tree、分支影响、六域范围、push/deploy、
数据迁移或验收变化均需新 revision 和重新确认。
