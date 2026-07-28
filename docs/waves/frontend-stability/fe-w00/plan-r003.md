---
schema: goclaw.wave/v1
wave_id: FE-W00
track_id: FE-STABILITY-2026-07
title: Team Web Console executable baseline and first issue split
revision: 3
plan_status: approved
wave_state: active
approved_by: user-directive-2026-07-26-start-repair
supersedes:
  - plan-r002
depends_on: []
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - docs/**
  - test-plans/**
  - non-production diagnostic fixtures
product_code_changes_allowed: false
---

# FE-W00 r003 — 可执行基线与首批问题拆分

## 修订原因

独立 Reviewer 批准了 `FE-EVID-W00-007` 对 `FE-ISSUE-002`–`004` 的根因
结论，但拒绝 r002 的 Wave 切换：

- r002 错把原 `FE-W00-S03` 从“角色/项目夹具”改成“Vite 协议探针”；
- W01 r002 缺少完整入口、影响、回滚、真实 WS proxy 和安全负例；
- Task base、范围门禁和 Registry 尚未冻结。

r003 保留 r001 的稳定 Step 含义，新增 Step ID 记录后来发生的 Git、工具链和
协议工作。W00 仍禁止产品代码。

## 权威基线

- Task/source base：
  `b288564361fac4f09d65e2a6a7ff80362a5cc12e`
- release content：Team Runtime `0.7.0`
- Go：官方 `1.25.5`
- Node/npm：`24.14.0` / `11.9.0`
- UI build aggregate：
  `08171d6fa7c60f0627b922172de86814cf7bacd819ec70f8adfaaf71a475d3e8`
- 产品 Go build：通过
- 全仓测试编译：仅 `FE-ISSUE-002` 的两个 Gateway 旧签名调用失败
- Browser：Cloud Browser localhost 被拒；页面/视觉未验收

## 稳定 Step 处理

| Step ID | 原始含义 | r003 结论 | 后续责任 |
|---|---|---|---|
| `FE-W00-S01` | runtime/Git/build/config/browser baseline | complete，Browser 可达性作为外部边界记录 | Browser gate 进入 W01/W04/W05 |
| `FE-W00-S02` | 43 个前端操作与 Gateway/RBAC 契约 | complete-static；运行返回与权限仍未全验 | W02/W03 |
| `FE-W00-S03` | 一次性项目/角色/状态夹具 | deferred，未改变原含义 | W02/W03 在各自写操作前建立 |
| `FE-W00-S04` | 登录、Shell 与页面读取运行验证 | partial；基础 transport 已协议复现，其余 deferred | W01/W02 |
| `FE-W00-S05` | 命令与通知运行验证 | deferred | W03 |
| `FE-W00-S06` | 拆分全部聚合问题 | partial；首批 3 项已拆，聚合 Issue 保持开启 | W01–W05 |
| `FE-W00-S07` | 复核后续 Wave 范围 | complete for first slice | `FE-W01 plan-r003` |

## 新增 Step

| Step ID | 动作 | 证据 | 状态 |
|---|---|---|---|
| `FE-W00-S08` | 在 0.7.0 审阅副本建立独立 Git、官方 Go 与可重建 build | `FE-EVID-W00-007` | complete |
| `FE-W00-S09` | 运行真实 Vite HTTP/WS proxy 与 TeamClient SSR transport 探针 | `FE-EVID-W00-007` | complete |
| `FE-W00-S10` | 将首批基础问题拆为 `FE-ISSUE-002`–`004` 并绑定 W01 r003 | Issue register | complete |
| `FE-W00-S11` | 独立复核 W00 Evidence 与 W01 r003 | Reviewer record | pending |
| `FE-W00-S12` | 原子切换 Registry、Journal、Evidence 与 Task 状态 | consistency validation | pending |

## 渐进迁移决策

`FE-ISSUE-001` 不关闭。W00 不再把 Browser 当前不可达的完整九页矩阵作为
首个确定性 transport 修复的无限期前置条件，但每个后续 Wave 必须在修改前
为其纳入问题建立失败证据。W01 的 browser exit gate 仍是硬门禁；Browser
不可达时可以实现并验证确定性修复，但 W01 只能保持 active/blocked，不能
标记 complete。

## W01 迁移门禁

- [x] 0.7.0 产品源码有精确 Git base。
- [x] 官方 Go、UI build 与产品 Go build 可重建。
- [x] `FE-ISSUE-002`–`004` 已获独立 Reviewer 批准为 root-caused。
- [x] 修复方向不放宽 Origin、CSRF、Cookie、身份或 RBAC。
- [x] W01 r003 包含入口、影响、先失败测试、安全负例、风险与回滚。
- [ ] 独立 Reviewer 批准 W01 r003 可执行。
- [ ] Registry、Journal、Evidence、Issue 与 Task 原子切换。

## 退出门禁

- [x] 权威源码、commit、工具链、构建和测试阻断已冻结。
- [x] 首批 transport 问题已拆分，未验证范围继续由聚合 Issue 跟踪。
- [x] 原 S03–S06 的未完成部分保留原含义并明确后续责任。
- [x] W00 没有修改产品代码。
- [ ] W01 r003 通过独立复核。
- [ ] Registry 与全部索引一致，且只有 W01 active。

