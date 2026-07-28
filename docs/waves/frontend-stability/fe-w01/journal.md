# FE-W01 Journal

本文件只追加；当前计划为 [`plan-r011.md`](plan-r011.md)，状态
`blocked`。r009 及更早执行历史只读保留；在 recovered base 上批准新
revision 前，不得迁移产品 patch 或启动本地 Playwright。

## 2026-07-28 — r011 批准归因更正

- r010 把第一轮 BLOCK reviewer 误写入 `approved_by`；
- 新建 r011，只保留用户授权；
- blocked、MVP-W00 依赖、docs-only 范围和外部 Gate 全部不变。

## 状态事件

| Seq | 时间 | Actor | From | To | 原因 | Evidence |
|---:|---|---|---|---|---|---|
| 1 | 2026-07-26 | planning |  | `planned` | 等待 W00 复现基线和 Issue 绑定 | `FE-DEC-004` |
| 2 | 2026-07-26 | Wave governance | `planned` | `active` | r003 通过独立 Wave 与安全复核，冻结任务进入 S01 | `FE-DEC-007`、`FE-EVID-W00-007` |
| 3 | 2026-07-26 | Wave governance | `active` | `active` | r004 通过第二轮 Wave、安全与文档一致性复核；扩大范围但不跳过 Task Base 门禁 | `FE-DEC-008`、`FE-EVID-W01-007` |
| 4 | 2026-07-26 | Wave governance | `active` | `active` | r005 通过 Wave、安全与文档终审；只新增 test-only S07 和当前 W01 的外部 owner S08，不放宽 Browser/安全门禁 | `FE-DEC-009`、`FE-EVID-W01-009` |
| 5 | 2026-07-26 | Wave governance | `active` | `active` | r006 通过 Wave、安全与文档终审；只新增 S09 两个 Memory 路径，不包含数据迁移且不放宽 owner/Browser 门禁 | `FE-DEC-010`、`FE-EVID-W01-012` |
| 6 | 2026-07-26 | Wave governance | `active` | `active` | r007 通过 Wave、安全与文档终审；只新增仓库外 S10/S05 local Playwright 回归，不放宽 S08/发布门禁 | `FE-DEC-011`、`FE-EVID-W01-014`、`FE-EVID-W01-015` |

## Change log

| Change ID | 时间 | 提出人 | 内容 | Material | 影响 | 新 Plan revision | 决策 |
|---|---|---|---|---:|---|---|---|
| `FE-CHG-W01-001` | 2026-07-26 | Codex | 将首批范围收敛为测试编译、Vite `/auth` Host 与 DEV `/ws` transport | yes | 新增 `ui/vite.config.ts`、`ui/tests/**`、package scripts 与两个 Gateway 测试文件；其余 r001 能力延后 | `plan-r002` | `FE-DEC-006` |
| `FE-CHG-W01-002` | 2026-07-26 | independent reviewers | 补稳定 Step、真实 WS 101、Origin/CSRF/session/token/RBAC、范围与回滚 Gate | yes | 删除未证明的 `transport.ts`/lockfile 变更权限；新建 r003 | `plan-r003` | `FE-DEC-007` |
| `FE-CHG-W01-003` | 2026-07-26 | S04 deterministic gate | 全 Gateway Gate 复现 `/dashboard/` 301 自循环，新增 shell route/test 两个精确路径 | yes | r003 S04 停止；登记 `FE-ISSUE-005`，候选 Task Revision 2 与 plan-r004；独立批准前不改产品 | `plan-r004` | `FE-DEC-008` |
| `FE-CHG-W01-004` | 2026-07-26 | S04 full-Go gate | channels test 固定 sleep 10m 且含 credential-shaped literals，新增一个精确 test path | yes | r004 S04 停止；登记 `FE-ISSUE-006/007`、候选 Task rev 3 与 plan-r005；不跳过 package 或提高 timeout | `plan-r005` | `FE-DEC-009` |
| `FE-CHG-W01-005` | 2026-07-26 | R3 S04 full-Go gate | Memory Catalog 默认 provenance kind 推断为 `markdown-markdown`，新增两个精确 Memory 路径 | yes | r005 S04 停止；登记 `FE-ISSUE-008`、候选 Task rev 4 与 plan-r006；不排除 package，不做数据迁移 | `plan-r006` | `FE-DEC-010` |
| `FE-CHG-W01-006` | 2026-07-26 | 用户明确授权 | Cloud Browser localhost policy 阻塞后，建立仓库外本地 Playwright 回归 Step | yes | 候选 Task rev 5 与 plan-r007；固定 synthetic runtime、桌面/移动与失败另起 r008；S08/发布阻塞不变 | `plan-r007` | `FE-DEC-011` |

## Decision log

| Decision ID | 时间 | 问题 | 备选方案 | 选择 | 最强反方论点 | Evidence | 决策人 |
|---|---|---|---|---|---|---|---|
| `FE-DEC-006` | 2026-07-26 | 首批修复是否同时重做全部 session/shell | 全量实施、最小 transport slice | 最小 transport slice | 后续仍需继续 W01 | `FE-ISSUE-002`–`004` 已 root-caused，其余未复现 | user directive + Codex |
| `FE-DEC-007` | 2026-07-26 | r002 安全回归是否足够 | 直接实施、扩大范围、补最小负例矩阵 | 补最小负例矩阵 | 测试实现更复杂 | 两路 reviewer 的一致阻断意见 | independent reviewers |
| `FE-DEC-008` | 2026-07-26 | S04 发现的 shell baseline failure 是否顺手修复 | 忽略、在 r003 越权修改、冻结并新建 revision | 冻结 S04，新建 Issue/Evidence/r004 | 推迟全量 Gate，增加 worktree 迁移成本 | `FE-EVID-W01-007` 证明核心 shell 自循环，相关路径不在 r003 allowlist | `wave_transition_review` + `transport_security_review` |
| `FE-DEC-009` | 2026-07-26 | channels hang 是否通过排除 package/延长 timeout 绕过 | 排除、等待十分钟、改确定性 test | 新增 r005/S07，只改 test | 再次迁移 worktree 增加成本 | `FE-EVID-W01-009` 证明 deterministic gate 与 credential hygiene 均不满足 | user directive + `wave_transition_review` + `transport_security_review` |
| `FE-DEC-010` | 2026-07-26 | Catalog 失败是否排除 package 或在 r005 越权修复 | 排除、顺手修、建立 r006 | 建立 r006/S09 与 Task rev 4 | 再迁移一次 worktree 增加成本 | `FE-EVID-W01-012` 两次稳定复现且 Memory 路径与 base 一致 | user directive + `wave_transition_review` + `transport_security_review` |
| `FE-DEC-011` | 2026-07-26 | Cloud Browser 拒绝 localhost 后如何完成 S05 | 无限等待、改用未授权通道、仓库外本地 Playwright | 用户授权本地 Playwright并建立 r007/S10 | 需要临时安装浏览器并再次迁移 worktree | Cloud Browser security policy rejection；R4 deterministic/independent gates passed | user directive；independent review pending |

## Evidence ledger

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `FE-EVID-W01-006` | 2026-07-26 | `FE-W01-S01` / `FE-ISSUE-002`–`004` / `FE-W01-TRANSPORT-R1` | [`s01-red-tests.md`](s01-red-tests.md) | `51f55ee067f0fd0dafae7619efde1f8048bdc659ba8721950ae02e74564ded07` | Node transport 先按两个根因失败；Gateway 编译恢复且安全控制通过 | `passed`；等待独立复核 | Codex | unassigned |
| `FE-EVID-W01-007` | 2026-07-26 | `FE-W01-S04` / `FE-ISSUE-005` / `FE-W01-TRANSPORT-R1` | [`s04-dashboard-shell-reproduction.md`](s04-dashboard-shell-reproduction.md) | `5f449f4b0693dc47112b4cbec14bdf4873a2d35110485bf0a6c5295728329e82` | `/dashboard/` 301 `Location: ./` 自循环、base 归属与 FileServer 根因可重复复核 | `passed` | Codex | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-008` | 2026-07-26 | `FE-W01-S06` / `FE-ISSUE-005` / `FE-W01-TRANSPORT-R1` rev 2 | 待 S06 产生 |  | shell route/security/cache 红绿回归 | `planned` | Codex | unassigned |
| `FE-EVID-W01-008` | 2026-07-26 | `FE-W01-S06` / `FE-ISSUE-005` / `FE-W01-TRANSPORT-R1` rev 2 | [`s06-dashboard-shell-red-green.md`](s06-dashboard-shell-red-green.md) | `2265b35cbbe27114883da075991ddc089484eaf4c6c0a81b11ab03d8f6d14f92` | GET/HEAD、安全头、asset 与 unknown-route 合同已加入；仅 slash GET 以 301 精确失败 | `collecting`；green pending | Codex | unassigned |
| `FE-EVID-W01-008` | 2026-07-26 | `FE-W01-S06` / `FE-ISSUE-005` / `FE-W01-TRANSPORT-R1` rev 2 | [`s06-dashboard-shell-red-green.md`](s06-dashboard-shell-red-green.md) | `a94fe604b265797b9e25d882f5c2ba5eb9f8841137ca1e99dd81dea2d7042bbf` | shell GET/HEAD、完整安全头、asset/cache 与 unknown 404 已按同一合同 red→green | `passed` | Codex | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-009` | 2026-07-26 | `FE-W01-S04` / `FE-ISSUE-006/007` / `FE-W01-TRANSPORT-R1` rev 2 | [`s04-full-go-channels-reproduction.md`](s04-full-go-channels-reproduction.md) | `4395d6c2dee91ad70f9f2bf7d0abf350577c5136738bbfec326e4e69e69035e6` | channels 30s timeout、10m sleep、network start 与 credential-shaped literals 脱敏归因；EOF 规范化，不改变证据语义 | `passed` | Codex | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-008` | 2026-07-26 | red phase immutable provenance correction | Commit `88e051e8309a33f79ef6c1fc745c942bcc1d4edb`，path `docs/waves/frontend-stability/fe-w01/s06-dashboard-shell-red-green.md` | `2265b35cbbe27114883da075991ddc089484eaf4c6c0a81b11ab03d8f6d14f92` | 前一 red-phase 行的相对链接现指向 green 内容；该 commit/path 可恢复当时原文 | `passed` | Codex | unassigned |
| `FE-EVID-W01-010` | 2026-07-26 | `FE-W01-S07` / `FE-ISSUE-006/007` / `FE-W01-TRANSPORT-R1` rev 3 | [`s07-deterministic-channel-green.md`](s07-deterministic-channel-green.md) | `58ab93d1610a4ad073e427644123ba4d2bbae6e7fe7ed9587271ccd01f567622` | source-first Gate、channels/race、Gateway/race 和 UI transport/build 通过；不包含 raw deletion diff | `passed` | Codex | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-012` | 2026-07-26 | `FE-W01-S04` / `FE-ISSUE-008` / `FE-W01-TRANSPORT-R1` rev 3 | [`s04-memory-catalog-reproduction.md`](s04-memory-catalog-reproduction.md) | `ab99d7e555434a79bc2a46b94046905d1913c802e15efcff7b0d7dc55e69d6af` | 全仓失败、两次目标复现、默认 kind 根因和冻结 base 归属 | `passed` | Codex | `wave_transition_review` + `transport_security_review` |
| `FE-EVID-W01-013` | 2026-07-26 | `FE-W01-S09/S04` / `FE-ISSUE-008` / `FE-W01-TRANSPORT-R1` rev 4 | [`s09-catalog-provenance-red-green.md`](s09-catalog-provenance-red-green.md) | `993b0d58e4c0969510e5fb440b81d6bbafd08a1f8aa2f63192bbf98d57d7acfc` | Catalog 推断矩阵 red→green、package/race、channels/Gateway/race、全仓、UI、scope 与 lockfile 全绿 | `passed` | Codex | `r4_code_review` + `r4_security_review` |
| `FE-EVID-W01-014` | 2026-07-26 | `FE-W01-S01/S02/S03/S04/S06/S07/S09` / `FE-ISSUE-002`–`008` / `FE-W01-TRANSPORT-R1` rev 5 | [`s10-r5-deterministic-revalidation.md`](s10-r5-deterministic-revalidation.md) | `8b5147b7ffcdc5d40c7c7d0036c6fc66f1346849f7a999ed0cf0db42907b74a7` | source-first 迁移、11 文件 manifest、channels/Catalog/Gateway race、全仓、vet、UI、scope 与 lockfile 全绿 | `collecting`；independent review pending | Codex | unassigned |

## 进度事件

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-26 | plan | `proposed → planned` | 仅完成候选计划；禁止实施 | 等待 FE-W00 complete |
| 2026-07-26 | plan r002 | `planned` | 首批 Issue、允许路径、先失败测试、回滚与安全不变量已冻结 | 独立复核 W00 Evidence 与 W01 r002 |
| 2026-07-26 | plan r002 review | `review → changes-requested` | 修复方向获批，但测试/范围/回滚不足，未激活 | 复核 plan r003 |
| 2026-07-26 | plan r003 review | `review → approved` | Wave 与安全 Reviewer 均批准执行；Browser exit gate 保持 blocked | 原子切换 Registry 并冻结 Task |
| 2026-07-26 | activation | `planned → active` | Base `b288564361fac4f09d65e2a6a7ff80362a5cc12e`；Plan `FE-W01 plan-r003` SHA `dbe0aab4842f0c7730be18d33917c1bbda5c9610baf15709b298f3a76092b275`；Task `FE-W01-TRANSPORT-R1` rev 1；Steps `FE-W01-S01/S02/S03/S04`；Issues `FE-ISSUE-002/003/004` | 执行专用 worktree 的先失败测试 |
| 2026-07-26 | task worktree | `created` | Branch `repair/fe-w01-transport-r1`；path `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r1`；HEAD/base `b288564361fac4f09d65e2a6a7ff80362a5cc12e`；allowlist 为 r003 的 7 项 | 带入 docs-only activation commit 后开始 S01 |
| 2026-07-26 | `FE-W01-S01` | `planned → active` | 只允许先添加失败测试和修复两个旧测试调用点；尚无产品实现变更 | 先观察三个独立失败 |
| 2026-07-26 | environment | `npm ci retry` | 默认 npm cache 不可写并留下不完整 ignored directory；移到 `/tmp` 后使用任务专用 cache 成功安装，lockfile SHA 保持 `46fd937f...` | 不登记为产品 Issue；继续 S01 |
| 2026-07-26 | `FE-W01-S01` | `active → complete` | TeamClient URL 与 `/auth` proxy 两个 Node red tests 精确失败；Gateway session/security tests 通过 | 执行 S02 的 proxy 配置修复 |
| 2026-07-26 | `FE-W01-S02` | `planned → active` | 只允许修改 `ui/vite.config.ts` 的 `/auth` 与 `/ws` changeOrigin | 修复后要求 proxy test 通过，client test 仍保持红色 |
| 2026-07-26 | `FE-W01-S02` | `active → complete` | 真实 Vite auth/WS proxy test 通过：auth 204、WS 101、两者 Host 均等于 Origin.host；TeamClient 仍因 `:28789` 单独失败 | 执行 S03 的同源 client URL 修复 |
| 2026-07-26 | `FE-W01-S03` | `planned → active` | 只允许修改 `ui/src/team/client.ts` 的 WebSocket URL；protocol 与 token 边界由现有 red test 锁定 | 运行完整 Node transport tests |
| 2026-07-26 | `FE-W01-S03` | `active → complete` | TeamClient 同源 URL test 与真实 Vite auth/WS proxy test 均通过；subprotocol 仍为 `goclaw.v1` 且不含 token | 执行 S04 全量确定性 Gate |
| 2026-07-26 | `FE-W01-S04` | `planned → active` | 开始 UI build、全 Gateway、race、全 Go、lockfile 和 tracked/untracked allowlist | 任何失败都停止 Wave 推进 |
| 2026-07-26 | `FE-W01-S04` | `active → blocked-scope` | 全 Gateway Gate 在既有 `TestTeamConsoleShellIsPublicButHardened` 失败：`/dashboard/` 返回 301；两次重复、response probe、URL 解析和 base 哈希确认 `FE-ISSUE-005` | 先评审 plan-r004；禁止在 r003 修改 shell 路径 |
| 2026-07-26 | plan r004 | `drafted → review` | 候选 Task `FE-W01-TRANSPORT-R1` rev 2；新增稳定 S06 和精确路径 `gateway/ui.go`、`gateway/server_auth_test.go`；Browser Gate 不变 | Wave 与 Security Reviewer 独立评审 |
| 2026-07-26 | plan r004 review 1 | `review → changes-requested` | 根因、new revision、S06 与最小范围获认可；阻断为合法状态、Task Base/累计 base、原子同步、双层范围/gofmt Gate，以及 GET/HEAD/Content-Type/完整 CSP/asset/unknown-404 合同 | 在未批准的 r004 draft 内补齐后重新独立评审；产品代码仍禁止修改 |
| 2026-07-26 | plan r004 review 2 | `review → approved` | Wave 与 Security Reviewer 无剩余阻断；文档 validator 确认 JSON/YAML、49 个相对链接、Evidence SHA、Task rev2 与双 base 语义闭合 | 原子激活 r004 并生成 docs-only Task Base |
| 2026-07-26 | plan r004 activation | `r003 → r004` | Plan SHA `8ca3a6255eebf86993a82230a6926b7512e6279fc48eb167a3be5a1191a3780c`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Task `FE-W01-TRANSPORT-R1` rev 2；Steps `S01/S02/S03/S06/S04`；Issues `002/003/004/005`；allowlist 为 4 个 UI/Node path、4 个 Gateway/test path 和 `docs/waves/**` | 形成 docs-only activation commit；R2 创建时把其精确 SHA 冻结为 Task Base |
| 2026-07-26 | task worktree revision 2 | `created → frozen` | Task Base/creation HEAD `1cc3c1188271f084e6412d62ef18d4edaf775193`；branch `repair/fe-w01-transport-r2`；path `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r2`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Plan r004 SHA `8ca3a6255eebf86993a82230a6926b7512e6279fc48eb167a3be5a1191a3780c`；Task `FE-W01-TRANSPORT-R1` rev 2；Assignee `Codex root agent`；Policy `wave-governance-v1`；Steps `S01/S02/S03/S06/S04`；Issues `002/003/004/005`；精确 allowlist 为 r004 frontmatter 的 9 项 | 先形成 docs-only freeze commit，再只迁移 allowlist 内 R1 patch |
| 2026-07-26 | `FE-W01-S06` | `ready-after-task-base → active` | Task Base 与 revision-specific worktree 已冻结；测试先行规则生效，`gateway/ui.go` 在新增 shell 合同精确失败前仍禁止修改 | 迁移 S01–S03 patch、建立 R2 独立依赖后，仅修改 `server_auth_test.go` 采集红测 |
| 2026-07-26 | `FE-W01-S01/S02/S03` R2 | `migrated → reverified` | R2 独立 npm install；Node transport 2/2、Origin/CSRF/browser-session/personal-token 目标测试通过；lockfile SHA 保持 `46fd937f...` | 进入 S06 red phase |
| 2026-07-26 | `FE-W01-S06` red phase | `active → red-confirmed` | 新 shell 合同只在 slash GET 精确失败：301 `Location: ./`；canonical redirect、真实 JS/CSS 与 unknown 404 通过 | 索引 red Evidence 后允许最小修改 `gateway/ui.go` |
| 2026-07-26 | `FE-W01-S06` green phase | `red-confirmed → complete-in-r2` | 最小 `ui.go` patch 后 GET/HEAD、CSP、安全头、MIME/length、canonical、JS/CSS cache 与 unknown 404 全部通过；完整 Gateway、transport、UI build 通过 | 进入 S04 Gateway race 与 full-Go Gate |
| 2026-07-26 | `FE-W01-S04` full-Go | `active → blocked-scope` | `go test ./...` 超过 3m；隔离 channels 以 30s timeout 证明 `TestNewWeWorkWsBotChannel` 固定 sleep 10m，并发现 versioned credential-shaped literals | 先评审 r005；禁止修改 channel test 或跳过全仓 Gate |
| 2026-07-26 | plan r005 | `drafted → review` | 候选 Task `FE-W01-TRANSPORT-R1` rev 3；新增 S07 和精确路径 `channels/weworkwsbot_test.go`；生产 channel 与 Browser Gate不变 | Wave 与 Security Reviewer 独立评审 |
| 2026-07-26 | plan r005 review 1 | `review → changes-requested` | Issue/root cause、new revision 与 test-only scope 获认可；阻断为禁止重跑潜在凭据网络 test、raw diff/history 语义、owner 撤销/轮换、完整 Task tuple、source-first gates 与 Evidence provenance/mapping | 修订 draft r005 后重新评审；Registry 仍为 r004 |
| 2026-07-26 | `FE-ISSUE-007` security escalation | `root-caused` | 当前 Task 新增外部 `FE-W01-S08` / `FE-EVID-W01-011`；external owner blocker 已加入。credential owner 未解析，无法授权外部 WeWork 撤销/轮换或证明从未有效；FE-W05 以后只复核复用 | 用户/组织指派有权 owner；在 W01-011 前 W01/发布不能 complete |
| 2026-07-26 | plan r005 review 2 | `review → changes-requested` | 安全合同获批；Wave Reviewer 发现 W01 exit 依赖 W05 Evidence 会与 W02–W05 依赖链死锁 | 将 owner action 移为当前 active W01 的外部 S08/Evidence 011；W05 只在激活后复核 |
| 2026-07-26 | plan r005 review 3 | `review → approved` | 候选 SHA `e91665c9a01b55f582ebbb38f2ab1c2ea55d67f5b9eeb7115f9948f800d5df6e`；Wave、Security 与文档 validator 均无剩余阻断；owner/browser 外部门禁保持 | 原子激活 r005；仍不得在 R3 Task Base 冻结前修改产品或 channel test |
| 2026-07-26 | plan r005 activation | `r004 → r005` | Approved Plan SHA `16ab574fe033772fd0884aeaa155e0dc157dc6d7b1d7140575f3c526338b33e6`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Task `FE-W01-TRANSPORT-R1` rev 3；Steps `S01/S02/S03/S06/S07/S08/S04`；Issues `002/003/004/005/006/007`；allowlist 为 4 个 UI/Node path、4 个 Gateway/test path、1 个 channel test path 和 `docs/waves/**` | 形成 docs-only activation commit；R3 创建时把其精确 SHA 冻结为 Task Base |
| 2026-07-26 | task worktree revision 3 | `created → frozen` | Task Base/creation HEAD `2b0ead819b0a2b276b8c9de6779beb03d84767b5`；branch `repair/fe-w01-transport-r3`；path `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r3`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Plan r005 SHA `16ab574fe033772fd0884aeaa155e0dc157dc6d7b1d7140575f3c526338b33e6`；Task `FE-W01-TRANSPORT-R1` rev 3；Assignee `Codex root agent`；Policy `wave-governance-v1`；Steps `S01/S02/S03/S06/S07/S08/S04`；Issues `002/003/004/005/006/007`；精确 allowlist 为 r005 frontmatter 的 10 项 | 先形成 docs-only freeze commit，再只迁移 allowlist 内 R2 patch |
| 2026-07-26 | `FE-W01-S07` | `ready-after-task-base → active` | Task Base、revision-specific branch/worktree 与完整 tuple 已冻结；旧 channel test 只允许 SHA-only 确认，禁止再次执行或输出 raw deletion diff | 先迁移 R2 allowlist patch，再按 source-first 安全顺序实施 test-only hygiene |
| 2026-07-26 | `FE-W01-S07` R3 green | `active → complete-in-r3` | synthetic-only/source-first gates、channels 普通/race、Gateway 普通/race、UI transport/build 通过；`FE-EVID-W01-010` 已建立 | 进入 S04 全仓 Gate；独立代码/安全复核仍待完成 |
| 2026-07-26 | `FE-W01-S04` R3 full-Go | `active → blocked-scope` | `memory/catalog` 默认 Markdown provenance kind 稳定变成 `markdown-markdown`；两次目标复现；候选路径与所有冻结 base 相同 | 先评审 plan-r006；禁止修改 Memory 或排除 package |
| 2026-07-26 | plan r006 | `drafted → review` | 候选 Task `FE-W01-TRANSPORT-R1` rev 4；新增 S09 与精确路径 `memory/catalog/ingest.go`、`memory/catalog/service_test.go`；数据迁移、Browser 与 owner Gate 不变 | Wave、Security 与文档一致性独立评审 |
| 2026-07-26 | plan r006 review 1 | `review → changes-requested` | 初始候选 SHA `bef2622d0b091a5f3e91b123a6fd20f874065510a11679f33995992e87bac0bb`；安全阻断为 R4 expected legacy SHA/失败关闭/Delete-Add 未自包含，以及自定义 scheme/项目边界测试矩阵缺失；文档阻断为 Evidence 012 复现主体字段不足 | 只修订 draft；Registry 仍为 r005，Memory 继续冻结 |
| 2026-07-26 | plan r006 review 2 | `review → approved` | 最终候选 SHA `05a282359869104a56f528dfe5cc955cb4e1f6b1807d1908edfb8b16a3ffd8e7`；Wave、Security 与文档 validator 均无剩余阻断；S08/Browser 保持 | 原子激活 r006；仍不得在 R4 Task Base 冻结前迁移 patch 或修改 Memory |
| 2026-07-26 | plan r006 activation | `r005 → r006` | Approved Plan SHA `a562cfc4ff45dc3990211408ef9b5c88c9fb3d337f60fb9b4017a92caeb2f52b`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Task `FE-W01-TRANSPORT-R1` rev 4；Steps `S01/S02/S03/S06/S07/S08/S09/S04`；Issues `002/003/004/005/006/007/008`；allowlist 为 4 个 UI/Node path、4 个 Gateway/test path、1 个 channel test、2 个 Memory path 和 `docs/waves/**` | 形成 docs-only activation commit；R4 创建时把其精确 SHA 冻结为 Task Base |
| 2026-07-26 | task worktree revision 4 | `created → frozen` | Task Base/creation HEAD `dec9b07bece5e76e20130c9262a273edc41e851f`；branch `repair/fe-w01-transport-r4`；path `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r4`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Plan r006 SHA `a562cfc4ff45dc3990211408ef9b5c88c9fb3d337f60fb9b4017a92caeb2f52b`；Task `FE-W01-TRANSPORT-R1` rev 4；Assignee `Codex root agent`；Policy `wave-governance-v1`；Steps `S01/S02/S03/S06/S07/S08/S09/S04`；Issues `002/003/004/005/006/007/008`；精确 allowlist 为 r006 frontmatter 的 12 项 | 先形成 docs-only freeze commit，再按 source-first 顺序迁移 R3 patch |
| 2026-07-26 | `FE-W01-S09` | `ready-after-task-base → active` | Task Base、revision-specific branch/worktree 与完整 tuple 已冻结；Memory 修改仍须先迁移 synthetic channel、通过 source gate，并观察 Catalog 推断矩阵红测 | 执行 R4 安全迁移与 S09 red phase |
| 2026-07-26 | `FE-W01-S09` red phase | `active → red-confirmed` | 只新增推断矩阵后，既有/default/explicit Markdown 精确失败为 `markdown-markdown`；显式 kind、git、自定义 scheme、ProjectID/URI/collection 未出现额外失败 | 允许最小修改 `memory/catalog/ingest.go` |
| 2026-07-26 | `FE-W01-S09` green phase | `red-confirmed → complete-in-r4` | 最小 scheme switch 后，目标 `count=2`、全 package 与 race 通过；无 schema、数据迁移或项目边界变化 | 执行 S04 Revision 4 全量 Gate |
| 2026-07-26 | `FE-W01-S04` R4 deterministic gates | `active → complete-deterministic` | channels/race、Gateway/race、全仓 Go、go vet、R4 独立 UI transport/build、双 base/scope/gofmt/lockfile 全绿；`FE-EVID-W01-013` 已建立 | 独立代码与安全复核；S08/Browser 仍阻断 W01 complete 与产品提交 |
| 2026-07-26 | R4 independent acceptance | `review → approved` | `r4_code_review` 与 `r4_security_review` 均无 S0–S3 代码 finding，并独立复跑 channels/Catalog/Gateway race、UI transport、全仓、vet、scope 与 lockfile Gate | 只允许记录 docs-only Evidence；S08 owner 与真实 Browser 通过前，W01 仍 active，产品代码不得提交或发布 |
| 2026-07-26 | security hardening note | `noted` | 当前 CSP `connect-src` 允许 `ws:`/`wss:` scheme，客户端同源 URL 与服务端 Origin 校验仍构成当前有效边界 | 后续独立 hardening Wave 评估收紧；不在当前已冻结范围顺手修改 |
| 2026-07-26 | plan r007 | `drafted → review` | 用户明确授权 local Playwright fallback；候选 Task rev 5 新增 S10，脚本/浏览器/状态/截图全部仓库外，发现新缺陷必须另起 r008 | Wave、Security 与文档一致性独立评审；Registry 仍为 r006，浏览器尚未启动 |
| 2026-07-26 | plan r007 review | `review → approved` | 最终候选 SHA `334a0f1af9d1a384cb4fd5e2fba56bb9da5128236c052a169362c568dc5cba93`；Wave、Security 与文档 validator 均批准；最小环境、精确 loopback、sentinel、两阶段清理和 Evidence 014/015 已闭合 | 原子激活 r007；仍不得在 R5 Task Base 冻结前迁移 patch、安装 Playwright 或启动服务 |
| 2026-07-26 | plan r007 activation | `r006 → r007` | Approved Plan SHA `183cfd53e79841384af7323cc37cf2da924f5164bdece2b97684009e424770ec`；Task `FE-W01-TRANSPORT-R1` rev 5；Steps `S01/S02/S03/S06/S07/S08/S09/S04/S10/S05`；Issues `002/003/004/005/006/007/008`；allowlist 仍为 11 个产品路径和 `docs/waves/**` | 形成 docs-only activation commit；R5 创建时把其精确 SHA 冻结为 Task Base |
| 2026-07-26 | task worktree revision 5 | `created → frozen` | Task Base/creation HEAD `2f9cd8289d7c05e44d30b70a07e7991036229bbf`；branch `repair/fe-w01-transport-r5`；path `/workspace/scratch/afe5d81cd055/worktrees/fe-w01-transport-r5`；累计 base `697f50e5f428769b75061dfd859d2549dd1c330d`；Approved Plan r007 SHA `183cfd53e79841384af7323cc37cf2da924f5164bdece2b97684009e424770ec`；Task `FE-W01-TRANSPORT-R1` rev 5；Assignee `Codex root agent`；Policy `wave-governance-v1`；Steps `S01/S02/S03/S06/S07/S08/S09/S04/S10/S05`；Issues `002/003/004/005/006/007/008`；精确 allowlist 为 r007 frontmatter 的 12 项 | 先形成 docs-only freeze commit；随后严格按 legacy SHA-only、Delete/Add channel、source gate、其余 10 文件、11 文件 manifest 顺序迁移 |
| 2026-07-26 | R5 deterministic revalidation | `ready-after-task-base → complete-deterministic` | source-first channel migration、11/11 manifest、channels/Catalog/Gateway 普通/race、全仓、vet、UI 2/2/build、双 base scope 与 lockfile 全部通过；Evidence `014` 已生成 | 允许进入 S10 Playwright 下载准备；Evidence 014 与后续 Browser 结果仍需独立复核 |

## Post-R5 authority and recovery ledger

本节是 reconstruction base 之后的追加式权威记录。文件顶部保存的是 R5
时点的历史快照，不能被覆盖或改写。当前权威指针以
[`wave-registry.json`](../../wave-registry.json) 和本节末尾最新
`Authority pointer` 为准。

### 追加决策

| Decision ID | 时间 | 问题 | 选择 | Evidence / Review | 状态 |
|---|---|---|---|---|---|
| `FE-DEC-012` | 2026-07-26 | 真实 Gateway 强制构造 provider，r007 的“全部 provider 关闭”与真实启动不相容 | r007 在任何 token/config/runtime 前停止；以 r008/S11 固定公开 marker、不可达 loopback inert provider 和 syscall 零出站 Gate | `FE-EVID-W01-016`；`r4_code_review` + `r4_security_review` + `wave_docs_validate` | active |
| `FE-DEC-013` | 2026-07-26 | R6 freeze/commit 缺 Repository、policy hash 和 mandatory trailers | R6 只读保留；从 R5 trailers-present commit `5160273fb17502cf02cd10e1a17f5a47b7eb30be` 重建 r009/R7 | `FE-EVID-W01-019`；`r4_code_review` + `r4_security_review` + `wave_docs_validate` | active |

### 追加 Evidence

| Evidence ID | 时间 | Revision / Step | Artifact | SHA-256 | 结果 | 复核 |
|---|---|---|---|---|---|---|
| `FE-EVID-W01-016` | 2026-07-26 | rev 5 / `FE-W01-S10` | [`s10-runtime-provider-preflight.md`](s10-runtime-provider-preflight.md) | `b54b03a4028ba8c55cca380a9adc539d6bb65f014c2c0cc8bbe49cb96ce4611c` | `passed`；r007 在 runtime 前安全停止 | code + security + docs |
| `FE-EVID-W01-017` | 2026-07-26 | rev 6 / deterministic revalidation | [`s11-r6-deterministic-revalidation.md`](s11-r6-deterministic-revalidation.md) | `799424f44078a9af13f60932dd1d8882cced99d7aedac7b3cc5ea64060416719` | `failed`；技术 Gate 通过但 traceability 治理失败 | technical code/security approved；docs rejected |
| `FE-EVID-W01-019` | 2026-07-26 | rev 6 governance review / r009 `FE-W01-S12` input | [`s12-r6-traceability-preflight.md`](s12-r6-traceability-preflight.md) | `5032c063118f939bedc70415cd2b1dbb6c4161c0088595db62b587f5f40314f2` | `passed`；R6 在 credential/runtime 前失败关闭 | code + security + docs |
| `FE-EVID-W01-020` | 2026-07-26 | rev 7 / `FE-W01-S12` | 待 R7 确定性重验产生 |  | `planned` | unassigned |

### 追加进度事件

| 时间 | Step | 状态变化 | 事实与不可变锚点 | 下一动作 |
|---|---|---|---|---|
| 2026-07-26 | plan r008 review | `drafted → approved` | 最终激活内容 SHA `dd25cb6397aeef4db1442ef79fea4e0a36fd3dcc2a11f5db87919f4904993392`；继承 r007 Browser 合同并增加 inert-provider/零出站 Gate | 建立 R6，先独立迁移与重验 |
| 2026-07-26 | r008/R6 activation | `approved → active` | activation `047306b3f4113804c04a39725a0d5ee25bcb87b7`；freeze `90278f4a2b84d01566f3286bb097a2bd85945b05`；Evidence commit `d7617219f0419f3889ad0ac725435ce4e10642df` | 代码/安全/文档独立复核 |
| 2026-07-26 | R6 deterministic Gate | `active → technical-pass/governance-fail` | 11 个产品补丁未 staged/commit；全部确定性技术 Gate 通过；三个 R6 commit 缺 mandatory trailers，freeze 缺 Repository 与 policy hash | 不启动 Gateway/Vite/Chromium，不创建凭据；保留 R6 |
| 2026-07-26 | plan r009 review | `drafted → approved` | 候选评审 SHA `a8e8890237228cbd524acabb135c72cf76b58fcb2c76a797fd6415463776bf57`；激活内容 SHA `ef7b3f829fbfa915e8a312459e083a0e78c9414ff948b8ac478a9c714338caa8`；三路 Reviewer 均批准 | 从 reconstruction base 创建 R7 |
| 2026-07-26 | r009/R7 activation | `approved → active` | branch `repair/fe-w01-transport-r7`；parent/reconstruction base `5160273fb17502cf02cd10e1a17f5a47b7eb30be`；Repository `repo-goclaw-source-review`；Policy SHA `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf` | 形成带完整 trailers 的 docs-only activation commit；其 SHA 冻结为 Task Base |

### Authority pointer

- Current plan: [`plan-r009.md`](plan-r009.md)
- Current plan SHA-256:
  `ef7b3f829fbfa915e8a312459e083a0e78c9414ff948b8ac478a9c714338caa8`
- Current Registry: [`../../wave-registry.json`](../../wave-registry.json)
- Current Task: `FE-W01-TRANSPORT-R1` revision `7`
- Current recovery step: `FE-W01-S12`
- Current runtime state: no R7 token, config, database, Gateway, Vite or Chromium
- Blocking gate: `FE-W01-S08 / FE-EVID-W01-011` remains unresolved; no
  product commit or release is authorized.

### R7 Task freeze event

| 字段 | 冻结值 |
|---|---|
| 时间 / 状态 | `2026-07-26` / `created → frozen` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `repo-goclaw-source-review` |
| Repository authority | local Git review repository；branch `repair/fe-w01-transport-r7`；no remote |
| Task-ID / Revision | `FE-W01-TRANSPORT-R1` / `7` |
| Assignee | `Codex root agent` |
| Task Base | `8006415eb59952823434740ba2d855b6c66990f9` |
| Reconstruction base | `5160273fb17502cf02cd10e1a17f5a47b7eb30be` |
| Cumulative W01 diff base | `697f50e5f428769b75061dfd859d2549dd1c330d` |
| Plan | `plan-r009.md` / SHA-256 `ef7b3f829fbfa915e8a312459e083a0e78c9414ff948b8ac478a9c714338caa8` |
| Policy bundle | `wave-governance-v1` / repository-root `AGENTS.md` / SHA-256 `98bacd6013032cbaffd15095012ed6fc7cd274b62a78d3fdd738aeeadff94ebf` |
| Work-Items | `FE-W01-S01`、`S02`、`S03`、`S06`、`S07`、`S08`、`S09`、`S04`、`S12`、`S11`、`S10`、`S05` |
| Issues | `FE-ISSUE-002`、`003`、`004`、`005`、`006`、`007`、`008`、`009` |
| Acceptance criteria | R7 traceability/source-first/deterministic Gate；S11 inert/zero-outbound；S05 Desktop/Mobile login/connected/refresh/logout；S08 owner closure |
| Deterministic verification | reconstruction ancestry；mandatory trailers；Journal prefix；plan/policy SHA；legacy channel SHA-only source-first Delete/Add；11/11 manifest；Go/race/vet；UI transport/build；double-base scope；lockfile；download/config/syscall/sentinel/cleanup |
| Product commit | 禁止自动或人工创建；S08 与全部退出门禁通过后必须另行决定 |

冻结结果：activation commit `8006415eb59952823434740ba2d855b6c66990f9`
即 R7 Task Base；本 freeze 记录的 parent 必须精确等于该 SHA。任何冻结字段、
策略哈希或允许路径漂移都必须失败关闭并新建 plan revision。

### Authority pointer — frozen R7

- Current plan: [`plan-r009.md`](plan-r009.md)
- Current Task Base: `8006415eb59952823434740ba2d855b6c66990f9`
- Current Task revision: `7`
- Current step: `FE-W01-S12`
- Next admissible output: `FE-EVID-W01-020`
- Runtime remains absent until Evidence 020 passes independent code, security and
  documentation review.

### R7 deterministic Evidence event

| 时间 | Step | 状态变化 | Artifact / SHA-256 | 结果 | 下一动作 |
|---|---|---|---|---|---|
| 2026-07-26 | `FE-W01-S12` | `frozen → complete-deterministic` | [`s12-r7-traceable-deterministic-revalidation.md`](s12-r7-traceable-deterministic-revalidation.md) / `39b44db3eb5e124f7cb6a6174f7e5e34d3f26d3bee153ec0b9d0430626825513` | ancestry、完整 trailers、policy/plan/Journal hash、source-first、11/11 manifest、Go/race/vet、UI、双 base scope、lockfile 全绿；产品仍 unstaged/uncommitted | 以完整 trailers 提交 docs-only Evidence；独立代码、安全和文档复核 |

### Authority pointer — Evidence 020 collecting

- Current plan: [`plan-r009.md`](plan-r009.md)
- Current Task Base: `8006415eb59952823434740ba2d855b6c66990f9`
- Current Task revision / step: `7` / `FE-W01-S12`
- Current Evidence: `FE-EVID-W01-020` / `collecting`
- Current runtime state: no R7 token, sentinel, config, database, Gateway, Vite
  or Chromium
- S11 remains blocked until Evidence 020 passes independent code, security and
  documentation review.
- `FE-W01-S08 / FE-EVID-W01-011` remains unresolved; no product commit or
  release is authorized.

### Evidence projection correction

| 时间 | Reviewer | 状态 | Finding | 修正 | 执行边界 |
|---|---|---|---|---|---|
| 2026-07-26 | `wave_docs_validate` | `review → failed-projection` | Evidence Index 仍把 `FE-EVID-W01-015` 指向 r007/S10、把 `FE-EVID-W01-018` 指向 r008/R6，与 active r009/R7 不一致 | EVID015 当前产生位置改为 r009/R7 `S10/S05`；EVID018 改为 r009/R7 `S11`；Track 同步，历史记录不改写 | 仅 docs-only 投影修正；S11 继续阻断，重新独立复核后方可放行 |

### Authority pointer — projection corrected

- Current plan: [`plan-r009.md`](plan-r009.md)
- Current Task Base / revision: `8006415eb59952823434740ba2d855b6c66990f9`
  / `7`
- `FE-EVID-W01-020` remains `collecting`; security review approved, code and
  corrected documentation review remain required.
- Current S11 output authority: `FE-EVID-W01-018` in r009/R7.
- Current S10/S05 output authority: `FE-EVID-W01-015` in r009/R7.
- No R7 token, sentinel, config, database, Gateway, Vite or Chromium exists.
- `FE-W01-S08 / FE-EVID-W01-011` still blocks product commit and release.

### Evidence 020 package-count correction

| 时间 | Reviewer | Finding | 独立复算 | 旧 Artifact SHA-256 | 修正后 SHA-256 | 状态 |
|---|---|---|---|---|---|---|
| 2026-07-26 | `r4_code_review` | EVID020 将全仓绿色输出中的有测试 package 数写为 `16`，与 R7 当前清单不符；Gate 本身通过 | Go 1.25.5、`GOFLAGS=-buildvcs=false` 的 `go list` 过滤 `TestGoFiles/XTestGoFiles` 得到 `20` | `39b44db3eb5e124f7cb6a6174f7e5e34d3f26d3bee153ec0b9d0430626825513` | `9b5b760bf7605ac3e775b65866952e5d2ede373d578df34cbcb5696759487956` | `corrected-pending-review` |

该修正只改变 Evidence 的 package 计数，不改变测试命令、绿色结果、产品
文件或运行时状态。旧 SHA 保留为历史 commit `92ed62e...` 的精确工件；
不得 amend。修正以新的 trailers-complete docs-only commit 前向记录。

### Authority pointer — corrected Evidence 020

- Current Evidence: [`s12-r7-traceable-deterministic-revalidation.md`](s12-r7-traceable-deterministic-revalidation.md)
- Current Evidence SHA-256:
  `9b5b760bf7605ac3e775b65866952e5d2ede373d578df34cbcb5696759487956`
- Current state: `collecting`; final code, security and documentation re-review
  required before S11.
- Product patch remains unstaged/uncommitted; runtime remains absent.

### Evidence 020 independent acceptance

| 时间 | Evidence / SHA-256 | Code | Security | Documentation | 结论 |
|---|---|---|---|---|---|
| 2026-07-26 | `FE-EVID-W01-020` / `9b5b760bf7605ac3e775b65866952e5d2ede373d578df34cbcb5696759487956` | `r4_code_review` APPROVE；无 S0–S3 finding；独立复跑 Go/race/vet/UI/manifest/scope | `r4_security_review` APPROVE；source/runtime/artifact 边界通过 | `wave_docs_validate` PASS；append-only、projection、trailers、links/JSON/YAML 通过 | `passed`；S12 complete，允许进入 r009/R7 S11 |

批准只适用于最终 Evidence SHA 和未变化的 11 个产品文件哈希。它不授权
产品 commit、W01 complete 或发布；`FE-W01-S08 / FE-EVID-W01-011` 继续
保持硬阻断。

### Authority pointer — S11 ready

- Current plan / Task revision: [`plan-r009.md`](plan-r009.md) / `7`
- Current Evidence: `FE-EVID-W01-020` / `passed`
- Current step: `FE-W01-S11`
- Required output: `FE-EVID-W01-018`
- Authorized action: Playwright download-artifact integrity preflight,
  inert-provider config preflight and Gateway/child syscall zero-outbound Gate
  only.
- Browser interaction `S10/S05` remains blocked until S11 passes.
- Product patch remains unstaged/uncommitted; S08 still blocks commit/release.

### README current-step projection correction

| 时间 | Reviewer | Finding | 修正 | 状态 |
|---|---|---|---|---|
| 2026-07-26 | `wave_docs_validate` | EVID020 已 passed 且 Track/Journal 已进入 S11，但 Wave README 当前 metadata 示例仍指向 S12/Issue009，可能使下一 Evidence commit 使用错误 trailers | README 当前 pointer 改为 `FE-W01-S11`，相关 Issue 逐行为 `FE-ISSUE-003/004/005`；历史 commit 不改写 | `corrected-pending-review` |

### Authority pointer — README aligned

- Current plan / Task revision: [`plan-r009.md`](plan-r009.md) / `7`
- Current Step / Evidence output: `FE-W01-S11` / `FE-EVID-W01-018`
- Current related Issues: `FE-ISSUE-003`、`FE-ISSUE-004`、`FE-ISSUE-005`
- EVID020 remains passed at SHA
  `9b5b760bf7605ac3e775b65866952e5d2ede373d578df34cbcb5696759487956`.
- No Playwright artifact cleanup, credential/config creation or runtime startup
  occurred before this projection correction.

### S11 syscall Gate environment block

| 时间 | Step / Issue | 状态变化 | Artifact / SHA-256 | 实际结果 | 安全停止 |
|---|---|---|---|---|---|
| 2026-07-26 | `FE-W01-S11` / `FE-ISSUE-003/004/005/010` | `ready → environment-blocked` | [`s11-inert-provider-environment-blocked.md`](s11-inert-provider-environment-blocked.md) / `4c581d32b816035cc2dac434f2f2aca7dacdd05a8aaf603ec626c8a7c0d9bb39` | 下载工件版本、canonical manifest、owner/mode/link、offline dependency tree 与 loopback 端口通过；sandbox 和授权宿主级 `/bin/true` strace 均被 ptrace 拒绝 | npm cache/debug log 已删除；strace 临时日志已删除；0 credential/config/database/profile/runtime；0 Gateway/Vite/Chromium |

`FE-DEC-014` 冻结为不以 socket 轮询或其他未批准机制替代 syscall Gate。保持
r009 合同时，恢复条件是先在受控本地环境通过相同 `/bin/true`
`strace -f -e trace=connect` 能力测试；若改变证明机制，必须先批准新的
plan revision。

### Authority pointer — S11 environment-blocked

- Current plan / Task revision: [`plan-r009.md`](plan-r009.md) / `7`
- Current Step / Evidence: `FE-W01-S11` / `FE-EVID-W01-018 failed`
- Current blocker: `FE-ISSUE-010` / ptrace capability unavailable
- S10/S05 browser execution did not start and remains blocked.
- Product patch remains unstaged/uncommitted.
- `FE-W01-S08 / FE-EVID-W01-011` also remains unresolved and independently
  blocks product commit/W01 complete/release.

### Decision 014 table correction

| 时间 | Reviewer | Finding | 修正 | 状态 |
|---|---|---|---|---|
| 2026-07-26 | `wave_docs_validate` | `FE-DEC-014` 比七列表头多一列，导致决策、原因、影响和 Supersedes 投影错位 | 将“触发条件 + 不替代/失败关闭/等待同一 strace 能力”合并到决策列；原因、影响和 Supersedes 恢复到各自列 | `corrected-pending-review` |

该前向修正不改变 `FE-EVID-W01-018 failed/environment-blocked` 的事实，不
放行 S10/S05，也不创建任何 credential 或 runtime。

### Authority pointer — environment block retained

- Current decision: `FE-DEC-014` / no syscall-Gate substitution.
- Current blocker: `FE-ISSUE-010` / ptrace capability unavailable.
- Current Evidence: `FE-EVID-W01-018 failed`.
- Product patch remains unstaged/uncommitted; S08 and release blocks remain.

### Evidence 018 tooling-version correction

| 时间 | Reviewer | Finding | 独立复算 | 旧 SHA-256 | 修正后 SHA-256 | 状态 |
|---|---|---|---|---|---|---|
| 2026-07-26 | `r4_code_review` | EVID018 未记录 plan-r008 要求的 canonical manifest 工具版本 | GNU find/xargs `4.9.0`；GNU sort/sha256sum `9.4`；npm `11.9.0`；Node `v24.14.0` | `4c581d32b816035cc2dac434f2f2aca7dacdd05a8aaf603ec626c8a7c0d9bb39` | `e8873f0332f5b8542170e41fe3c3fbd7b4ed6e402d5ad71d7d9e96d087019c20` | `corrected-pending-review` |

本次仅补齐工具版本，不改变下载 manifest、ptrace 失败、安全停止或
`environment-blocked` 结论。旧 SHA 保留为 commit `3b1ff9c...` 的历史
Evidence，不 amend。

### Authority pointer — corrected failed Evidence 018

- Current Evidence:
  [`s11-inert-provider-environment-blocked.md`](s11-inert-provider-environment-blocked.md)
- Current Evidence SHA-256:
  `e8873f0332f5b8542170e41fe3c3fbd7b4ed6e402d5ad71d7d9e96d087019c20`
- Current state: `failed/environment-blocked`; S10/S05 remains blocked.
- Product patch remains unstaged/uncommitted; no runtime exists.

### S11 user-triggered capability retry

| 时间 | 用户动作 | 合同 | 能力探测 | 实际结果 | 安全停止 |
|---|---|---|---|---|---|
| 2026-07-26 | 明确要求“启动真实浏览器测试” | 继续使用 `plan-r009` / `FE-DEC-014`，不替代 syscall Gate | 授权宿主级 `strace -f -e trace=connect` 包裹 `/bin/true` | `PTRACE_TRACEME` 与 `PTRACE_SETOPTIONS` 仍被拒；进程定向终止，exit `130`；精确日志 `0` 行并已删除 | 0 credential/config/database/profile/runtime；0 Gateway/Vite/Chromium/Playwright；未进入 artifact read-only 运行阶段 |

本次是同一失败关闭合同的恢复能力重试，不是新的 Browser 用例执行，也不
改变产品范围。`FE-ISSUE-010` 继续为 `root-caused` 环境阻断；只有相同
`/bin/true` 能力测试先通过，才允许继续 S11。

### Authority pointer — S11 retry remains environment-blocked

- Current plan / Task revision: [`plan-r009.md`](plan-r009.md) / `7`
- Current Step / Evidence: `FE-W01-S11` / `FE-EVID-W01-018 failed`
- Current Evidence SHA-256:
  `56cf8491f7751d7c4b49fdd79843d97e30e3a06e95add7957e028fa3ce1c0e36`
- Current blocker: `FE-ISSUE-010` / ptrace capability unavailable after
  user-triggered authorized host retry.
- S10/S05 browser runtime remains blocked and was not started.
- Product patch remains unstaged/uncommitted; `FE-W01-S08` still independently
  blocks product commit, W01 completion and release.

### S11 retry independent review

| 时间 | Evidence / SHA-256 | Code | Security | Documentation | 结论 |
|---|---|---|---|---|---|
| 2026-07-26 | `FE-EVID-W01-018` / `56cf8491f7751d7c4b49fdd79843d97e30e3a06e95add7957e028fa3ce1c0e36` | `r4_code_review` APPROVE；5/5 docs delta、Journal append-only、11/11 产品 SHA 与零 staged product 通过 | `r4_security_review` APPROVE；r009/DEC014 未降级、零 runtime、失败关闭通过 | `wave_docs_validate` PASS；Evidence SHA、Issue/Track/Index、24 表、94 links、JSON/YAML 通过 | 接受本次失败 Evidence；状态仍为 `failed/environment-blocked`，不放行 S10/S05 |

复核接受的是“阻断被准确记录且安全停止”，不是 S11 绿色通过。只有
`FE-ISSUE-010` 的 ptrace capability 恢复后，才允许重新收集绿色
EVID018；S08、产品提交、W01 complete 与 release 的阻断均保持不变。
## 2026-07-28 — r010 暂停并等待 recovered base

- `MVP-W00` 是唯一 active Wave；
- 新建 `plan-r010.md`，把 FE-W01 机器状态、范围和 product flag 收敛为
  `blocked`、`docs/waves/**`、`false`；
- r010 增加 `MVP-W00` 依赖，不改写 r009 历史；
- 恢复完成后必须用新 revision 重新绑定 Task base 与未完成 Evidence。
