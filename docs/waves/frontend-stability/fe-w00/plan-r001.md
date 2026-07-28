---
schema: goclaw.wave/v1
wave_id: FE-W00
track_id: FE-STABILITY-2026-07
title: Team Web Console intake and baseline
revision: 1
plan_status: approved
wave_state: active
approved_by: user-directive-2026-07-26
depends_on: []
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - docs/**
  - test-plans/**
  - non-production diagnostic fixtures
product_code_changes_allowed: false
---

# FE-W00 — 前台异常收件、复现与基线

## 目标

把“前台页面很多功能使用异常”的聚合报告拆成可重复、可证伪、可排序的
独立 Issue，并冻结前端调用、Gateway 注册、权限、返回结构和部署拓扑基线。

本 Wave 完成前不修复产品代码。

## 权威输入

- 用户于 2026-07-26 报告 Team Web Console 多项功能使用异常，并要求所有逐步计划先进入 Wave 文档。
- 当前源码基线标记为 Team Runtime `0.7.0`。
- [`TEAM_WEB_CONSOLE_CN.md`](../../../TEAM_WEB_CONSOLE_CN.md) 记录预期页面和安全边界，但“代码存在”不再作为“功能可用”的证据。
- [`issue-register.md`](../../issue-register.md) 中的 `FE-ISSUE-001` 是当前唯一用户报告。
- 当前目录是源码归档展开结果，缺少可验证 Git base commit；进入任何代码 Wave 前必须在真实仓库冻结 commit。

## 入口门禁

- [x] 用户报告已登记为稳定 Issue。
- [x] W00 明确禁止产品代码变更。
- [x] 九个页面和共享基础设施源码可读取。
- [ ] 已确定可运行的 Go 1.25.5、Gateway、浏览器和测试项目环境。
- [ ] 已冻结实际 Git base commit、部署配置摘要和前端构建哈希。

未完成项属于 W00 Step，不阻止文档与诊断开始。

## 范围

### 包含

- 登录、恢复、退出、CSRF、WebSocket、hash 导航、项目/Topic 上下文；
- 总览、对话、规格、记忆、审批、开发、团队、进度、Harness 九个页面；
- 所有页面 loader、命令、通知、本地状态和剪贴板动作；
- authorized project A 与 unauthorized project B；
- 直连与同源反向代理；
- loading、empty、denied、error、disconnected、stale/conflict；
- 真实 RPC 返回与 TypeScript 类型、页面语义之间的对照；
- 问题拆分、严重度、依赖归属和后续 Wave 分配。

### 不包含

- React、CSS、Gateway、TeamControl、Harness、Catalog、Ouroboros 或 Orchestrator Lite 修复；
- 新功能、视觉改版或产品范围扩张；
- 恢复 Obsidian 为默认控制面；
- 通过放宽 RBAC、CSRF、职责分离或 DoneGate 来“修复”页面；
- 在未授权生产数据上执行高风险命令。

## 当前覆盖矩阵

| Surface | 查询/恢复 | 命令、事件或本地动作 | W00 必测边界 |
|---|---|---|---|
| 登录与 Shell | `GET /auth/session` | login、logout、`/ws`、项目/Topic/Reviewer、hash 导航 | 401/403、过期、撤销、同源代理、重连、切项目 |
| 总览 | Work、Issue、Runner、Policy、Dev、Trace、Knowledge 七路查询 | retry | 正常、空、单路失败、拒绝、项目一致性 |
| 对话 | 当前没有历史 loader | `agent`、`chat.event` | 顺序、重复、失败、断线、切项目/Topic、刷新语义 |
| 规格 | `ouroboros.sessions` | start、answer、reassess、crystallize、compile、evaluate、evolve | 状态按钮、长超时、空答案、并发变化、任务关联 |
| 记忆 | Catalog status/list/search | search | active/pending、来源、过期、共享记忆、项目隔离 |
| 审批 | Knowledge、Catalog、Harness、Dev、Ouroboros 五组 loader | 知识/记忆/Seed/四审/freeze/accept/revise/Harness 治理 | Reviewer、理由/反方/证据、职责分离、CAS、重复提交 |
| 开发 | `dev.tasks` | enqueue、revise、copy task ID | revision、按钮状态、幂等、剪贴板、Evidence/DoneGate |
| 团队 | Member、Work、Issue、Runner、Policy、Docs、Components 七路查询 | 无服务端命令 | 10 人、可选字段、投影语义、项目 RBAC |
| 进度 | Dev、Trace | 无服务端命令 | 统计语义、异常轨迹、长内容、时间、空数据 |
| Harness | status、experiments、traces | approve、promote、reject、rollback、本地保留 | 项目边界、状态分支、角色、回滚刷新 |

## 待验证契约观察

以下来自静态阅读，只能作为复现线索，不能标记为 Bug 或根因：

| Observation ID | 观察 | 必须怎样验证 |
|---|---|---|
| `FE-OBS-001` | 前端 WorkItem/Issue 的 status、priority 联合类型与 TeamControl 枚举不完全同构 | 保存真实 RPC 夹具，验证页面分支、tone、统计和动作是否错误 |
| `FE-OBS-002` | Ouroboros `assessment.overall` 后端语义为 ambiguity score；页面存在“歧义度/规格明确度”两种表述 | 用边界分值验证数值方向、标签和可访问名称 |
| `FE-OBS-003` | 多页面使用 `Promise.all`，单路失败会进入整页 ErrorState | 逐路注入 401/403/404/500/超时，依据产品契约判断整页失败是否正确 |
| `FE-OBS-004` | 项目切换加载期间可能保留旧 loader 数据；Chat/Memory 等还有页面本地状态 | 验证切项目和 Topic 时是否短暂或持续显示旧项目数据 |
| `FE-OBS-005` | RPC 401 会清除 TeamClient 内部 session，Provider 是否同步退出需运行确认 | 让会话服务端过期，观察页面与后续请求 |
| `FE-OBS-006` | 全局搜索与通知控件在源码中没有可见动作绑定 | 对照冻结产品范围，判断是占位、缺失能力还是应隐藏 |
| `FE-OBS-007` | Chat 没有历史查询 loader | 验证“共享会话”是否承诺刷新恢复历史 |
| `FE-OBS-008` | 部分 Harness/Knowledge loader 未显式传左侧项目 ID | 在多项目成员下验证服务绑定项目与选择器语义 |
| `FE-OBS-009` | UI package 当前只有 build 脚本，没有自动化前端测试入口 | 记录现有 Gateway 测试和浏览器测试覆盖缺口 |

## 测试夹具矩阵

W00 必须冻结下列维度，避免用单一管理员/空项目误判：

- 项目角色：owner、maintainer、developer、reviewer、viewer；
- 项目：authorized A、unauthorized B、shared `*`；
- 数据量：empty、normal、large/long text；
- Dev：review_pending、frozen、running、repair_pending、awaiting_acceptance、done、failed；
- Ouroboros：interviewing、clarification_required、seed_ready、approved、compiled、evaluated；
- Harness：draft、validated、human_approved、active、rejected、rolled_back；
- Runner：online、busy、draining、offline；
- 会话：fresh、expired、revoked、disconnected/reconnected；
- 拓扑：直接访问、Caddy 同源反向代理；
- 浏览器：桌面 Chromium、移动视口；其他浏览器在 W04 扩展。

所有写操作只在一次性测试项目和测试 principal 中执行。

## 分步计划

| Step ID | 前置 | 计划动作 | 输出 | 验证 | 状态 |
|---|---|---|---|---|---|
| `FE-W00-S01` | 用户报告 | 冻结 runtime、Git base、UI 构建、配置摘要、浏览器和代理环境 | baseline manifest | checksum 与可重建性 | `planned` |
| `FE-W00-S02` | S01 | 对照前端 16 个查询 RPC、23 个命令 RPC、3 个 session HTTP 操作和 `chat.event` 与 Gateway 注册/RBAC | contract matrix | 每项都有 handler、参数、返回和权限来源 | `planned` |
| `FE-W00-S03` | S01 | 建立脱敏、一次性项目/角色/状态夹具 | fixture manifest | A/B 项目隔离与重置可重复 | `planned` |
| `FE-W00-S04` | S02–S03 | 验证登录、Shell 与所有页面读取路径 | read reproduction bundles | success/empty/denied/error 覆盖 | `planned` |
| `FE-W00-S05` | S02–S03 | 在一次性项目验证全部命令和通知路径 | command reproduction bundles | 状态机、角色、重复提交和刷新 | `planned` |
| `FE-W00-S06` | S04–S05 | 把聚合报告拆为独立 Issue，标严重度和依赖 Wave | issue register revision | 每项有期望、实际、步骤和证据 | `planned` |
| `FE-W00-S07` | S06 | 根据证据复核 W01–W05 范围，形成必要的 Plan r002 或确认 r001 | Wave decision | 无未登记修复步骤 | `planned` |

任何 Step 发现产品代码需要改变时，只登记 Issue 和目标 Wave；W00 不实施。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W00-001` | coverage matrix | 每个 Surface 有环境、用例和负责人 | `planned` |
| `FE-EVID-W00-002` | reproduction bundles | 每个具体 Issue 有可重复证据 | `planned` |
| `FE-EVID-W00-003` | contract baseline | 前端调用与 Gateway/RBAC/返回结构对应 | `planned` |
| `FE-EVID-W00-004` | fixture reset | 测试项目可重复建立、清理且无真实数据 | `planned` |

## 风险与停止条件

| 风险 | 停止信号 | 处理 |
|---|---|---|
| 泄露凭据或跨项目数据 | HAR/日志/截图含秘密或 B 项目内容 | 立即停止、销毁证据、轮换凭据并登记安全事件 |
| 诊断动作改变真实状态 | 非一次性项目发生 mutation | 停止测试，从备份恢复并记录影响 |
| 把静态观察写成根因 | 无真实复现却创建修复 Task | 拒绝 Task，Issue 保持 `unverified` |
| 环境不可重建 | baseline 无 commit/hash/config | Wave 进入 `blocked`，不进入 W01 |

## 退出门禁

- [ ] 聚合报告已拆成独立 Issue，或有证据证明某个表面无异常。
- [ ] 每个具体 Issue 都有环境、步骤、期望、实际和脱敏证据。
- [ ] 16 个查询、23 个命令、3 个 session HTTP 操作和通知覆盖均有结论。
- [ ] 前端类型、真实 JSON、RBAC 和项目归属基线已冻结。
- [ ] W01–W05 的问题归属和依赖顺序已依据证据复核。
- [ ] 没有任何产品代码变更混入 W00。
- [ ] 独立复核者确认 W00 Evidence。
- [ ] Registry、Issue、Decision、Evidence 和 Journal 一致。

退出门禁未全部通过时，`FE-W01` 不得转为 `active`。
