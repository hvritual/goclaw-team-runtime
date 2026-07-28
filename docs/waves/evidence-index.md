# Wave 证据索引

本文件只保存证据元数据和稳定引用，不复制 Token、完整生产日志或大型二进制。
原始证据应进入受控 Trace、EvidencePackage、CI 工件或脱敏测试报告。

## 状态

| Evidence ID | Wave | Issue | 类型 | 通过条件 | 位置 | 状态 | 复核者 |
|---|---|---|---|---|---|---|---|
| `FE-EVID-W00-001` | `FE-W00` | `FE-ISSUE-001` | coverage matrix | 九个页面及共享基础设施都有验证负责人、环境和用例 | [W00 r003 责任迁移](frontend-stability/fe-w00/plan-r003.md#稳定-step-处理) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-002` | `FE-W00` | 待拆分 | reproduction bundle | 每个异常都有步骤、期望、实际、RPC、日志和截图 | [W00 r003 责任迁移](frontend-stability/fe-w00/plan-r003.md#稳定-step-处理) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-003` | `FE-W00` | 待拆分 | contract baseline | 前端调用与 Gateway 注册、RBAC、返回结构逐项对应 | [静态契约预备清单](frontend-stability/fe-w00/contract-inventory-preliminary.md) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-004` | `FE-W00` | 待拆分 | fixture reset | 一次性项目、角色和状态夹具可重复建立并清理 | [W00 r003 责任迁移](frontend-stability/fe-w00/plan-r003.md#稳定-step-处理) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-005` | `FE-W00` | `FE-ISSUE-001` | environment baseline | Git、工具链、构建、运行拓扑和浏览器环境可冻结 | [环境与构建基线](frontend-stability/fe-w00/baseline-manifest.md) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-006` | `FE-W00` | 待拆分 | runtime harness preflight | 同源运行入口、一次性夹具、现有测试与浏览器阻塞均明确 | [运行夹具预检](frontend-stability/fe-w00/runtime-harness-preflight.md) | `superseded` | `wave_transition_review` |
| `FE-EVID-W00-007` | `FE-W00` | `FE-ISSUE-002`–`004` | authority/build/protocol reproduction | 0.7.0 Git、Go/UI build、Gateway test 阻断和真实 Vite HTTP/WS 行为可复核 | [权威基线与首批复现](frontend-stability/fe-w00/authority-runtime-reproduction.md) | `passed` | `wave_transition_review` |
| `FE-EVID-W01-001` | `FE-W01` | `FE-ISSUE-003`、`004` | Node transport tests | TeamClient URL/protocol 与 Vite transport 确定性测试先失败后通过 | [R4 deterministic gates](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-002` | `FE-W01` | `FE-ISSUE-002`–`004` | Gateway security tests | Origin、CSRF、browser session、personal token、TeamGuard/RBAC 与 race 全通过 | [R4 deterministic gates](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-003` | `FE-W01` | `FE-ISSUE-003`、`004` | browser | 登录、connected、刷新与退出在真实可达页面通过 | 当前 Browser localhost policy blocked | `planned` | unassigned |
| `FE-EVID-W01-004` | `FE-W01` | `FE-ISSUE-003`、`004` | real Vite proxy | `/auth` 保留同源 Host 返回 204，`/ws` 保留同源 Host 返回 101 | [R4 deterministic gates](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-005` | `FE-W01` | `FE-ISSUE-002`–`004` | scope/build gate | UI build、全 Go、lockfile 与 tracked/untracked allowlist 通过 | [R4 deterministic gates](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-006` | `FE-W01` | `FE-ISSUE-002`–`004` | red-test baseline | 旧签名测试恢复编译；两个 transport tests 在未修实现上按根因失败；安全负例通过 | [S01 red tests](frontend-stability/fe-w01/s01-red-tests.md) | `passed` | independent review pending |
| `FE-EVID-W01-007` | `FE-W01` | `FE-ISSUE-005` | dashboard shell reproduction | `/dashboard/` 重复返回 301 `Location: ./`；Location 解析回同一 URL；根因与冻结 base 对应 | [S04 dashboard shell reproduction](frontend-stability/fe-w01/s04-dashboard-shell-reproduction.md) | `passed` | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-008` | `FE-W01` | `FE-ISSUE-005` | dashboard shell red/green regression | canonical route、GET/HEAD shell、完整安全头、缓存分层、真实 JS/CSS asset 与 unknown 404 先失败后通过 | [S06 shell red/green](frontend-stability/fe-w01/s06-dashboard-shell-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-009` | `FE-W01` | `FE-ISSUE-006`、`007` | full-Go/channels reproduction | 30 秒失败关闭 timeout、10 分钟 sleep、真实网络调用和 credential-shaped literals 脱敏归因 | [S04 channels reproduction](frontend-stability/fe-w01/s04-full-go-channels-reproduction.md) | `passed` | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-010` | `FE-W01` | `FE-ISSUE-006`、`007` | deterministic channel test red/green | synthetic config、required-field/default/init assertions，无 network/unbounded-goroutine/sleep/credential material | [S07 deterministic channel green](frontend-stability/fe-w01/s07-deterministic-channel-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-011` | `FE-W01` | `FE-ISSUE-007` | external credential owner attestation | owner 记录 credential-shaped material 已撤销/轮换，或权威证明从未有效；不回显原值 | 待 credential owner 产生 | `planned` | unassigned |
| `FE-EVID-W01-012` | `FE-W01` | `FE-ISSUE-008` | full-Go/catalog reproduction | 全仓失败与两次目标复现一致；根因和冻结 base 归属可复核 | [S04 catalog reproduction](frontend-stability/fe-w01/s04-memory-catalog-reproduction.md) | `passed` | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-013` | `FE-W01` | `FE-ISSUE-008` | catalog provenance red/green | 默认/显式/git/custom source kind 推断矩阵、package/race 与全仓通过 | [S09 catalog red/green](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) | `passed` | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-014` | `FE-W01` | `FE-ISSUE-002`–`008` | R5 deterministic revalidation | source-first channel hygiene、R4 manifest、channels/Catalog/Gateway race、全仓、vet、UI、scope 与 lockfile 在 R5 独立通过 | [R5 确定性重验](frontend-stability/fe-w01/s10-r5-deterministic-revalidation.md) | `collecting` | independent review pending |
| `FE-EVID-W01-015` | `FE-W01` | `FE-ISSUE-003`、`004`、`005` | local Playwright browser bundle | 最小环境和 loopback-only synthetic runtime 下 Desktop/Mobile 登录、connected、刷新恢复、退出、页面身份、Console、HTTP/WS、sentinel scan、截图与两阶段清理均通过 | 待 r009/R7 `S10/S05` 产生 | `planned` | unassigned |
| `FE-EVID-W01-016` | `FE-W01` | `FE-ISSUE-003`、`004`、`005` | runtime provider source preflight | 在 R5 Task Base/HEAD 上按真实启动源码路径证明 provider、非空 constructor marker 与 model name 必需；r007 未启动 runtime 并安全停止 | [S10 provider preflight](frontend-stability/fe-w01/s10-runtime-provider-preflight.md) | `passed` | `r4_code_review` + `r4_security_review` + `wave_docs_validate` |
| `FE-EVID-W01-017` | `FE-W01` | `FE-ISSUE-002`–`008` | R6 deterministic revalidation | 技术 Gate 与代码复核通过，但 R6 freeze/commit traceability 失败，不能作为后续验收链 | [R6 deterministic revalidation](frontend-stability/fe-w01/s11-r6-deterministic-revalidation.md) | `failed` | technical code/security approved；governance rejected |
| `FE-EVID-W01-018` | `FE-W01` | `FE-ISSUE-003`、`004`、`005`、`010` | inert provider syscall gate | 下载执行树 manifest 全部匹配；sandbox、首次授权宿主及用户再次触发的授权宿主重试均拒绝 ptrace；未创建 credential/runtime，未启动服务 | [S11 syscall Gate environment-blocked](frontend-stability/fe-w01/s11-inert-provider-environment-blocked.md) | `failed` | retry delta: `r4_code_review` + `r4_security_review` + `wave_docs_validate` |
| `FE-EVID-W01-019` | `FE-W01` | `FE-ISSUE-009` | R6 traceability preflight | freeze 缺 Repository/policy hash，三个 commit 缺 mandatory trailers；runtime 前失败关闭 | [R6 traceability preflight](frontend-stability/fe-w01/s12-r6-traceability-preflight.md) | `passed` | `r4_code_review` + `r4_security_review` + `wave_docs_validate` |
| `FE-EVID-W01-020` | `FE-W01` | `FE-ISSUE-002`–`009` | R7 traceable deterministic revalidation | 完整 Task tuple/trailers/ancestry、source-first、11 manifest、Go/race/vet、UI、scope 与 lockfile 通过；产品未提交 | [R7 traceable deterministic revalidation](frontend-stability/fe-w01/s12-r7-traceable-deterministic-revalidation.md) | `passed` | `r4_code_review` + `r4_security_review` + `wave_docs_validate` |
| `FE-EVID-W02-001` | `FE-W02` | 待绑定 | contract tests | 查询页面对合法、空、拒绝、部分失败响应均行为正确 | 待 W02 产生 | `planned` | unassigned |
| `FE-EVID-W03-001` | `FE-W03` | 待绑定 | workflow tests | 高风险命令幂等、职责分离、刷新和失败恢复通过 | 待 W03 产生 | `planned` | unassigned |
| `FE-EVID-W04-001` | `FE-W04` | 待绑定 | browser matrix | 桌面/移动、键盘、读屏语义和断线状态通过 | 待 W04 产生 | `planned` | unassigned |
| `FE-EVID-W05-001` | `FE-W05` | 全部纳入 Issue | release gate | 前端、Go、权限、浏览器、归档和部署回归全部通过 | 待 W05 产生 | `planned` | unassigned |
| `FE-EVID-W05-006` | `FE-W05` | `FE-ISSUE-007` | credential owner revalidation | 独立复核 `FE-EVID-W01-011` 仍适用于候选 release，且归档不含当前材料 | 待 W05 产生 | `planned` | unassigned |
| `PILOT-EVID-001` | `PILOT-W00` | `PILOT-ISSUE-001`–`013` | baseline/capability | R7 source、工具版本、三路审计、strace/Browser 能力准确记录 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | root + three bounded implementation reviews |
| `PILOT-EVID-002` | `PILOT-W00` | `PILOT-ISSUE-001`、`002` | Runner security/platform | platform/path/wrapper/TMP/Git/process group 正负例、race 与双架构通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | `runner_xplat_impl` + root revalidation |
| `PILOT-EVID-003` | `PILOT-W00` | `PILOT-ISSUE-003`、`008`–`010` | governance | Wave binding、planner-service、bypass 与 terminal guard 正负例通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | governance implementation review + root revalidation |
| `PILOT-EVID-004` | `PILOT-W00` | `PILOT-ISSUE-004`、`005` | recovery | consistency、cold backup roundtrip、tamper/缺件/运行中拒绝通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | root deterministic revalidation |
| `PILOT-EVID-005` | `PILOT-W00` | `PILOT-ISSUE-006`、`007`、`013` | UI/RPC | scope/auth/history/seq/无碰撞键/安全迁移/刷新/mutation tests 与 UI build 通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | `pilot_frontend_impl` + root revalidation |
| `PILOT-EVID-006` | `PILOT-W00` | `PILOT-ISSUE-001`–`010` | three-runner E2E | 三 owner 并发、签名 Evidence、严格 review/final、掉线恢复无串领 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `collecting` | deterministic pieces passed；three-device owner pending |
| `PILOT-EVID-007` | `PILOT-W00` | `PILOT-ISSUE-006`、`007` | browser | Desktop/Mobile 与三个独立 BrowserContext 的核心流程通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `failed` | cloud Browser localhost policy blocked；现场 owner pending |
| `PILOT-EVID-008` | `PILOT-W00` | `PILOT-ISSUE-001`–`010` | release artifact | Linux 双架构、控制 CLI 交叉编译、source/manifest/SHA256 通过 | [`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) | `passed` | build-release + root checksum recovery |
| `PILOT-EVID-009` | `PILOT-W00` | `PILOT-ISSUE-011` | external attestation | owner 证明历史材料已撤销/轮换或从未有效 | 待 credential owner 产生 | `planned` | unassigned |
| `PILOT-EVID-010` | `PILOT-W00` | `PILOT-ISSUE-001`、`002` | target substrate smoke | 真机 native Linux、WSL2 与 Lima 均通过 doctor、无网络 bwrap 和中断清理 | 待三名试点 owner 产生 | `planned` | unassigned |
| `PILOT-EVID-011` | `PILOT-W00` | 试点外部 Gate | credential/integration | 三名成员各自 Codex OAuth、真实飞书路由和可选 Obsidian Desktop 交互通过 | 待三名试点 owner 与飞书管理员产生 | `planned` | unassigned |
| `MVP-EVID-001` | `MVP-W00` | `MVP-ISSUE-001` | provenance | 原归档 SHA、611 个文件、内容、执行位和 Git import tree 一致 | [`SOURCE_PROVENANCE`](../recovery/SOURCE_PROVENANCE.md) | `passed` | r005 deterministic revalidation；final review pending |
| `MVP-EVID-002` | `MVP-W00` | `MVP-ISSUE-001` | Go test/race/vet | 全包 test、6 个关键包 race/vet 通过 | [`RECOVERY_GATE_REPORT`](../recovery/RECOVERY_GATE_REPORT.md) | `passed` | r005 deterministic revalidation；final review pending |
| `MVP-EVID-003` | `MVP-W00` | `MVP-ISSUE-001` | Web test/build | Web 8/8、build 通过且 tracked bundle 无 diff | [`RECOVERY_GATE_REPORT`](../recovery/RECOVERY_GATE_REPORT.md) | `passed` | r005 deterministic revalidation；final review pending |
| `MVP-EVID-004` | `MVP-W00` | `MVP-ISSUE-001` | Obsidian test/build | Adapter 6/6、build 通过且 tracked `main.js` 无 diff | [`RECOVERY_GATE_REPORT`](../recovery/RECOVERY_GATE_REPORT.md) | `passed` | r005 deterministic revalidation；final review pending |
| `MVP-EVID-005` | `MVP-W00` | `MVP-ISSUE-001` | release rebuild | 锁、stage、规范化归档、精确合同、原子版本目录和同 commit 双构建通过 | [`RECOVERY_GATE_REPORT`](../recovery/RECOVERY_GATE_REPORT.md) | `passed` | r005 candidate `e262b8c`；final review pending |
| `MVP-EVID-006` | `MVP-W00` | `MVP-ISSUE-001`–`003` | independent review | 三路只读复核无未关闭 P0/P1 | [`RECOVERY_REVIEW`](../recovery/RECOVERY_REVIEW.md) | `passed` | r005 `526e14b` / `524c764`；code/security/docs PASS |

证据状态允许：`planned`、`collecting`、`passed`、`failed`、`superseded`。

## Evidence ID 规则

格式为 `<TRACK>-EVID-...`，例如 `FE-EVID-W01-020`、
`PILOT-EVID-010`、`MVP-EVID-006`。证据失败后不覆盖结果；新增子证据 ID
或追加执行记录，并用 Wave Journal 说明它替代了哪次运行。

## 脱敏要求

证据入库前必须移除：

- Gateway、Team、Reviewer Token；
- Cookie、CSRF Token、Runner device key；
- Codex OAuth、SSH agent、云凭据；
- 用户私人消息和未授权项目数据；
- 本地绝对路径中可识别个人的信息。
