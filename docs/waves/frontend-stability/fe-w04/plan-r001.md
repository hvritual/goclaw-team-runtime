---
schema: goclaw.wave/v1
wave_id: FE-W04
track_id: FE-STABILITY-2026-07
title: UX resilience and accessibility
revision: 1
plan_status: draft
wave_state: planned
depends_on:
  - FE-W03
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - shared UI primitives and styles
  - explicit state rendering
  - responsive and accessibility fixes
  - directly related tests and documentation
product_code_changes_allowed: true
---

# FE-W04 — 状态韧性、响应式与可访问性

## 目标

在读取和命令契约稳定后，统一九个页面的 loading、empty、denied、error、
disconnected、blocked、stale/conflict 状态，并验证桌面/移动、键盘、焦点、
读屏语义、长文本和表格溢出。

## 入口门禁

- [ ] `FE-W03` 为 `complete`。
- [ ] W04 Issue 都是 UI 状态/交互问题，不掩盖未解决契约 Bug。
- [ ] 已冻结目标浏览器、视口和可访问性标准。
- [ ] 不包含视觉品牌重设计或新功能。

## 分步计划

| Step ID | 前置 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `FE-W04-S01` | W03 | 建立共享状态组件和页面状态覆盖表 | component/semantic tests | `planned` |
| `FE-W04-S02` | S01 | 修复 denied 与 empty、error 与 disconnected 的可辨识性 | role/status/alert assertions | `planned` |
| `FE-W04-S03` | S01 | 修复键盘顺序、焦点恢复、菜单/表单标签和 busy 状态 | keyboard + accessibility scan | `planned` |
| `FE-W04-S04` | S01 | 修复 320/390/768/1440 视口、侧栏、表格、长文本和触控 | responsive screenshots | `planned` |
| `FE-W04-S05` | S02–S04 | 验证 copy、超时、重试、刷新、项目切换和断线恢复反馈 | browser interaction matrix | `planned` |
| `FE-W04-S06` | S05 | 独立视觉与可访问性复核 | evidence review | `planned` |

## UX 不变量

- denied 不能展示为空数据。
- loading 期间不能误导用户认为旧项目数据属于新项目。
- 错误文案不泄露 Token、绝对路径或跨项目资源。
- busy 控件必须防重复提交并向辅助技术暴露状态。
- 移动抽屉打开后可完整关闭，背景不可误操作。
- 不使用颜色作为唯一状态信号。
- prefers-reduced-motion 下不依赖动画表达完成。

## 浏览器矩阵

最低覆盖：

- Chromium 桌面 1440×900；
- Chromium 移动 390×844；
- 320 px 最小宽度；
- 键盘-only；
- reduced motion；
- 200% zoom；
- 长中文/英文、超长 ID 和空字段。

Firefox/Safari 若无法在当前 CI 自动化，必须进入人工证据而不是假定通过。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W04-001` | accessibility | 无阻断级名称、焦点、角色和对比问题 | `planned` |
| `FE-EVID-W04-002` | responsive | 关键视口无页面级横向溢出和不可达控件 | `planned` |
| `FE-EVID-W04-003` | state matrix | 九页全部状态可区分且可恢复 | `planned` |
| `FE-EVID-W04-004` | visual review | 与冻结设计方向一致，无意外回归 | `planned` |

## 风险与回滚

| 风险 | 触发信号 | 回滚 |
|---|---|---|
| 共享组件改动回归所有页面 | 多页面布局/动作同时异常 | 按组件最小提交回滚 |
| 只修视觉掩盖契约问题 | 状态看似正常但 RPC 仍错误 | Issue 退回所属 W01–W03 |
| 可访问性修复改变业务动作 | 键盘触发重复 mutation | 回滚交互变更并补 W03 回归 |

## 退出门禁

- [ ] 九页全部显式状态通过。
- [ ] 键盘、焦点、标签、busy 和 alert 语义通过。
- [ ] 320/390/768/1440、200% zoom 和长文本通过。
- [ ] 无颜色唯一信号和阻断级可访问性问题。
- [ ] 没有借 UX Wave 修改 RPC、权限或状态机语义。
