---
schema: goclaw.wave/v1
wave_id: FE-W05
track_id: FE-STABILITY-2026-07
title: Integrated verification and release readiness
revision: 1
plan_status: draft
wave_state: planned
depends_on:
  - FE-W04
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - tests
  - release scripts
  - deployment examples
  - documentation
  - release-blocking fixes returned to their owning Wave
product_code_changes_allowed: false
---

# FE-W05 — 集成验证、安全回归与发布就绪

## 目标

停止新增功能和直接产品修复，用真实嵌入构建完成浏览器 E2E、Gateway/Go、
项目隔离、CSP、反向代理、归档和回滚验证，并形成可以审计的发布结论。

若发现产品缺陷，必须退回其责任 Wave，新建 Plan revision 和修复 Task；
不得在 W05 顺手修复。

## 入口门禁

- [ ] `FE-W04` 为 `complete`。
- [ ] 所有纳入 Issue 已 `fixed`，等待独立验证。
- [ ] Go 1.25.5、Node、浏览器和 Linux 构建环境可用。
- [ ] 发布配置、一次性测试项目和回滚备份已冻结。
- [ ] 候选 release version 与 changelog 已准备。
- [ ] `FE-EVID-W01-011` 已通过；W05 只复核它仍适用于候选 release，记录不得
  回显原值。

## 分步计划

| Step ID | 前置 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `FE-W05-S01` | W04 | 从干净环境重建嵌入 Team Console 和 Linux binary | reproducible hashes | `planned` |
| `FE-W05-S02` | S01 | 运行 UI build/tests、Go test/race/vet 和 Gateway contract tests | deterministic suite | `planned` |
| `FE-W05-S03` | S01 | 对真实嵌入服务执行登录、九页读取和全部命令 E2E | browser E2E | `planned` |
| `FE-W05-S04` | S03 | 执行跨项目、角色、CSRF、Origin、CSP、会话撤销与职责分离回归 | security matrix | `planned` |
| `FE-W05-S05` | S03 | 执行断线、进程重启、代理、备份恢复和回滚演练 | operational evidence | `planned` |
| `FE-W05-S08` | `FE-EVID-W01-011` | 独立复核 owner closure 仍适用于候选 release，且归档不含当前 material | `FE-EVID-W05-006` | `planned` |
| `FE-W05-S06` | S02–S05、S08 | 构建发行包，检查凭据、成员、权限和 SHA-256 | release audit | `planned` |
| `FE-W05-S07` | S06 | 独立 reviewer 给出 release/no-release 结论 | exit review | `planned` |

## 必需验证

- `npm` 的 TypeScript、单元、组件和浏览器测试；
- `go test -race`、`go vet`、格式与配置校验；
- Team Console 嵌入资源与 CSP；
- 五类项目角色和 unauthorized project B；
- session fresh/expired/revoked，WebSocket 断线/重连；
- 16 个查询、23 个命令和 `chat.event`；
- 320/390/768/1440 与键盘/读屏语义；
- source/binary archive 凭据与危险成员扫描；
- Obsidian 不安装时全部强制控制面仍可工作；
- 可选 Obsidian 适配器只做兼容构建，不成为 release 前置。

## 发布声明分级

| 状态 | 允许声明 |
|---|---|
| `implemented_unverified` | 代码存在，但未通过完整 Wave 证据 |
| `verified` | 候选构建在受控环境通过 W05 |
| `released` | 发行包、哈希和部署/回滚证据均完成 |

不能再用“页面存在”替代 `verified`。

## 验证与证据计划

| Evidence ID | 类型 | 通过条件 | 状态 |
|---|---|---|---|
| `FE-EVID-W05-001` | full test suite | UI/Go/contract 全部通过 | `planned` |
| `FE-EVID-W05-002` | embedded browser E2E | 登录、九页、命令、断线通过 | `planned` |
| `FE-EVID-W05-003` | security | RBAC、CSRF、Origin、CSP、职责分离通过 | `planned` |
| `FE-EVID-W05-004` | release audit | archives、permissions、credentials、SHA 通过 | `planned` |
| `FE-EVID-W05-005` | rollback | 恢复旧 binary/config/data 可执行 | `planned` |
| `FE-EVID-W05-006` | credential owner revalidation | `FE-EVID-W01-011` 仍有效，候选归档不含当前 material | `planned` |

## 停止条件

- 任一 S0/S1 回归；
- 跨项目数据或凭据出现在证据；
- Go race、权限矩阵、CSP 或职责分离失败；
- 嵌入 UI 与已验证静态 UI 不是同一构建；
- 归档扫描或回滚演练失败；
- 有 blocking Issue 未说明。
- `FE-EVID-W01-011` 缺失，或 W05 复核发现它不再适用于候选 release。

## 退出门禁

- [ ] 所有确定性、浏览器、安全和运维证据通过。
- [ ] 每个纳入 Issue 由非修复者标为 `verified`。
- [ ] 发行包哈希、内容和构建基线一致。
- [ ] 文档只声明证据支持的成熟度。
- [ ] 回滚演练通过。
- [ ] 独立 reviewer 明确批准 release。
