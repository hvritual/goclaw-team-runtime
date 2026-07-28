---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Inert-provider local Playwright browser regression gate
revision: 8
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review + wave_docs_validate
supersedes:
  - plan-r007
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

# FE-W01 r008 — Inert-provider 本地 Playwright 回归

## 变更原因与目标

r007/R5 已完成确定性重验和 Playwright 下载，但在创建任何 Token/config 或
启动服务前，`FE-EVID-W01-016` 证明真实 `goclaw gateway run` 必须构造一个
LLM provider；r007 同时要求 synthetic config 关闭全部 provider，合同冲突。
r007 按失败关闭规则停止。

r008 保持原页面目标不变：

> 打开 `/dashboard/`，以全合成 Gateway/Team 凭据登录，确认 WebSocket
> connected，刷新后恢复会话，再退出并回到登录页。

唯一实质变化是新增稳定 Step `FE-W01-S11`：使用一个无真实密钥、只含固定
公开 synthetic marker、不可达且严格受 socket Gate 约束的 inert loopback
provider 满足启动构造器。它不是可用 provider，Browser 流程不得调用它。

## 冻结 Task

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `6` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S07`、`FE-W01-S08`、`FE-W01-S09`、`FE-W01-S04`、`FE-W01-S11`、`FE-W01-S10`、`FE-W01-S05` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005`、`FE-ISSUE-006`、`FE-ISSUE-007`、`FE-ISSUE-008` |
| Assignee | Codex root agent |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Task Base | 待 r008 获批后的 docs-only activation commit；R6 在 Journal 冻结 |
| Plan | `FE-W01 plan-r008` |
| Policy bundle | `wave-governance-v1` |
| Auto product commit | 禁止；S08 和所有退出门禁通过后另行决定 |

r008 不增加产品路径、不修改产品合同、不修改 provider 实现。Browser 发现
任何新页面异常仍必须固化 Evidence 并另起 plan-r009；不得顺手修复。

## R5→R6 迁移

1. Plan、Registry、Wave README、Track、Issue Register、Decision、Evidence
   和 Journal 原子切换，形成 docs-only activation commit；
2. 从 activation commit 创建 `repair/fe-w01-transport-r6`，创建时
   `HEAD == Task Base`，再以 docs-only commit 冻结完整 Task tuple；
3. 在任何产品迁移/Go 运行前，静默比较 Task Base legacy channel SHA-256
   与 `2514948eb0a9fdee39c084ec0cde09eab2b144e2cf9a95511b562c8e4c01f01b`；
4. 匹配后以 Delete/Add 换成 synthetic channel test，先跑无输出 source
   gates；禁止输出 raw deletion diff/history；
5. 迁移其余 10 个 R5 产品文件，最后核对 11 个文件与 r007 的 R4 manifest
   精确一致；
6. R6 使用独立 `npm ci`，复跑 r007 的全部确定性、race、全仓、vet、UI、
   双 base scope 和 lockfile Gate，写入 `FE-EVID-W01-017`；
7. R1–R5 与旧 0.6.0 worktree 保留只读，不 reset、不删除。

11 个产品 SHA-256 以批准的
[`plan-r007`](plan-r007.md#r4-已验收产品内容清单) 为冻结清单；r008 不重复
复制，避免同一 manifest 漂移。

## 稳定 Step

| Step ID | 动作 | Evidence | 状态 |
|---|---|---|---|
| `FE-W01-S01`–`S04`、`S06`、`S07`、`S09` | R6 独立迁移与重验 | `FE-EVID-W01-014`、`017` | ready-after-task-base |
| `FE-W01-S08` | credential owner 撤销/轮换或从未有效证明 | `FE-EVID-W01-011` | blocked-external-owner |
| `FE-W01-S11` | 冻结 inert provider fixture 与零出站 Gate | `FE-EVID-W01-016`、`018` | ready-after-r6-gates |
| `FE-W01-S10` | 建立仓库外 synthetic Gateway/Vite/Playwright runtime | `FE-EVID-W01-015` | blocked-by-S11 |
| `FE-W01-S05` | Desktop/Mobile 登录、connected、刷新、退出 | `FE-EVID-W01-003`、`015` | blocked-by-S10 |

## S11 inert provider 合同

synthetic config 必须只包含一个 provider：

```text
provider id: e2e-disabled
api: openai-completions
baseUrl: http://127.0.0.1:9
model id: synthetic-disabled
model name: Synthetic Disabled
apiKey: test-inert-provider-key
auth/headers/runtime: absent
agents.defaults.model.primary: e2e-disabled/synthetic-disabled
```

端口 `9` 在启动前必须确认无监听。配置预检必须解析 JSON 并断言：

- provider 数量精确为 1，所有字段与上表一致；
- `model name` 精确为 `Synthetic Disabled`，满足 config validator 对显式
  model 定义的非空 name 要求；
- `apiKey` 精确为公开 synthetic marker `test-inert-provider-key`，只为满足
  constructor 的 non-empty 校验；marker 仅允许存在于 `0600` synthetic
  config 和 `0600` sentinel，不通过环境变量或 CLI 传入，并不得进入截图、
  报告或日志；
- 不含真实 API key、auth、header、runtime command 或环境变量引用；
- 所有真实/生产 provider、Codex app-server 和宿主 provider 环境均不存在；
- 全部 channel、workstation、development、Harness、Ouroboros、
  tools shell/web/browser、memory catalog auto-ingest 均关闭；
- TeamControl 与 Gateway 双层认证是仅有的必要启用能力。

provider constructor 只创建 client，不应建立连接。Gateway 必须在启动前由
`strace -f -e trace=connect` 方向感知 syscall Gate 包裹，以覆盖 Gateway
及其子进程的短暂或失败 `connect()`，不能用 socket 轮询替代：

- 允许 Gateway 在 `8080/28789` 接受 loopback 入站连接；
- Gateway 及其子进程不允许任何出站 `connect()`，包括
  `127.0.0.1:9`；Vite 与 Browser 的允许连接按独立 PID/进程树核算；
- 一旦 chat/model RPC、provider request 或任意 Gateway 出站连接出现，立即
  终止所有进程，S11/S10 失败；
- Browser 脚本不得进入对话页或调用 chat/model RPC。
- 若 syscall Gate 不可用或不能归属进程树，标记 `environment-blocked`，
  不以已建立 socket 快照替代；Evidence 只记录脱敏计数和目标类别。

该 inert provider 只是测试构造器依赖，不得描述为生产 provider 已启用。

## 下载工件继承边界

r007 下载阶段发生在任何 synthetic Token/config/runtime 之前。r008 可只读
复用以下仓库外工件：

| 工件 | 冻结值 |
|---|---|
| Root | `/tmp/goclaw-fe-w01-r5-playwright-MUTsW4` |
| Playwright | `1.55.0` |
| tooling package-lock SHA-256 | `53622035b305ccadd941f377f72f9231deb8394810387cea36196b2fb6a7e3fe` |
| `playwright/package.json` SHA-256 | `bb26592b48d8a2157291e96a8a23ca39e3def369165283a5e7c883b24faa41b4` |
| `playwright-core/package.json` SHA-256 | `36ca1b094edaa37835521c008b26cab5375cbab895ad4e7a9ab6577db23abec5` |
| node_modules regular-file manifest SHA-256 | `4a87376e407b7093d8dfa42b3051ffbac1b4f5ec86ac05fddc7da08654968988` |
| node_modules symlink manifest SHA-256 | `ee2013d54217dc845b497a758e9c010bed74001f1fc3cffd487e70a946587f2e` |
| Chromium build | `1187` |
| Chromium version | `140.0.7339.16` |
| Chromium executable SHA-256 | `2fa605e3639b8cfbe8037d0b8e0324dbf7f9e6ad7beb345374ecd26764e2d92b` |
| Chromium regular-file manifest SHA-256 | `0f88026f00f407c0c858d3ed95da311baec3320450da01cf12fc97363c7b20e7` |
| Chromium symlink manifest SHA-256 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |

复用前必须确认：

- 版本、package lock、关键 package manifest、node_modules/Chromium
  regular-file 与 symlink manifest SHA 精确匹配，并以 quiet `npm ls`
  验证依赖树；
- 先删除下载阶段 npm cache 和其中 debug log；之后下载 root 顶层只允许
  `tooling` 与 `browsers`，并在排除受冻结 npm/Chromium 程序文件本身后确认
  不存在 runtime token、sentinel、synthetic config、database、user
  profile、cookie、trace、HAR、video、screenshot 或运行日志；
- owner 必须是当前测试主体；禁止越出 root 的 symlink/hardlink、setuid/
  setgid 文件和 group/world writable 文件；
- package/browser 目录切成只读，运行时 user-data-dir 指向新的 R6 runtime
  root；
- npm cache 删除并完成无 credential/runtime scan 后才创建任何 R6
  credential；
- 不重新运行下载进程，不允许下载与测试阶段重叠。

manifest 采用同一 canonical 算法：

1. 分别以 `tooling/node_modules` 和 `browsers/chromium-1187` 为 root；
2. 先拒绝路径或 symlink target 含换行/NUL 的工件；
3. regular-file manifest 等价于在该 root 内执行
   `find . -type f -printf '%P\0' | LC_ALL=C sort -z | xargs -0 sha256sum |
   sha256sum`：记录 root-relative path 与逐文件内容 SHA，不含绝对路径；
4. symlink manifest 等价于
   `find . -type l -printf '%P -> %l\n' | LC_ALL=C sort | sha256sum`：
   记录 root-relative path 和 exact target；解析后的 target 必须仍在 root；
5. mode、owner、hardlink 与 setuid/setgid 使用前述独立 Gate，不混入内容
   manifest；EVID018 记录 `find/sort/xargs/sha256sum/npm/node` 版本与最终
   digest/计数，禁止输出文件名清单。

任一检查失败则销毁该下载工件并重新执行独立下载阶段；不得降低版本/hash
Gate 或改用其他自动化框架。

## S10 隔离、安全与网络合同

r007 以下合同全部原样继承：

- bootstrap、每个 fixture setup CLI、Gateway、Vite、Playwright 分进程
  `env -i`，HOME/TMPDIR/XDG 全指向新的
  `/tmp/goclaw-fe-w01-r6-runtime-*`；
- synthetic Gateway/Team Token 精确 holder、`0600` config/token/log/
  sentinel、Gateway/Vite 不持有 Team/Reviewer Token；
- Browser 只允许 `http://127.0.0.1:5173` 和
  `ws://127.0.0.1:5173`；Vite 只连接 loopback `8080/28789`；
- deny proxy、Chromium background-network 禁用、request/WS/worker hook 和
  方向感知 PID socket audit；
- 登录全程禁 raw trace/HAR/video/network snapshot/自动失败 screenshot；
  credential input screenshot mask，Console/HTTP/WS 只保存派生元数据；
- exact token sentinel quiet scan；命中即销毁工件并判失败；
- 两阶段清理；仅保留通过 scan 的报告与截图到仓库外 `0700` Evidence 目录。

除 S11 明确列出的 inert provider 外，“生产能力关闭”语义不变。完整字段、
截图时机、Desktop `1440×1000`、Mobile `390×844` 和登录/刷新/退出矩阵以
批准的 [`plan-r007`](plan-r007.md#s05-浏览器验收矩阵) 为准。

## Evidence

| Evidence ID | 内容 |
|---|---|
| `FE-EVID-W01-014` | R5 已通过的确定性重验 |
| `FE-EVID-W01-016` | provider constructor 预检与 r007 安全停止 |
| `FE-EVID-W01-017` | R6 独立确定性重验 |
| `FE-EVID-W01-018` | inert config 预检、下载工件完整性与 Gateway 零出站绿色证明 |
| `FE-EVID-W01-015` | 最终 local Playwright Desktop/Mobile browser bundle |
| `FE-EVID-W01-011` | 外部 credential owner 证明 |

## 风险、停止与回滚

| 信号 | 动作 | 回滚 |
|---|---|---|
| inert provider 字段偏离、synthetic key marker 不匹配、端口 9 有监听 | 启动前停止 | 不创建 Token/runtime；修订计划 |
| Gateway 出站连接或 provider/model RPC | 立即终止并判失败 | 清理 runtime；不保留含值工件 |
| 下载工件含 runtime/credential 或 hash 漂移 | 禁止复用 | 销毁并独立重下 |
| R6 产品 manifest、scope、lockfile 漂移 | 停止 S11 | 不启动服务；重新复核迁移 |
| 页面/Console/HTTP/WS Gate 失败 | 固化脱敏失败 Evidence | 新 Issue（若需要）与 plan-r009 |
| sentinel 命中、非允许 origin/peer、残留 PID/端口 | 失败关闭 | 销毁敏感工件；不索引 raw artifact |

## 入口门禁

- [x] `FE-EVID-W01-014` 已生成；R5 deterministic Gate 通过。
- [x] `FE-EVID-W01-016` 已证明 provider constructor 合同冲突。
- [x] r007 下载阶段未创建 credential/runtime。
- [x] Wave、Security 与文档 Reviewer 批准 r008。
- [x] Registry 与 8 个原子文档切换到 r008。
- [ ] R6 Task Base/worktree/Plan SHA/完整 tuple 冻结。
- [ ] R6 `FE-EVID-W01-017` Gate 通过。

## 退出门禁

- [ ] R6 source-first、manifest、全量确定性、scope/lockfile 通过。
- [ ] inert provider config 预检与 Gateway 零出站 Gate 通过。
- [ ] Desktop/Mobile 登录、connected、刷新恢复、退出全通过。
- [ ] 页面身份、非空、无 overlay、Console、HTTP/WS、截图与 sentinel 通过。
- [ ] 两阶段清理、PID/端口关闭和受控 Evidence 保留通过。
- [ ] Evidence 014/015/016/017/018 通过独立代码、安全与文档复核。
- [ ] `FE-W01-S08 / FE-EVID-W01-011` 通过。

前六项通过但 S08 未完成时，S05 可 complete，W01 仍保持 active；产品补丁
不得提交或发布。
