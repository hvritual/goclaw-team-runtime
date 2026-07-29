# 前台问题登记

本登记表只记录可观察事实和证据状态。用户报告不自动等于已复现缺陷，
页面返回空数据也不能自动解释为后端无数据或权限错误。

## 状态定义

| 状态 | 含义 |
|---|---|
| `reported` | 用户或操作者报告，尚未在受控环境复现 |
| `unverified` | 已进入验证队列，但证据不足 |
| `reproduced` | 有稳定环境、步骤、期望、实际结果和原始证据 |
| `root-caused` | 根因由代码、契约或运行时证据支持 |
| `fixing` | 已绑定 active Wave、冻结 Task 和允许范围 |
| `fixed` | 实施完成，等待独立回归 |
| `verified` | 原复现失败、回归通过且无越权副作用 |
| `deferred` | 有明确原因、风险和目标 Wave |
| `not-a-bug` | 证据证明符合冻结契约，并记录文档/体验改进需求 |

只有 `reproduced` 或 `root-caused` 的记录可以进入修复 Task。

## 用户报告

| Issue ID | 来源 | 表面症状 | 状态 | 当前证据 | 目标 Wave |
|---|---|---|---|---|---|
| `FE-ISSUE-001` | 用户，2026-07-26 | Team Web Console 多项功能使用异常 | `unverified` | 已拆出 `FE-ISSUE-002`–`010`；其余页面仍不得推断为正常或归因 | `FE-W00`–`FE-W05` |

`FE-ISSUE-001` 是收件箱条目，不可作为包含多个修复的永久聚合 Issue。
各责任 Wave 每复现一个独立症状，就拆出一个新的稳定 Issue ID。

## 具体问题

以下记录按发现顺序追加。协议探针不是浏览器视觉验收；其证据范围已在
[`FE-EVID-W00-007`](frontend-stability/fe-w00/authority-runtime-reproduction.md)
中限定。

| Issue ID | 页面/流程 | 环境 | 最小复现 | 期望 | 实际 | 严重度 | 状态 | 根因证据 | Wave | Task |
|---|---|---|---|---|---|---|---|---|---|---|
| `FE-ISSUE-002` | Gateway session 测试基线 | Go 1.25.5；base `b288564` | 编译 Gateway 的 Web Session/Origin 目标测试 | 测试包可编译并运行 | `team_runtime_test.go:166,171` 仍按 2 返回值调用 3 返回值函数，build failed | `S2` | `fixed` | `FE-EVID-W00-007` 的编译输出与调用/实现签名逐行对应；R4 Gateway/race 全绿 | `FE-W01` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-003` | Vite 开发态登录 | Node 24.14；Vite 5.4.21；一次性 target | 从页面 Origin 经实际 `/auth` proxy POST `/auth/session` | proxy 保留页面 Host，严格同源检查通过 | `changeOrigin:true` 把 Host 改为 target，Origin 不变，探针返回 403 | `S2` | `fixed` | `FE-EVID-W00-007` 的真实 proxy Host/Origin/status；R4 transport 2/2 | `FE-W01` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-004` | Vite 开发态 WebSocket | Node 24.14；Vite 5.4.21；一次性 upgrade target | TeamClient 建连，并对照直连与 `/ws` proxy | client 使用页面 host 的 `/ws`，proxy 完成 101 | client 捕获 `:28789`；直连 403，同源 proxy 101 | `S2` | `fixed` | `FE-EVID-W00-007` 的真实 upgrade 对照与 SSR URL 捕获；R4 transport 2/2 | `FE-W01` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-005` | Team Web Console shell | Go 1.25.5；base `b288564`；内存 HTTP recorder | `GET /dashboard/`，重复两次并检查 `Location` | 200、no-store、安全头、无重定向 | 301 `Location: ./`；相对位置解析回同一 `/dashboard/`，形成自循环 | `S1` | `fixed` | [`FE-EVID-W01-007`](frontend-stability/fe-w01/s04-dashboard-shell-reproduction.md)；R4 shell/security/cache 与 Gateway race 全绿 | `FE-W01` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-006` | 全仓 Go Gate / WeWork channel test | Go 1.25.5；受控网络；base `697f50e` | `go test ./channels -count=1 -v -timeout=30s` | constructor 单测快速、离线结束 | `TestNewWeWorkWsBotChannel` 无条件 sleep 10m，30s timeout | `S2` | `fixed` | [`FE-EVID-W01-009`](frontend-stability/fe-w01/s04-full-go-channels-reproduction.md)；R4 channels/race 与全仓全绿 | `FE-W01` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-007` | WeWork channel test credential hygiene | 版本化 0.7.0 source；有效性未知 | 脱敏检查 `weworkwsbot_test.go` config literals | 测试只含明显 synthetic placeholder；owner 证明旧值已撤销/轮换或从未有效 | 源码直接包含 35/43 字符 credential-shaped Bot/Secret literals；未证明有效或已被使用 | `S0` | `fixing` | [`FE-EVID-W01-009`](frontend-stability/fe-w01/s04-full-go-channels-reproduction.md)；[`FE-EVID-W01-021`](frontend-stability/fe-w01/s13-r012-mvp-environment-blocked.md)；内容不复制，外部有效性未验证 | `FE-W01-S13D` owner closure；external owner blocker；FE-W05 revalidation | `FE-W01-MVP-BROWSER-012`；credential owner action pending |
| `FE-ISSUE-008` | 全仓 Go Gate / Memory Catalog provenance | Go 1.25.5；R3；累计 base `697f50e` | `go test ./... -count=1 -timeout=5m`，再独立重复目标测试两次 | 默认 Markdown ingestion 的 provenance kind 为 `markdown` | 空 scheme 先默认成 `markdown`，随后又统一追加 `-markdown`，得到 `markdown-markdown` | `S2` | `fixed` | [`FE-EVID-W01-012`](frontend-stability/fe-w01/s04-memory-catalog-reproduction.md)；[`FE-EVID-W01-013`](frontend-stability/fe-w01/s09-catalog-provenance-red-green.md) 红绿/包级/race/全仓全绿 | `FE-W01-S09` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-009` | Wave Task/commit traceability | R6；Plan r008；runtime 前 | 检查 freeze tuple 与 activation/freeze/Evidence commit trailers | Task 冻结 Repository 与 policy hash；每个 commit 含 AGENTS mandatory trailers | freeze 缺 Repository/hash；`047306b/90278f4/d761721` 只有 subject | `S1` | `verified` | [`FE-EVID-W01-019`](frontend-stability/fe-w01/s12-r6-traceability-preflight.md)；[`FE-EVID-W01-020`](frontend-stability/fe-w01/s12-r7-traceable-deterministic-revalidation.md)；代码、安全与文档独立复核通过，产品仍未提交 | `FE-W01-S12` | `FE-W01-TRANSPORT-R1` rev 7 |
| `FE-ISSUE-010` | S11 Gateway syscall 零出站 Gate | Linux 6.12.13；strace 6.8；credential/runtime 前 | 以 `strace -f -e trace=connect` 包裹无网络 `/bin/true`；sandbox、首次授权宿主、用户再次触发的授权宿主及 recovered-base r012 预检 | 能归属进程树并生成可用 syscall trace | 四次均在 trace 前被 ptrace 权限拒绝；能力检查被定向终止并清理 | `S1` | `root-caused` | [`FE-EVID-W01-018`](frontend-stability/fe-w01/s11-inert-provider-environment-blocked.md)；[`FE-EVID-W01-021`](frontend-stability/fe-w01/s13-r012-mvp-environment-blocked.md)；未用 socket 轮询替代 | `FE-W01-S13B/S13C` | `FE-W01-MVP-BROWSER-012`；等待 ptrace-capable environment |

## Team Runtime 问题

| Issue ID | 流程 | 环境/基线 | 期望 | 实际 | 严重度 | 状态 | 证据 | Wave | Task |
|---|---|---|---|---|---|---|---|---|---|
| `TR-ISSUE-001` | Team Control/Runner 应用部署 | recovered `0.8.0-pilot.1`；单一 `goclaw` 入口 | 控制面和工作站是独立命令面、构建物和升级单元 | 现有代码虽有完整服务/CLI 模块，但同一入口同时暴露两类职责，发行包也只有单一 runtime | `S1` | `verified` | [`TR-EVID-W00-001`](team-runtime/tr-w00/s05-application-boundary-verification.md) + [`TR-EVID-W00-003`](team-runtime/tr-w00/s08-r002-final-verification.md) | `TR-W00-S01`–`S08` | `TR-W00-APP-SPLIT-001` r001；`TR-W00-ACCEPTANCE-002` r002 |
| `TR-ISSUE-002` | TR-W00 独立验收 | exact head `9d6a252...`；PR #1 已提前合并 | source/cross-build/credential/OAuth read boundary 与 current docs 全部满足冻结 Gate | code 2、security 2、docs 4 个 P1；P0=0；PR 在复核前合并 | `S1` | `verified` | [`TR-EVID-W00-002`](team-runtime/tr-w00/s05-independent-review-r001.md) 保留失败；[`TR-EVID-W00-003`](team-runtime/tr-w00/s08-r002-final-verification.md) exact `60465b59...` 三路 P0=0/P1=0 | `TR-W00-S06`–`S08` | `TR-W00-ACCEPTANCE-002` r002 |
| `TC-ISSUE-001` | 中央全局治理 | 现有 TeamControl 文件存储与 RPC | 管理预算、知识源、Skill、Runner release 和 Context Bundle | r004 review 继续复现 decoded control、device whitespace 与跨平台 relative path；历史 RBAC provenance 已前向恢复 | `S1` | `fixing` | [`TC-EVID-W01-002`](team-runtime/tc-w01/s05-independent-review-r002.md)、[`004`](team-runtime/tc-w01/s08-r003-independent-review-1.md)、[`006`](team-runtime/tc-w01/s08-r003-independent-review-2.md)、[`008`](team-runtime/tc-w01/s08-r003-independent-review-3.md)、[`010`](team-runtime/tc-w01/s09-r004-independent-review-4.md)、[`012`](team-runtime/tc-w01/s11-r004-independent-review-5.md)、[`014`](team-runtime/tc-w01/s11-r004-independent-review-6.md) failed；[`015`](team-runtime/tc-w01/s11-r004-review-remediation-7.md) collecting；[`r004 Freeze`](team-runtime/tc-w01/task-freeze-r004.md) | `TC-W01-S09`–`S11` | `TC-W01-ACCEPTANCE-006` r004 |
| `RN-ISSUE-001` | Runner 生命周期 | 现有 register/doctor/work/update/key rotation | 兼容性协商、校验下载、原子升级与回滚 | 本地执行闭环已实现，版本管理仍依赖人工替换同版本 binary | `S1` | `planned` | `RN-W01` 激活后复现 | `RN-W01` | 未冻结 |
| `INT-ISSUE-001` | Runner 项目上下文 | 现有 Memory Catalog/Harness knowledge/Codex runner | Runner 以 project-scoped MCP 读取批准知识与 Skill，并验证 Context Bundle | 知识接口存在于 Agent/Gateway，但 Runner/Codex 执行包没有统一 MCP/Context 合同 | `S1` | `planned` | `INT-W01` 激活后复现 | `INT-W01` | 未冻结 |

## Recovery 问题

| Issue ID | 流程 | 环境/基线 | 期望 | 实际 | 严重度 | 状态 | 证据 | Wave | Task |
|---|---|---|---|---|---|---|---|---|---|
| `MVP-ISSUE-001` | Recovery 治理与最终发布 | recovered Git；r003 final review | current revision/权限唯一；批准者真实；Task 可从 exact base 解析完整 tuple/policy | r002 的 projection/批准/tuple 缺口在 r003 修复，但 r003 Task base 尚不含 active r003 Plan/Registry/Policy，形成 self-authorizing freeze | `S1` | `verified` | r005 activation `df8fe9f` 与 freeze `96de00a` 顺序分离；r005 code/docs 独立复核通过 | `MVP-W00-S08A`–`S08E` | `MVP-W00-RECOVERY-005` r005 |
| `MVP-ISSUE-002` | Journal 历史完整性 | recovered Git；r003 final review | 已冻结历史字节不变，Recovery 状态只在 EOF 追加 | FE-W01 前 26641 bytes SHA 从 `33a50e...` 变为 `b98013...`；FE-W00/PILOT/MVP 也存在顶部改写或中间插入 | `S1` | `verified` | r005 S08C 五个冻结前缀复算通过；r005 docs 独立复核通过 | `MVP-W00-S08C`–`S08E` | `MVP-W00-RECOVERY-005` r005 |
| `MVP-ISSUE-003` | Recovery 冻结验收常量 | recovered Git；r004 S07C 实算 | Plan/Task 的 FE-W01 26641-byte SHA 与 r009 Plan、R7 Evidence、import tree 一致 | r004 误抄为 `33a50e8f3a...`；三份权威来源与实算均为 `33a50e1bbd...` | `S1` | `verified` | r005 Plan/Task 引用完整权威 SHA；四来源与三路独立复核一致 | `MVP-W00-S08A`–`S08E` | `MVP-W00-RECOVERY-005` r005 |

## 三人试点问题

下列条目来自 2026-07-27 三路只读源码审计。根因均能由当前实现路径直接
定位，因此进入 `PILOT-W00`；验收证据仍需在修复后独立收集。

| Issue ID | 影响面 | 表面症状 | 严重度 | 状态 | 根因摘要 | Wave |
|---|---|---|---|---|---|---|
| `PILOT-ISSUE-001` | Runner 平台 | 只有 Linux bwrap/linux-amd64 包，却可能被理解为原生三平台执行 | `S1` | `fixed` | 无平台 preflight、双架构包、WSL/Lima 基线和非 Linux fail-close | `PILOT-W00-S02` |
| `PILOT-ISSUE-002` | Runner 隔离 | 取消、TMP、Git hook/filter 可逃出验证边界 | `S0` | `fixed` | CommandContext 只杀直接进程；宿主 TMP/Git config 未隔离 | `PILOT-W00-S02` |
| `PILOT-ISSUE-003` | Wave 治理 | freeze/enqueue 不验证 active Wave、plan/registry hash、Step/scope | `S0` | `fixed` | Wave 目前只存在于文档，Task/ExecutionPack/trailers 无 binding | `PILOT-W00-S03` |
| `PILOT-ISSUE-004` | 一致性 | 跨三存储崩溃可留下 partial state | `S1` | `fixed` | 跨存储步骤无事务、reconciler 或统一 consistency Gate | `PILOT-W00-S04` |
| `PILOT-ISSUE-005` | 恢复 | 无覆盖三根、Evidence/credential 与 Git base 的一致快照 | `S1` | `fixed` | 缺 maintenance lock、manifest/hash、restore-to-new-root 验证 | `PILOT-W00-S04` |
| `PILOT-ISSUE-006` | Web scope/session/chat | 切项目显示旧数据、401 不退出、刷新丢聊天 | `S0` | `fixed` | loader 保留旧 data；client/provider 双 session；无 team-safe history | `PILOT-W00-S05` |
| `PILOT-ISSUE-007` | 三人协作 | Team 页只读，其他成员变化不会及时出现 | `S2` | `fixed` | 无 mutation UI，非 Chat 页面无重连/聚焦/定时刷新 | `PILOT-W00-S05` |
| `PILOT-ISSUE-008` | DoneGate | linked WorkItem/Issue 可由通用 transition 提前进入终态 | `S0` | `fixed` | Gateway mutation 未检查关联 DevTask/共享 Issue 完成状态 | `PILOT-W00-S03` |
| `PILOT-ISSUE-009` | 职责分离 | 严格治理在恰好三名真人时不可满足 | `S1` | `fixed` | creator 禁 self-review + 两 reviewer + 独立 final approver 需四身份 | `PILOT-W00-S03` |
| `PILOT-ISSUE-010` | 治理旁路 | manager raw enqueue/Ouroboros direct compile 绕过统一冻结链 | `S0` | `fixed` | TeamGuard 允许两个 direct mutation path | `PILOT-W00-S03` |
| `PILOT-ISSUE-011` | 外部凭据 | 历史 credential-shaped material 缺 owner closure | `S0` | `deferred` | 只能由凭据责任人撤销/轮换或证明从未有效 | external owner |
| `PILOT-ISSUE-012` | Browser 安全证据 | 当前环境无法收集 syscall 零出站 trace | `S1` | `deferred` | ptrace 被宿主安全策略拒绝 | ptrace-capable environment |
| `PILOT-ISSUE-013` | 项目聊天隔离 | 旧会话键的 `:`→`_` 文件名规范化会让不同 project/topic 组合碰撞 | `S0` | `fixed` | 会话键分段边界在文件名层丢失；读写两端没有统一的无歧义编码 | `PILOT-W00-S05` |

上述 `fixed` 只表示代码和确定性回归已完成，尚未等同于 `verified`。真实
WSL2/Lima/bwrap、三台电脑 Codex OAuth、飞书、Obsidian Desktop 和浏览器
现场 Gate 仍须由独立试点操作者完成；状态与阻断记录在
[`PILOT-W00 Journal`](pilot-readiness/pilot-w00/journal.md) 和
[`evidence-index.md`](evidence-index.md)。

严重度只描述影响，不决定修复顺序：

- `S0`：凭据泄露、跨项目越权或不可逆数据损坏；
- `S1`：核心流程无法继续，且无安全替代路径；
- `S2`：主要功能异常，但存在受控替代路径；
- `S3`：局部体验、展示或低频边界异常。

## 复现证据最低要求

每个进入 `reproduced` 的 Issue 必须同时具备：

1. Runtime 版本与构建哈希；
2. 浏览器、视口、访问路径和部署拓扑；
3. principal、项目角色和项目 ID 的脱敏描述；
4. 前置数据或夹具；
5. 可重复操作步骤；
6. 期望结果的权威依据；
7. 实际页面状态；
8. 相关 HTTP/RPC 请求、状态码和脱敏响应；
9. Console、Gateway 或 Trace 证据；
10. 截图或视频，仅作为辅助证据；
11. 是否可以在第二次全新会话中复现。

Token、Cookie、Reviewer Token、设备密钥、Codex OAuth 和私人数据不得进入
本文件、截图、网络导出或测试夹具。

## 关闭规则

- 修复者不能单独把自己的 Issue 标为 `verified`。
- “无法复现”不是 `not-a-bug`，应保持 `unverified` 并记录环境差异。
- 一个 Issue 跨 Wave 时保留 ID，并在每个 Wave 的进度日志中引用。
- 聚合报告只有在全部拆分 Issue 完成分流后才可关闭。
