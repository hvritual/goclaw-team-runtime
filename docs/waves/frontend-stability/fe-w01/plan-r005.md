---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Session transport, dashboard shell, and deterministic full-Go gate
revision: 5
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review
supersedes:
  - plan-r004
depends_on:
  - FE-W00
created_at: 2026-07-26
updated_at: 2026-07-26
allowed_change_scope:
  - ui/src/team/client.ts
  - ui/vite.config.ts
  - ui/package.json
  - ui/tests/team-transport.test.mjs
  - gateway/team_runtime_test.go
  - gateway/web_sessions_test.go
  - gateway/ui.go
  - gateway/server_auth_test.go
  - channels/weworkwsbot_test.go
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r005 — Transport、Dashboard Shell 与确定性 Full-Go Gate

## 目标与冻结任务候选

完整保留 r004 的 transport、dashboard shell、Origin/CSRF/身份/RBAC、安全头
和缓存合同；新增一个测试文件范围，移除全仓 Gate 中真实网络、十分钟 sleep
和 credential-shaped literal。生产 `channels` 实现不变。

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `3` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S07`、`FE-W01-S08`、`FE-W01-S04` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005`、`FE-ISSUE-006`、`FE-ISSUE-007` |
| Assignee | Codex root agent |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Task Base | 待 r005 获批后的 docs-only activation commit；R3 创建时在 Journal 冻结精确 SHA |
| Plan | `FE-W01 plan-r005` |
| Policy bundle | `wave-governance-v1` |
| Auto product commit | 禁止；确定性验证与独立验收后再决定 |

本 revision 不覆盖 `channels/weworkwsbot.go`、外部 WeWork 管理面、Git 历史
重写、项目/Topic/hash、业务 loader、页面命令、认证 handler 或 Browser
策略绕行。

## r004 迁移规则

r004/R2 worktree 中的 transport、shell tests 和最小 `ui.go` patch 保持未提交。
r005 获批后：

1. 将 Plan、Registry、Track、Issue、Decision、Evidence 和 Journal 的原子切换
   形成 docs-only activation commit；
2. 从该 commit 创建 `repair/fe-w01-transport-r3`，创建时
   `HEAD == Task Base`；
3. 在 R3 先以 docs-only commit 冻结 Task Base、branch、worktree、Plan SHA
   和完整任务元组；
4. 只迁移 r005 allowlist 内 R2 patch，不复用 R2 `node_modules` 或进程；
5. 只以 SHA-256 静态确认旧 test blob 与 `FE-EVID-W01-009` 一致；不得再次
   执行旧 test，因为它会携 credential-shaped material 调用真实网络路径；
6. 立即把 test 改为 synthetic constructor-only，先通过无输出 source gate，
   然后才允许运行 channels 或其他 Go tests；
7. R3 重新执行全部 green 验证；只继承已复核的 red Evidence，不继承 R2
   green 结论。

R1、R2 与旧 0.6.0 worktree 均保留只读，不 reset、不删除。

## 入口门禁

- [x] `FE-ISSUE-002`–`007` 均有确定性 reproduction/root-cause Evidence。
- [x] `channels/weworkwsbot_test.go` 与两个冻结 base 哈希一致。
- [x] r004 shell red→green 已完成目标测试，但 S04 全仓 Gate 未通过。
- [x] S07 red Evidence、成功条件、安全边界和回滚已定义。
- [x] Wave Reviewer 批准 r004→r005、稳定 Step 和 Task revision 迁移。
- [x] Security Reviewer 批准 credential handling、历史/轮换边界和无网络测试。
- [x] Registry、README、Track、Issue、Decision、Evidence 与 Journal 原子切换。
- [ ] R3 worktree 与 Task Base 精确 SHA 已冻结。

入口未全部通过前，不得修改 `channels/weworkwsbot_test.go`，不得把 R2 product
patch 迁移到 R3。

## 稳定 Step

| Step ID | 动作 | 先失败证据 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | 保留 r003 transport/session tests | R1 已记录 compile 与 transport red | R3 目标测试重验通过 | complete-in-r2; reverify-in-r3 |
| `FE-W01-S02` | 保留 Vite 同源 `/auth`、`/ws` proxy | auth 403 | R3 auth 204、WS 101 | complete-in-r2; reverify-in-r3 |
| `FE-W01-S03` | 保留 TeamClient 同源 `/ws` | direct `:28789` | R3 URL/protocol/token gate 通过 | complete-in-r2; reverify-in-r3 |
| `FE-W01-S06` | 保留 dashboard shell red/green 与最小实现 | slash 301 `./` | R3 GET/HEAD、安全头、缓存、asset、unknown 404 通过 | complete-in-r2; reverify-in-r3 |
| `FE-W01-S07` | 将 WeWork constructor test 改为无网络、无 secret、无 sleep 的确定性单测 | `FE-EVID-W01-009`：channels 30s timeout 与 credential-shaped literals；禁止在 R3 重跑旧 test | `FE-EVID-W01-010`：source gates 与 channels/race 快速通过，只含 synthetic placeholders | ready-after-task-base |
| `FE-W01-S08` | credential owner 提供撤销/轮换或从未有效的脱敏权威记录 | owner 未解析，外部有效性未知 | `FE-EVID-W01-011` passed；不回显原值 | blocked-external-owner |
| `FE-W01-S04` | 全 Gateway/race、全 Go、UI build、scope/lockfile Gate | r004 被 channels timeout 阻断 | 所有确定性 Gate 通过 | blocked-by-S07 |
| `FE-W01-S05` | 真实 Browser 登录、connected、刷新、退出 | localhost policy blocked | 页面级验证通过 | blocked-external |

S07 是新 Step，不改写 S04/S05；r004 的 S06 全部安全合同继续具有约束力。

## S07 测试合同

`channels/weworkwsbot_test.go` 必须：

1. 只使用 `test-bot-id`、`test-secret-id` 等明显 synthetic values；
2. 以 table-driven cases 断言缺 BotID、缺 SecretID 都返回 error；
3. 合法最小 config 返回非 nil channel，name/account ID、message bus 和内部
   maps/channels 初始化正确；
4. 空 URL 解析为非空默认 endpoint，ReconnectDelay 为 3、Heartbeat 为 30；
   test 不嵌入或比较真实 network endpoint literal；
5. 不调用 `Start`、`doConnect`、`Send` 或任何网络路径；
6. 不启动 test-owned goroutine，不使用 sleep/ticker，不等待外部消息；若使用
   `bus.NewMessageBus`，必须以 `t.Cleanup` 调用 `Close`，确保它拥有的 fanout
   goroutine 有界退出；
7. 单包命令在 30 秒 timeout 内通过；测试自身预期远低于一秒；
8. 不在 source、输出、Evidence 或 commit message 中复制旧 credential-shaped
   literals。

本 Step 不改变 `NewWeWorkWsBotChannel` 生产语义，也不增加 mock server 或新依赖。

## Credential 响应边界

- 当前源码中的非 synthetic、credential-shaped material 作为潜在凭据处理，
  有效性未知；本计划不声称已经证实有效凭据泄露；
- patch 只从新 HEAD 删除 material；raw deletion diff 与 Git history 必然仍含
  旧内容，属于受限敏感材料，不得输出到聊天、Evidence、日志或发布归档；
- 当前 tree、新增内容、日志、Evidence 和 commit message 不得含旧 material；
- 若值曾有效，外部撤销/轮换是 W01 与发布的安全阻断项；
- 本任务无权登录 WeWork、轮换密钥或改写 Git 历史；
- credential owner 当前未解析；必须由用户/组织指派有权管理该 WeWork
  credential 的负责人，代码修复不自动获得此外部权限；
- `FE-ISSUE-007` 的代码 hygiene green 后保持合法状态 `fixing`，并单独记录
  external security blocker；直到当前 W01 的 `FE-W01-S08` 获得 credential
  owner 的撤销/轮换记录，或权威证明从未有效；
- reviewer 和最终说明不得回显这些值，原始 patch 不进入普通 EvidencePackage。

## S07 执行顺序

1. 在未运行任何 Go test 前，确认
   `channels/weworkwsbot_test.go` SHA-256 仍为
   `2514948eb0a9fdee39c084ec0cde09eab2b144e2cf9a95511b562c8e4c01f01b`；
2. 不打开外部连接、不再次执行旧 test；立即实施 test-only patch；
3. 先运行下方两个无输出 source gates，任一非零即停止；
4. 再运行 `go test` 与 race；
5. 只用 `--name-only`、`--numstat` 和哈希做范围/审计，不显示 raw diff。

## r004 合同继承

以下 r004 合同原样保留：

- TeamClient 和真实 Vite `/auth` 204、`/ws` 101；
- strict Origin、Sec-Fetch-Site、CSRF、browser session、personal token、
  Gateway/Team 身份和 TeamGuard/RBAC；
- `/dashboard` 单次 canonical redirect；
- GET/HEAD shell 的 HTML MIME、Content-Length、完整 CSP、安全头与 no-store；
- 真实 hashed JS/CSS MIME、nosniff、immutable；
- unknown route 404、无 Location、无 shell marker；
- 不新增非 GET/HEAD 405，不修改旧 `DashboardHandler`。

## Evidence 映射

| Step | Evidence |
|---|---|
| `FE-W01-S01`–`FE-W01-S03` | `FE-EVID-W01-001`、`002`、`004`、`006` |
| `FE-W01-S06` | `FE-EVID-W01-007`、`008` |
| `FE-W01-S07` | `FE-EVID-W01-009`、`010` |
| `FE-W01-S08` | `FE-EVID-W01-011` |
| `FE-W01-S04` | `FE-EVID-W01-005` |
| `FE-W01-S05` | `FE-EVID-W01-003` |
| FE-W05 release revalidation | 复用 `FE-EVID-W01-011`，产生 `FE-EVID-W05-006` |

## 验证命令

```text
perl -ne 'if (/(BotID|SecretID):\s*"([^"]*)"/ && $2 !~ /^test-/) { exit 1 }' \
  channels/weworkwsbot_test.go
! rg -q '\.(Start|doConnect|Send|SendStream)\(|time\.(Sleep|NewTicker|After)\(|go[[:space:]]+func|gorilla/websocket|net/http|wss?://' \
  channels/weworkwsbot_test.go
go test ./channels -count=1 -v -timeout=30s
go test -race ./channels -count=1 -timeout=30s
npm --prefix ui run test:transport
npm --prefix ui run build
go test ./gateway -run 'TestTeamConsole' -count=1 -v
go test ./gateway -count=1
go test -race ./gateway -count=1
go test ./... -count=1 -timeout=5m
test -z "$(gofmt -l channels/weworkwsbot_test.go \
  gateway/team_runtime_test.go gateway/web_sessions_test.go \
  gateway/ui.go gateway/server_auth_test.go)"
git diff --check <TASK_BASE_SHA> >/dev/null
git diff --check 697f50e5f428769b75061dfd859d2549dd1c330d >/dev/null
git diff --name-only <TASK_BASE_SHA>
git diff --name-only 697f50e5f428769b75061dfd859d2549dd1c330d
git diff --numstat <TASK_BASE_SHA>
git diff --numstat 697f50e5f428769b75061dfd859d2549dd1c330d
git ls-files --others --exclude-standard
sha256sum ui/package-lock.json
```

Task Base、累计 base 与 untracked 三组路径必须合并、去重后精确匹配 10 项
allowlist。`package-lock.json` SHA 必须保持
`46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb`。

## 风险、停止与回滚

| 信号 | 动作 | 回滚 |
|---|---|---|
| 测试仍访问网络、sleep 或 hang | 停止 S04 | 不合并 R3；保留 Evidence，新建 revision |
| 旧 material 出现在当前 tree、新增内容、输出或 commit message | 按潜在安全事件停止 | 销毁含值工件；不部署；由负责人外部轮换；raw deletion diff/history 继续按受限材料处理 |
| constructor 生产语义需要修改 | 停止范围扩张 | 不改 `weworkwsbot.go`；另建 Issue/Plan |
| Origin/CSRF/身份/RBAC 或 shell 安全回归 | 停止 | 不合并、不部署，保留失败 worktree |
| 其他全仓测试失败 | 停止，不顺手修 | 新 Issue/Evidence/Plan revision |
| 范围外路径或 lockfile 变化 | 停止范围审查 | 反向撤销本任务精确 patch；不 reset 用户文件 |

## 退出门禁

- [ ] 已复核的 `FE-EVID-W01-009` red 与修复后的 `FE-EVID-W01-010` green
  均为脱敏 Evidence；R3 未再次执行旧 test。
- [ ] 当前 source、新增内容、日志、Evidence 和 commit message 不含旧
  material；raw diff/history 未输出或归档。
- [ ] r004 transport、session、shell 与安全合同在 R3 全部重验通过。
- [ ] channels、全 Gateway、Gateway race、全 Go、UI build 全部通过。
- [ ] 双 base、untracked、gofmt、diff check 和 lockfile Gate 通过。
- [ ] Evidence 通过独立代码与安全复核。
- [ ] `FE-W01-S08` 已完成：credential owner 记录撤销/轮换或权威证明从未
  有效，`FE-EVID-W01-011` 通过。
- [ ] Browser 页面级回归通过。

代码与确定性 Gate 通过但 S08 owner 或 Browser 证据缺失时，W01
分别保持 `active/security-blocked` 或 `active/blocked-external`，不能
complete，也不能声称全部前台功能已恢复。
