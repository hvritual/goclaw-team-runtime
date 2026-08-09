---
schema: goclaw.wave/v1
wave_id: TCF-W01
track_id: TEAM-CONTROL-FRONTEND-2026-08
title: Multica-referenced Team Control Web Console
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-08-09-team-control-first
owner: Codex root agent
issue_id: TC-UI-ISSUE-001
task_id: TC-UI-FRONTEND-001
depends_on:
  - MVP-W00
created_at: 2026-08-09
updated_at: 2026-08-09
product_code_changes_allowed: true
---

# TCF-W01 r001 — Team Control 完整前端

本 Wave 响应 2026-08-09 的明确路线调整：先完成团队日常研发使用的
Team Control Web Console，再继续 Team Control/Runner 独立应用边界、预算和
Context Compiler。`TR-W00` 已有实现与文档保留，但当前执行顺序被本 Wave
替代，不将未验收步骤伪造为完成。

视觉和交互参考 `multica-ai/multica` 的工作区侧栏、分组导航、紧凑集合页、
详情工作面、显式空/错状态和响应式行为；不复制品牌、业务文案或其后端模型。

## 目标

把既有九个孤立页面收敛为能支撑团队日常研发的三个闭环：

1. 需求/规格 → 方案/决策 → WorkItem → 开发任务 → Evidence/DoneGate；
2. Bug/风险 → 分级/负责人 → WorkItem → 修复 → 验证/关闭；
3. PR/自动 Review/质量证据 → 知识候选 → 文档/组件/记忆复用。

同时补齐 Team Control 已有真实 RPC 的项目级管理入口：成员与容量、Repository、
Runner、Artifact/Correlation、Document、Component 和 Policy。页面只读写中央
Gateway，不建立本地任务事实库，不提供 mock/fallback 数据。

## 不做

- 不改变 Go TeamControl、Runner、Orchestrator Lite 的权威边界和状态机；
- 不新增未经后端实现的成功路径；
- 不在浏览器持久化 Gateway、Team 或 Reviewer Token；
- 不绕过 RBAC、CSRF、四审、Freeze、独立 Review 或 DoneGate；
- 不处理外部 credential owner、ptrace 或发布环境阻断。

## Steps

| Step | 内容 | 允许产物 |
|---|---|---|
| `TCF-W01-S01` | 冻结 IA、真实 RPC、视觉参考与状态合同 | Plan、Registry、contract tests |
| `TCF-W01-S02` | Multica 式 Shell、分组导航、项目上下文、命令搜索 | AppShell、tokens、responsive tests |
| `TCF-W01-S03` | 需求/方案/任务工作台 | Spec、WorkItem、Development 组合与真实 mutation |
| `TCF-W01-S04` | Bug/风险/任务质量中心 | type-aware issue create/filter/detail/transition/assignment |
| `TCF-W01-S05` | PR/Review/证据与知识复利中心 | DevTask、Artifact、Correlation、Document/Component views |
| `TCF-W01-S06` | 成员、Runner、Repository、Policy 管理 | project-scoped administration views |
| `TCF-W01-S07` | 确定性、桌面/移动和浏览器回归 | tests、build、screenshots、Evidence、journal |

## 真实 RPC 合同

读路径至少覆盖：`project.list/get/members`、`repository.list`、
`work.items`、`issue.list`、`assignment.list`、`runner.list/tasks`、
`dev.tasks`、`artifact.list`、`correlation.list`、`document.list`、
`component.list`、`policy.list/status`、`memory.catalog.*`、
`harness.*`、`ouroboros.*`。

写路径仅调用已注册方法，并在 UI 显示服务端拒绝：`issue.create/transition`、
`work.create/transition`、`assignment.create/release`、`repository.create`、
`document.register`、`component.register`、`policy.put`、`dev.task.*` 和现有
Governance 方法。

## Acceptance criteria

- [x] 三个研发闭环都有明确入口、来源关系、负责人、状态、证据和下一动作；
- [x] Bug 与 Risk 使用同一 Issue 聚合但保留不同类型和统计，不再硬编码 Bug；
- [x] Review 中心可看到待验收任务、DoneGate、PR/CI/证据 Artifact 和关联关系；
- [x] 成员、Runner、仓库、文档、组件、Policy 均从当前项目真实 RPC 加载；
- [x] mutation 成功后只通过 RPC 重载权威状态，不把本地状态当作完成事实；
- [ ] loading、empty、denied/error、disconnected、pending、blocked、stale/conflict 显式呈现；
- [x] Team/Gateway/Reviewer Token 不进入 Web Storage、URL、日志和页面可复制文本；
- [ ] TypeScript build 与 Node contract tests 已通过；桌面与移动渲染回归待可访问预览环境复核；
- [ ] 独立代码、安全与文档复核 `P0=0/P1=0` 后才可完成 Wave。

## 风险与回滚

- RPC 形状与 UI 类型不一致：失败关闭并保留原入口，不提供假数据兜底；
- 导航改造造成关键操作不可达：回滚 Shell，保留页面逻辑；
- 大列表阻塞交互：默认筛选、分段呈现并延迟非当前工作区加载；
- 高风险写操作误触：显式确认、服务端 RBAC、revision/CAS 参数优先；
- 回滚单位为本 Wave 的 UI/文档 commit，不改中央存储格式。
