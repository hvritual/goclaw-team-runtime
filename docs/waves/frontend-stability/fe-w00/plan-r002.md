---
schema: goclaw.wave/v1
wave_id: FE-W00
track_id: FE-STABILITY-2026-07
title: Team Web Console executable baseline and first issue split
revision: 2
plan_status: approved
wave_state: active
approved_by: user-directive-2026-07-26-start-repair
supersedes:
  - plan-r001
depends_on: []
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - docs/**
  - test-plans/**
  - non-production diagnostic fixtures
product_code_changes_allowed: false
---

# FE-W00 r002 — 可执行基线与首批问题拆分

## 修订原因

`plan-r001` 假定 W00 必须先完成九页、43 个接口与全部角色状态的完整浏览器
矩阵，再允许任何修复。执行后出现两个新事实：

1. Cloud Browser 安全策略拒绝本地运行时，完整页面矩阵暂时不可执行；
2. Go、真实 Vite proxy 与协议探针已经能够确定性复现会话基础层问题。

继续等待完整浏览器矩阵会阻止已经证实且可独立回归的基础修复。r002 将 W00
退出条件调整为“冻结可执行基线并拆出首批基础 Issue”。聚合问题
`FE-ISSUE-001` 保持开启；W01–W05 在每个 Step 实施前继续执行对应表面的
失败测试与复现，不把未测页面描述为正常。

本修订不放宽 W00 的产品代码禁令，也不降低 W01 的浏览器退出门禁。

## 已冻结基线

- source base：`b288564361fac4f09d65e2a6a7ff80362a5cc12e`
- release content：Team Runtime `0.7.0`
- Go：官方 `1.25.5`
- Node/npm：`24.14.0` / `11.9.0`
- UI build aggregate：
  `08171d6fa7c60f0627b922172de86814cf7bacd819ec70f8adfaaf71a475d3e8`
- 产品 Go build：通过
- Gateway test package：存在 `FE-ISSUE-002` 编译阻断
- 浏览器：Cloud Browser localhost 被拒；不得声明页面/视觉通过

## 纳入的具体问题

| Issue | 结论 | 责任 Wave |
|---|---|---|
| `FE-ISSUE-002` | Gateway session 测试使用过期返回值签名，测试包无法编译 | FE-W01 S01 |
| `FE-ISSUE-003` | Vite `/auth.changeOrigin=true` 破坏严格同源登录 | FE-W01 S01–S02 |
| `FE-ISSUE-004` | DEV TeamClient 绕过 `/ws` proxy，跨端口直连被安全策略拒绝 | FE-W01 S01–S03 |

## 执行步骤

| Step ID | 动作 | 证据/输出 | 状态 |
|---|---|---|---|
| `FE-W00-S01` | 冻结独立 Git、官方 Go、Node、build 与产物哈希 | `FE-EVID-W00-007` | complete |
| `FE-W00-S02` | 盘点 43 个前端操作与 Gateway 路径 | `FE-EVID-W00-003` | complete-static |
| `FE-W00-S03` | 运行一次性 Vite HTTP/WS 协议探针 | `FE-EVID-W00-007` | complete |
| `FE-W00-S04` | 尝试 Cloud Browser 本地运行时 | Browser localhost policy rejection | blocked-external |
| `FE-W00-S06` | 拆分首批可执行基础 Issue | `FE-ISSUE-002`–`004` | complete |
| `FE-W00-S07` | 修订并独立复核 W01 r002，冻结最小任务范围 | W01 `plan-r002` | pending-review |

## W01 迁移门禁

- [x] 0.7.0 产品源码有本地 Git commit。
- [x] Go 与 UI 产品 build 可执行。
- [x] 纳入 W01 的 Issue 均有稳定期望、实际和脱敏证据。
- [x] 修复方向不要求放宽 Origin、CSRF、Cookie、身份或 RBAC。
- [x] W01 r002 明确新增测试、允许文件、回滚和停止条件。
- [ ] 独立 Reviewer 确认 Issue、范围与失败测试设计。
- [ ] Registry 与 Journal 完成唯一 active Wave 切换。

## 退出门禁

- [x] 权威源码、commit、工具链、构建和测试阻断已冻结。
- [x] 首批会话基础问题已拆成独立 Issue。
- [x] 所有产品文件在 W00 保持不变。
- [x] 未验证的九页与命令矩阵继续由 `FE-ISSUE-001` 跟踪。
- [ ] 独立 Reviewer 确认 `FE-EVID-W00-007` 与 W01 r002。
- [ ] Registry、Issue、Decision、Evidence 与 Journal 一致。

只有最后两项完成后，W00 才能标记 complete，W01 才能成为唯一 active Wave。

