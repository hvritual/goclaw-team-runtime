---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Transport, dashboard, deterministic channels, and catalog provenance gate
revision: 6
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review
supersedes:
  - plan-r005
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
  - memory/catalog/ingest.go
  - memory/catalog/service_test.go
  - docs/waves/**
product_code_changes_allowed: true
---

# FE-W01 r006 — Transport、Dashboard、确定性 Channel 与 Catalog Gate

## 目标与冻结任务候选

完整保留 r005 的 transport、dashboard shell、Origin/CSRF/身份/RBAC、
synthetic channel test 和 credential 边界；修复全仓 Gate 新发现的默认
Markdown provenance kind 推断错误。既有 Catalog 数据迁移不在本 revision。

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `4` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S07`、`FE-W01-S08`、`FE-W01-S09`、`FE-W01-S04` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005`、`FE-ISSUE-006`、`FE-ISSUE-007`、`FE-ISSUE-008` |
| Assignee | Codex root agent |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Task Base | 待 r006 获批后的 docs-only activation commit；R4 创建时在 Journal 冻结精确 SHA |
| Plan | `FE-W01 plan-r006` |
| Policy bundle | `wave-governance-v1` |
| Auto product commit | 禁止；确定性验证与独立验收后再决定 |

本 revision 不覆盖 Catalog schema/数据库迁移、既有 record 重写、知识审批、
检索排序、项目隔离、生产 channel、业务 loader、页面命令、认证 handler
或 Browser 策略绕行。

## r005 迁移规则

r005/R3 worktree 中的 transport、shell 和 synthetic channel patch 保持未提交。
r006 获批后：

1. Plan、Registry、Track、Issue、Decision、Evidence 与 Journal 的原子切换
   先形成 docs-only activation commit；
2. 从该 commit 创建 `repair/fe-w01-transport-r4`，创建时
   `HEAD == Task Base`；
3. 先以 docs-only commit 冻结 Task Base、branch、worktree、Plan SHA 和完整
   Task tuple；
4. 只迁移 r006 allowlist 内 R3 patch，不复用 R3 `node_modules` 或进程；
5. 因 Task Base 仍含旧 channel test，R4 在运行任何 Go test 前以无输出命令
   将其 SHA-256 与已复核 expected SHA
   `2514948eb0a9fdee39c084ec0cde09eab2b144e2cf9a95511b562c8e4c01f01b`
   精确比较；不相等立即停止。比较通过后以 Delete/Add 方式立即换成 R3
   已验证的 synthetic test，避免回显删除行，再先跑无输出 source gates；
   不得重跑旧 test，不得显示 raw deletion diff/history；
6. 在修改 `memory/catalog/ingest.go` 前，先迁移/补齐 S09 推断矩阵测试并观察
   `FE-EVID-W01-012` 对应失败；
7. R4 重新执行全部 green Gate；只继承经复核的 red Evidence，不继承 R3
   green 结论。

R1、R2、R3 与旧 0.6.0 worktree 均保留只读，不 reset、不删除。

## 入口门禁

- [x] `FE-ISSUE-002`–`008` 均有确定性 reproduction/root-cause Evidence。
- [x] Memory 两个候选路径与累计 base、R3 Task Base 和当前 HEAD 哈希一致。
- [x] S07 source gate、channels/race、Gateway/race 与 UI build 已在 R3 通过。
- [x] S09 red Evidence、成功条件、数据边界和回滚已定义。
- [x] Wave Reviewer 批准 r005→r006、稳定 Step 和 Task revision 迁移。
- [x] Security Reviewer 批准 provenance、credential history 与无网络迁移顺序。
- [x] Registry、README、Track、Issue、Decision、Evidence 与 Journal 原子切换。
- [ ] R4 worktree 与 Task Base 精确 SHA 已冻结。

入口未全部通过前，不得修改 Memory 路径，也不得把 R3 product patch 迁移
到 R4。

## 稳定 Step

| Step ID | 动作 | 先失败证据 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01` | 保留 session test 适配 | R1 编译 red | R4 目标与全 Gateway 通过 | complete-in-r3; reverify-in-r4 |
| `FE-W01-S02` | 保留 Vite 同源 `/auth`、`/ws` proxy | auth 403 | R4 auth 204、WS 101 | complete-in-r3; reverify-in-r4 |
| `FE-W01-S03` | 保留 TeamClient 同源 `/ws` | direct `:28789` | R4 URL/protocol/token gate 通过 | complete-in-r3; reverify-in-r4 |
| `FE-W01-S06` | 保留 dashboard shell/security/cache 修复 | slash 301 | R4 shell/asset/unknown-route 合同通过 | complete-in-r3; reverify-in-r4 |
| `FE-W01-S07` | 保留 synthetic、离线、无等待 constructor test | `FE-EVID-W01-009` | R4 source gate 与 channels/race 通过 | complete-in-r3; reverify-in-r4 |
| `FE-W01-S08` | owner 提供撤销/轮换或从未有效证明 | owner 未解析 | `FE-EVID-W01-011` passed | blocked-external-owner |
| `FE-W01-S09` | 规范化默认 Markdown provenance kind 推断 | `FE-EVID-W01-012` | 推断矩阵、包级/race 与全仓通过 | ready-after-task-base |
| `FE-W01-S04` | 全 Gateway/race、全 Go、UI build、scope/lockfile Gate | r005 被 Catalog test 阻断 | 所有确定性 Gate 通过 | blocked-by-S09 |
| `FE-W01-S05` | 真实 Browser 登录、connected、刷新、退出 | localhost policy blocked | 页面级验证通过 | blocked-external |

S09 是新 Step，不改写 S04/S05；r005 的全部 transport、shell、channel 和安全
合同继续具有约束力。

## S09 测试合同

`memory/catalog/service_test.go` 必须先锁定：

1. `SourceScheme=""` 且 `SourceKind=""` 时，最终 scheme/kind 为
   `markdown` / `markdown`；
2. 显式 `SourceScheme="markdown"` 且 kind 为空时仍推断为 `markdown`；
3. 显式 kind（例如 `managed-markdown`）原样保留；
4. `git+markdown` 且 kind 为空时仍推断为 `git-markdown`；测试必须提供
   synthetic `SourceRevision`，不得依赖真实 Git remote；
5. synthetic 非 Markdown/Git scheme（例如 `obsidian`）且 kind 为空时推断为
   `obsidian-markdown`；
6. 每个推断 case 同时断言 ProjectID 保持调用方项目、Source URI 使用该项目
   namespace，默认 collection 仍为 `knowledge-markdown`；
7. 不同目录下同名 Markdown 的 Source URI 仍不同；
8. 测试只使用 `t.TempDir` 和 synthetic Markdown，不访问网络或生产数据。

现有 `TestIngestStableRootDistinguishesSameNamedItems` 是先失败合同；新增矩阵
不得删除或放宽它。

## S09 实现合同

仅当 `SourceKind` 为空时推断：

| Source scheme | 推断 kind |
|---|---|
| `markdown`（包括空 scheme 的默认值） | `markdown` |
| `git+markdown` | `git-markdown` |
| 其他合法 scheme | `<scheme>-markdown` |

显式 `SourceKind` 必须保持不变。不得改变 Source URI、SourceRevision、
collection、稳定根目录、symlink/size 防护、生命周期、审批或项目边界。

本修复只影响后续 ingestion，不追溯重写已存在的 `markdown-markdown` record。
若实际数据审计发现需要迁移，必须建立独立 Issue/Plan，不得在此顺带处理。

## Credential 与 Browser 边界继承

- R4 不再次运行旧 channel test；只静默比较到本 Plan 冻结的 expected SHA，
  不匹配即停止，匹配后以 Delete/Add 立即安全替换；
- 当前 tree、新增内容、日志、Evidence 和 commit message 不得含旧 material；
- raw deletion diff/history 不输出、不归档；
- credential owner 未完成 `FE-W01-S08` 前，W01/发布不能 complete；
- Browser 不可达时不得使用另一浏览器通道规避，也不得宣称页面全量恢复。

## Evidence 映射

| Step | Evidence |
|---|---|
| `FE-W01-S01`–`S03` | `FE-EVID-W01-001`、`002`、`004`、`006` |
| `FE-W01-S06` | `FE-EVID-W01-007`、`008` |
| `FE-W01-S07` | `FE-EVID-W01-009`、`010` |
| `FE-W01-S08` | `FE-EVID-W01-011` |
| `FE-W01-S09` | `FE-EVID-W01-012`、`013` |
| `FE-W01-S04` | `FE-EVID-W01-005` |
| `FE-W01-S05` | `FE-EVID-W01-003` |

`FE-EVID-W01-013` 是 R4 的 Catalog 推断矩阵 green Evidence。

## 验证命令

```text
perl -ne 'if (/(BotID|SecretID):\s*"([^"]*)"/ && $2 !~ /^test-/) { exit 1 }' \
  channels/weworkwsbot_test.go
! rg -q '\.(Start|doConnect|Send|SendStream)\(|time\.(Sleep|NewTicker|After)\(|go[[:space:]]+func|gorilla/websocket|net/http|wss?://' \
  channels/weworkwsbot_test.go
go test ./channels -count=1 -v -timeout=30s
go test -race ./channels -count=1 -timeout=30s
go test ./memory/catalog \
  -run 'TestIngest(StableRootDistinguishesSameNamedItems|SourceKindInference)' \
  -count=2 -timeout=30s
go test ./memory/catalog -count=1 -timeout=30s
go test -race ./memory/catalog -count=1 -timeout=30s
npm --prefix ui run test:transport
npm --prefix ui run build
go test ./gateway -run 'TestTeamConsole' -count=1 -v
go test ./gateway -count=1
go test -race ./gateway -count=1
go test ./... -count=1 -timeout=5m
test -z "$(gofmt -l channels/weworkwsbot_test.go \
  memory/catalog/ingest.go memory/catalog/service_test.go \
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

Task Base、累计 base 与 untracked 三组路径合并去重后必须精确匹配 12 项
allowlist。`package-lock.json` SHA 必须保持
`46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb`。

## 风险、停止与回滚

| 信号 | 动作 | 回滚 |
|---|---|---|
| channel test 访问网络/等待或旧 material 出现 | 安全停止 | 不合并；销毁含值工件；owner 外部处理 |
| 显式 kind、git kind、URI/collection 语义回归 | 停止 S04 | 反向撤销 S09 精确 patch；不 reset 用户文件 |
| 修复需要 schema/数据迁移 | 停止范围扩张 | 另建 Issue/Plan，不写既有 record |
| Origin/CSRF/身份/RBAC/shell 回归 | 安全停止 | 不合并、不部署 |
| 其他全仓测试失败 | 停止，不顺手修 | 新 Issue/Evidence/Plan revision |
| 范围外路径或 lockfile 改变 | 停止范围审查 | 反向撤销任务精确 patch，不 reset |

## 退出门禁

- [ ] R4 未运行旧 channel test，source gate、channels 和 race 通过。
- [ ] Catalog 推断矩阵先失败后通过，包级、race 与全仓 Gate 通过。
- [ ] r005 transport、session、shell 与安全合同在 R4 全部重验通过。
- [ ] UI transport/build、双 base、untracked、gofmt、diff 和 lockfile Gate 通过。
- [ ] `FE-EVID-W01-010`、`012`、`013` 通过独立代码与安全复核。
- [ ] `FE-W01-S08` owner Evidence 通过。
- [ ] Browser 页面级回归通过。

前五项通过但 owner 或 Browser 仍不可达时，W01 必须保持 `active` 并记录
对应 blocker reason，不能标记 complete、不能产品提交或发布。
