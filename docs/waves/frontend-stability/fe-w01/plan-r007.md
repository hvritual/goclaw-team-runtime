---
schema: goclaw.wave/v1
wave_id: FE-W01
track_id: FE-STABILITY-2026-07
title: Local Playwright browser regression gate
revision: 7
plan_status: approved
wave_state: active
approved_by: user-directive + wave_transition_review + transport_security_review + wave_docs_validate
supersedes:
  - plan-r006
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

# FE-W01 r007 — 本地 Playwright 真实浏览器回归

## 授权、目标与冻结任务候选

用户于 2026-07-26 明确授权建立 r007，并在 Cloud Browser 因安全策略拒绝
localhost 后改用本地 Playwright。该授权只解除 `FE-W01-S05` 的浏览器执行
方式阻塞，不解除 `FE-W01-S08` credential owner 阻塞，也不授权发布。

本 revision 的目标流程只有一句：

> 打开 `/dashboard/`，以全合成 Gateway/Team 凭据登录，确认 WebSocket
> connected，刷新后恢复会话，再退出并回到登录页。

| 字段 | 冻结值 |
|---|---|
| Project-ID | `goclaw-team-runtime` |
| Task-ID | `FE-W01-TRANSPORT-R1` |
| Task-Revision | `5` |
| Work-Item | `FE-W01-S01`、`FE-W01-S02`、`FE-W01-S03`、`FE-W01-S06`、`FE-W01-S07`、`FE-W01-S08`、`FE-W01-S09`、`FE-W01-S04`、`FE-W01-S10`、`FE-W01-S05` |
| Issue | `FE-ISSUE-002`、`FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005`、`FE-ISSUE-006`、`FE-ISSUE-007`、`FE-ISSUE-008` |
| Assignee | Codex root agent |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Prior verified R4 HEAD | `9d01549cc96be05810dc614fe757e3a772fc862a` |
| Task Base | 待 r007 获批后的 docs-only activation commit；R5 创建时在 Journal 冻结精确 SHA |
| Plan | `FE-W01 plan-r007` |
| Policy bundle | `wave-governance-v1` |
| Browser runner | 本地 Playwright `1.55.0` + 其固定 Chromium revision；安装与缓存仅在任务临时目录 |
| Auto product commit | 禁止；S08 与所有退出门禁通过后另行决定 |

r007 不增加业务功能，不修改新产品合同，不把临时 e2e 脚本、浏览器二进制、
截图、Trace、Token、Cookie、CSRF、运行数据库或日志写入仓库。若真实页面
出现新异常，只允许登记新的稳定 Issue、脱敏 Evidence 和下一 Plan revision；
本 revision 立即停止，不顺手修复。

## r006/R4 迁移规则

R4 的 11 个产品路径保持未提交。r007 获批后：

1. Plan、Registry、Wave README、Track、Issue Register、Decision、
   Evidence 与 Journal 原子切换，形成 docs-only activation commit；
2. 从该 commit 创建 `repair/fe-w01-transport-r5`；创建时
   `HEAD == Task Base`，不得复用 R4 进程、`node_modules` 或运行状态；
3. 在迁移任何产品 patch 前，以 docs-only freeze commit 记录 Task Base、
   branch、worktree、Plan SHA 和完整 Task tuple；
4. 因 Task Base 仍含旧 channel test，R5 在迁移任何产品 patch、运行任何
   Go 程序或 test 前，以
   无输出命令把其 SHA-256 与 expected SHA
   `2514948eb0a9fdee39c084ec0cde09eab2b144e2cf9a95511b562c8e4c01f01b`
   精确比较；不相等立即停止。比较通过后以 Delete/Add 立即替换为已验收
   synthetic test，再先跑无输出 source gates；
5. source gates 通过后，只迁移其余 10 个 r007 allowlist 内、已由 R4
   独立验收的产品文件；不得再次迁移 channel，不得加入新产品改动；
6. 核对全部 11 个产品文件与下方 R4 SHA-256 manifest 精确一致；
7. 禁止输出或归档旧 channel test 的 raw deletion diff/history；
8. R5 独立安装 UI 依赖并重跑前置确定性 Gate；green 结论不能直接继承；
9. R1–R4 和旧 0.6.0 worktree 全部保留只读，不 reset、不删除。

迁移后如果任何 allowlist 文件与 R4 已验收内容不一致，停止并重新进行代码/
安全复核，不得开始浏览器。

### R4 已验收产品内容清单

R5 迁移后、运行前必须逐项精确匹配：

| 路径 | R4 SHA-256 |
|---|---|
| `ui/src/team/client.ts` | `ce72f1b15721a5be6ed9fe5625182d94bd3274c6181ce82d615c7c62d81813f6` |
| `ui/vite.config.ts` | `f335cbbc7c10201930c4e360ec95df99b7817891907fe98f21a7ad5bcf3868f3` |
| `ui/package.json` | `bc254d466186012ca175eab62e42a5be6ec81e3977a9997464e31e0fd93c2ee0` |
| `ui/tests/team-transport.test.mjs` | `0724cba8b1474263e9698f842d9529d15b9f6a9abb6e3d63d8772e65d89530ae` |
| `gateway/team_runtime_test.go` | `548ca84c27b667d100dffa0f54b5e895847a2ca747af4e39707922fe3dd40b09` |
| `gateway/web_sessions_test.go` | `920bbb7d55d6930f9de068a03939f17f961127b578c485d29fb8b3c5e7219405` |
| `gateway/ui.go` | `42287ecacc994c3d592ac870d9ebab9a4526c66caa23d36caa4383361eb4bbfa` |
| `gateway/server_auth_test.go` | `78eefc861967ad21603ecc122df1eb92ec53adaa2810a20c567ace3fe7d07215` |
| `channels/weworkwsbot_test.go` | `69b89a9fe880a68ae1a4ae50db0b0b3266a40cc405b38ceb57934c8901f6a92c` |
| `memory/catalog/ingest.go` | `391acede0cde724274281beaa854e8c07d170111503077a9fb2197ae30663032` |
| `memory/catalog/service_test.go` | `9a5a068fe7a29afebf0f9937c95b8194b5768fd89dfb42191e9bd0c43b482e64` |

该清单只证明迁移内容一致；R5 仍必须独立复跑 Gate。

## 入口门禁

- [x] 用户明确授权本地 Playwright fallback。
- [x] fallback 原因冻结：先前 Browser invocation 因 Browser 安全策略拒绝
  localhost；本轮不再次调用 Cloud Browser。
- [x] R4 确定性 Gate 与独立代码/安全复核通过。
- [x] 浏览器目标流程、视口、凭据、状态和留证边界已定义。
- [x] 本机未发现现成 Playwright package 或浏览器二进制；临时安装策略已定义。
- [x] Wave Reviewer、Security Reviewer 与文档一致性 Reviewer 批准 r007。
- [x] Registry、Wave README、Track、Issue Register、Decision、Evidence 与
  Journal 原子切换。
- [ ] R5 worktree、Task Base、Plan SHA、分支与完整 Task tuple 精确冻结。
- [ ] R5 source-first channel hygiene 和全部浏览器前置 Gate 通过。

入口未全部通过前，不得启动本地浏览器、Gateway、Vite，不得安装 Playwright，
也不得迁移 R4 产品 patch。

## 稳定 Step

| Step ID | 动作 | 先前证据/前置 | 通过条件 | 状态 |
|---|---|---|---|---|
| `FE-W01-S01`–`S04`、`S06`、`S07`、`S09` | 迁移并独立复验 R4 已验收 patch | `FE-EVID-W01-013` | R5 全部确定性 Gate 通过 | ready-after-task-base |
| `FE-W01-S08` | owner 提供撤销/轮换或从未有效证明 | owner 未解析 | `FE-EVID-W01-011` passed | blocked-external-owner |
| `FE-W01-S10` | 建立仓库外本地 Playwright runtime 与一次性合成 Gateway/Team fixture | Cloud Browser localhost policy blocked；本机无现成 Playwright/browser | 版本、进程、端口、fixture、清理和脱敏 Gate 通过 | ready-after-r5-gates |
| `FE-W01-S05` | 桌面与移动执行登录、connected、刷新恢复、退出 | `FE-W01-S10` | 页面身份、非空、无 overlay、Console/HTTP/WS、交互和截图全部通过 | blocked-by-S10 |

S10 是浏览器执行基础设施 Step，不改写既有 S05 身份。它不在仓库新增
Playwright 依赖，不改变 `package.json` 或 lockfile。

## S10 本地运行合同

### 隔离环境

- 临时根目录使用 `/tmp/goclaw-fe-w01-r5-playwright-*`；
- `HOME`、XDG、TeamControl root、workspace、sessions、日志、Token 文件、
  npm cache、Playwright cache、脚本、截图和 Trace 全在该临时根目录；
- 只使用 synthetic user/team/project/token，不读取生产 Vault、TeamControl、
  浏览器 profile、Cookie 或用户 Codex/OAuth；
- Gateway HTTP、WebSocket 和 Vite 只绑定 `127.0.0.1`；因受测
  `vite.config.ts` 已冻结 proxy target，执行前必须确认 `8080`、`28789`
  和 `5173` 均空闲，分别用于 HTTP、WebSocket 和 Vite，否则安全停止；
- 进程 PID、端口与退出码可追踪；结束后先停止子进程，再清理临时状态；
- Token 只通过 `0600` 文件或子进程环境传递，不进入命令行、日志、截图、
  Console dump、Wave 文档或 git diff。

### 最小进程环境

- bootstrap、每个 team/project fixture setup CLI、Gateway、Vite 与
  Playwright 必须分别由 `env -i` 启动，只显式传入该进程需要的 `PATH`、
  `HOME`、`TMPDIR`、`XDG_CONFIG_HOME`、
  `XDG_CACHE_HOME`、`XDG_DATA_HOME`、`XDG_STATE_HOME`、locale 和 loopback
  地址；所有路径都指向本次临时根；
- 不继承 `GOCLAW_*`、OpenAI/Codex、SSH agent、Git credential、Docker/
  Kubernetes、云、provider、Reviewer、HTTP(S)/SOCKS proxy 或用户 shell
  环境；执行前只对变量名做静默 denylist 检查，命中即停止且不回显值；
- synthetic Gateway Token 只允许存在于 `0600` config、`0600` sentinel，
  以及 Playwright runner 为填写 password input 所需的短期内存；不得进入
  Gateway/Vite 环境、日志、CLI 参数或保留工件；
- synthetic Team Token 只允许存在于 bootstrap `0600` token file、`0600`
  sentinel，以及 fixture setup CLI/Playwright 的短期内存；setup CLI 的
  `GOCLAW_GATEWAY_TOKEN`/`GOCLAW_USER_TOKEN` 必须从文件显式注入该次
  `env -i` 子进程，其他进程不得继承；Gateway、Vite 环境不得持有 Team/
  Reviewer Token；
- synthetic config 显式关闭所有 channel、provider、workstation、
  development execution、tools browser/web/shell、Harness/Ouroboros 和
  Codex 调用能力；启动前以结构化 config 检查失败关闭，启动后通过
  health/channel 状态与进程树再次确认没有启用生产能力；
- config、Token、日志、profile、数据库和 cache 目录均为 `0700`，其中
  config、Token 与日志文件为 `0600`。

### Playwright 获取

- 固定 `playwright@1.55.0` 和该版本管理的 Chromium revision；
- package、browser 与 npm cache 只安装到临时根目录；
- 不运行 `npm install` 修改 `ui/node_modules`，不修改 repo manifest/lockfile；
- 安装后的 package 版本、browser executable hash 与版本字符串写入脱敏
  Evidence；若固定版本/Chromium 无法取得，安全停止并记录环境阻塞；
- 禁止为通过测试改用第二套浏览器自动化库。

Playwright/Chromium 下载是独立准备阶段；只有该阶段允许 npm/浏览器下载。
下载完成后关闭下载进程，测试运行阶段使用上述最小环境且不得访问外网。

### 真实服务拓扑

1. 从 R5 源码构建一次性 `goclaw` 二进制到临时根目录；
2. 写入最小 synthetic config，启用 TeamControl 和 Gateway 双层认证；
3. 使用 CLI bootstrap 生成一次性管理员 Team Token，并建立 synthetic team、
   project 和 membership；命令输出必须重定向到受限临时日志并脱敏摘要；
4. 启动 Gateway HTTP/WS，再启动 R5 `ui` 的 Vite dev server；
5. 浏览器只能访问 Vite 页面 origin；`/auth`、`/rpc`、`/ws` 走真实 proxy；
6. readiness 只检查健康端点和页面 shell，不预先调用受测登录流程。

任何生产 provider/channel/workstation/development execution 必须关闭，测试
不得访问外网或启动 Codex。

### 网络失败关闭

- Playwright 对 request、WebSocket 和 worker URL 建立精确 allowlist：
  Browser 只允许 `http://127.0.0.1:5173` 和
  `ws://127.0.0.1:5173`；任何其他 origin 立即中止并使 Gate 失败，只记录
  脱敏 origin；
- Chromium 以禁用 background networking、component update、sync、
  safe-browsing update 和首次运行服务的参数启动；
- Chromium 不继承任何 proxy 变量；测试另启 loopback deny proxy，
  Chromium 只对 `127.0.0.1,localhost` bypass，其他请求必须进入 deny
  proxy 并令 Gate 失败；
- Gateway/Vite/Browser 运行期间对其 PID 做 socket 审计；Browser 只允许
  loopback `5173`，Vite 只允许 loopback `8080/28789`，出现其他 TCP/UDP
  peer 即失败；
- 安装阶段的网络连接不得与测试运行阶段重叠，安装结束后才允许创建
  synthetic Token、config、数据库和 browser context。

### Trace、截图与 sentinel

- 登录凭据输入和 `/auth/session` 全程禁用 Playwright raw trace、video、
  HAR、network snapshot 和自动失败 screenshot；
- 登录前截图只在两个 credential input 均为空时采集；登录后截图只在表单
  已消失后采集；失败时若登录表单仍存在，必须通过 Playwright locator
  mask 两个 credential input，且不得保留 raw 页面 dump；
- Browser Console/HTTP/WS 只保存派生元数据，不保存 header、Cookie、
  request/response body 或表单值；
- synthetic Gateway/Team Token 以原值逐行写入 `0600` sentinel 文件；
  所有拟保留文本、截图、归档和报告都必须执行无输出 exact fixed-string
  sentinel scan。任何命中都立即销毁该工件并使 Gate 失败；
- 不得编辑 raw trace 伪装脱敏；本 revision 不保留 raw trace。

## S05 浏览器验收矩阵

每个视口都使用全新 browser context：

| 场景 | Desktop `1440×1000` | Mobile `390×844` | 必须断言 |
|---|---:|---:|---|
| 未登录 | 是 | 是 | URL 与 Team Console 标题正确；页面 DOM 非空；登录表单可见；无框架错误覆盖层 |
| 登录 | 是 | 是 | 使用 synthetic token 成功；长期 Token 不出现在 URL、storage 或 DOM |
| connected | 是 | 是 | 顶部连接状态在超时内变成 connected/`Production`；WS upgrade 成功 |
| 刷新恢复 | 是 | 是 | reload 后仍进入 Console；HttpOnly session 恢复；Reviewer Token 不持久化 |
| 退出 | 是 | 是 | 点击“退出登录”后回到登录页；刷新仍为未登录；session cookie 被清除 |

每个 context 额外收集：

- 最终 URL、document title、主要 heading 和非空 DOM；
- `pageerror`、Console error/warning；
- `/auth/session`、关键 `/rpc` 状态和 `/ws` upgrade/close，不保存敏感 header/body；
- 登录前、connected、刷新后、退出后截图；
- 桌面与移动至少各完成一次真实点击/表单交互；
- 检查 React/Vite/framework error overlay 不存在。

Console warning 只有在证明与产品无关、可重复且记录理由后才可列为非阻断；
未解释的 error/warning、401/403/5xx、WS 失败、空白页或 overflow 均判失败。

## Evidence 映射

| Step | Evidence |
|---|---|
| `FE-W01-S01`–`S04`、`S06`、`S07`、`S09` | `FE-EVID-W01-013`、`FE-EVID-W01-014` |
| `FE-W01-S08` | `FE-EVID-W01-011` |
| `FE-W01-S10` | `FE-EVID-W01-015` |
| `FE-W01-S05` | `FE-EVID-W01-003`、`FE-EVID-W01-015` |

`FE-EVID-W01-014` 保存 R5 source-first、确定性、scope/lockfile 重验；
`FE-EVID-W01-015` 保存脱敏 QA 报告和仓库外截图的绝对位置与 SHA。不得复制
Token、Cookie、CSRF、完整网络 body 或 raw Trace。

### 两阶段清理与 Evidence 保留

1. 关闭 browser context、Chromium、Gateway 和 Vite，确认记录的 PID 不存在，
   且 `5173`、`8080`、`28789` 均无监听；
2. 删除临时根内 Token、sentinel、config、数据库、profile、Cookie、日志、
   package/browser cache 和 raw runtime 状态；
3. 仅把通过 sentinel scan 的脱敏 QA 报告与截图移动到仓库外
   `/workspace/scratch/afe5d81cd055/evidence/fe-w01-r5-playwright/`，
   目录 `0700`、文件 `0600`；记录 SHA 后保留至独立复核完成；
4. raw Trace、HAR、video、network body 和未通过 sentinel scan 的工件一律
   不保留、不索引。

## 验证命令

在 R5 中先执行 r006 全部验证命令，并将 `<TASK_BASE_SHA>` 替换为冻结值。
此外：

```text
test "$(sha256sum ui/package-lock.json | cut -d' ' -f1)" = \
  46fd937f66b1b7a16950df8347619831948e9dded477b7d4ba8139018974bdbb
test -z "$(git status --short -- ui/package-lock.json)"
test ! -e ui/playwright.config.ts
test ! -e ui/playwright.config.js
test ! -d ui/test-results
test ! -d ui/playwright-report
```

R5 重验写入 `FE-EVID-W01-014`。Playwright 执行命令、固定 package/browser
版本、服务端口、视口、断言计数、Console/HTTP/WS 摘要、截图 SHA 和清理
结果在运行后写入 `FE-EVID-W01-015`，不预先伪造命令。

Task Base、累计 base 与 untracked 三组路径合并去重后必须仍精确匹配本
revision 的 12 项 allowlist。`package-lock.json` SHA 必须保持冻结值。

## 风险、停止与回滚

| 信号 | 动作 | 回滚 |
|---|---|---|
| 旧 credential material、生产 Token/数据进入进程或工件 | 立即安全停止并销毁含值临时工件 | 不提交、不发布；通知 owner，保留脱敏事件 |
| 敏感宿主变量被继承、config 启用生产能力 | 启动前失败关闭 | 不启动 Gateway/Vite/Browser；修正测试夹具计划 |
| 非 loopback request、WebSocket、proxy hit 或 socket | 立即中止运行 | 保存脱敏 origin/PID 摘要；关闭所有测试进程 |
| sentinel scan 命中拟保留工件 | 销毁工件并判 Gate 失败 | 不索引、不事后编辑 raw artifact |
| R5 patch 与 R4 已验收内容不一致 | 停止浏览器 | 只反向撤销 R5 精确迁移；不 reset 用户文件 |
| Playwright/Chromium 获取失败 | 标记 environment-blocked | 不改 repo 依赖，不换自动化框架 |
| 登录、WS、刷新、退出或 Console/HTTP Gate 失败 | 固化脱敏失败 Evidence | 新建 Issue 与 plan-r008；r007 不修产品 |
| 浏览器要求新增 fixture/product path | 停止范围扩张 | 先建新 Plan revision |
| 临时文件进入 repo 或 lockfile 改变 | 停止并清理明确的任务临时文件 | 不删除用户文件；重跑 scope/lockfile Gate |
| Gateway/Vite 子进程未退出 | 不宣称完成 | 按记录 PID 定向终止；复核监听端口 |

## 退出门禁

- [ ] R5 source-first channel hygiene、r006 全量确定性 Gate 与 scope/lockfile 通过。
- [ ] `FE-EVID-W01-014` 证明 R5 全量确定性、scope 与 lockfile 重验通过。
- [ ] `FE-EVID-W01-015` 证明本地 Playwright 和一次性 synthetic runtime 可复核。
- [ ] Desktop 与 Mobile 的登录、connected、刷新恢复、退出全通过。
- [ ] 页面身份、非空、无 overlay、Console、HTTP/WS 与截图 Gate 全通过。
- [ ] 临时 Token/状态未进入仓库，所有子进程与监听端口已关闭。
- [ ] Browser Evidence 通过独立代码/安全/文档复核。
- [ ] `FE-W01-S08` owner Evidence 通过。

前七项通过但 S08 未完成时，S05 可标记 `complete`，W01 仍必须保持 `active`；
产品补丁仍不得提交或发布。
