# PILOT-W00 执行 Journal

本文件只追加。原始日志、二进制和敏感值不进入仓库。

## 2026-07-27 — S01 激活

- 用户授权：补充 Runner 跨平台要求并推进到 3 人试点；
- 源码基线：`goclaw-r7-local-handoff-20260726` 的 `source/`；
- 隔离目录：独立于旧 0.6 脏工作树；handoff 不含 Git 历史；
- 三路审计：`runner_xplat_audit`、`pilot_control_audit`、
  `pilot_frontend_audit` 完成只读复核；
- 平台决策：Linux native、Windows/WSL2、macOS/Lima；
- 治理决策：`planner-service` 作为不可登录创建者，三名真人保持严格分离；
- 旧 `FE-W01`：转为 `blocked`，外部 owner 和 ptrace 事实不改写；
- 当前环境：Node `v24.14.0`、npm `11.9.0`；Go 尚未安装；
- syscall capability：`strace -f -e trace=connect /bin/true` 因
  `PTRACE_TRACEME/PTRACE_SETOPTIONS: Operation not permitted` 失败并终止，
  未创建 credential 或启动 runtime。

当前授权 Step：`PILOT-W00-S02`、`S03`、`S04`、`S05` 可在冻结 allowlist
内并行实施；进入 S06 前必须完成对应确定性测试。

## 2026-07-27 — r002 会话隔离修订

- S05 反例确认旧会话文件名规范化存在跨项目/Topic 碰撞；
- 创建 `plan-r002.md`，新增 `PILOT-ISSUE-013` 和 `PILOT-DEC-005`；
- allowlist 只扩展 `agent/manager.go`，用于让写入端与 history 读取端复用
  `session` 的版本化无碰撞键；
- 旧键仅允许可证明无歧义的迁移；模糊映射必须失败关闭。

## 2026-07-27 — r003 测试范围修订

- r002 漏列既有 `agent/manager_project_session_test.go`；
- 在修改该测试前创建 r003，只把这一文件加入 allowlist；
- 不改变产品范围或验收标准，精确断言必须随权威 key builder 更新。

## 2026-07-27 — S02–S05 实现收敛

- Runner 执行合同统一为 Linux substrate：native Linux、WSL2 Linux guest、
  Lima Linux guest；原生 Windows/macOS 的 `runner work` 失败关闭；
- `runner doctor` 增加宿主画像、guest-local 路径、架构、Codex
  `login status`、归属链、Git 执行配置、device key、root-owned bwrap
  wrapper 与实际启动探针；
- Runner 注册自动带 `goclaw-runtime-linux-v1`、GOOS/GOARCH、host profile、
  bwrap backend 和 wrapper SHA-256；三人准入要求三台摘要一致；
- Codex、内部 Git 与 verifier 使用固定安全 PATH、隔离 HOME/XDG/TMP，
  Git hook/filter/fsmonitor/include/credential helper、外部 diff/merge 和
  可变 `.gitattributes` 失败关闭；取消会终止整个 Unix 进程组；
- Team `dev create` 只接收 `wave_step` 意图。Gateway 从注册仓库的精确
  base commit 解析唯一 active Registry/plan/dependency/scope/hash，
  覆盖客户端自报 Wave 权威字段并冻结 `planner-service` 创建者；
- freeze、enqueue、accept 重验 Wave；Team raw enqueue 和 Ouroboros
  direct compile 旁路被拒绝；terminal Issue/WorkItem guard 生效；
- 恰好三名真人的严格职责分离落为 Alice scenario/capacity、Bob risk/cost、
  Carol independent final，并继续禁止每人越过 review-kind 上限；
- 新增 `control.consistency.check`，critical 跨存储 finding 阻止 enqueue
  和 accept；
- 新增维护锁、age 加密冷备、语义解密验证、Git bundle、事件链、文件
  digest/mode/type 校验和仅恢复到新目录的 `pilot backup/verify-backup/restore`；
- 项目会话使用版本化逐段 base64url key；模糊 legacy 映射失败关闭；
- Web Console 增加项目选择、项目级聊天持久化、401 退出、刷新/重连、
  Bug/WorkItem/Assignment 状态操作，并拒绝缺失或冲突的 WebSocket
  `project_id/topic_id`。

## 2026-07-27 — 确定性验证与发布候选

- Go `1.25.5` 选定包发布 Gate 全部通过：
  `memory`、`memory/catalog`、`governance`、`ouroboros`、
  `orchestratorlite`、`harness`、`teamcontrol`、`workstation`、
  `providers`、`gateway`、`agent`、`agent/tools`、`config`、`cli`、
  `cli/commands`、`internal/start`；
- `go test -race -count=1` 对 `session`、`orchestratorlite`、
  `teamcontrol`、`workstation`、`gateway`、`cli` 全部通过；
- 同一组关键包 `go vet` 通过，发布与 sandbox/安装脚本 `bash -n` 通过；
- Web Console 状态与传输测试 `8/8` 通过，TypeScript/Vite 生产构建通过；
- Obsidian adapter 测试 `6/6` 通过，TypeScript/esbuild 构建通过；
- Linux amd64/arm64 Runner 构建与可恢复归档通过；Darwin
  amd64/arm64、Windows amd64/arm64 控制 CLI 交叉编译通过但不作为
  Runner 包发布；
- 源码 archive allowlist、危险成员/链接/常见二进制/credential-like
  material 二次恢复扫描通过；四个发布归档的 `sha256sum -c` 全部通过；
- active plan SHA-256 为
  `976ece8d7d1814b356e86aadf0dea3ce6c343a8fd35a6b1ca21b16bfffccf4d1`，
  Registry SHA-256 为
  `0658275a1b9d63df4b2da0ef8b7b911ad007d88d660c73e457e21d2296b8d86d`。

## 2026-07-27 — 三身份夹具与现场阻断

- 一次性本地夹具成功建立 Alice、Bob、Carol 三个 principal、三份独立
  `0600` personal token、三个项目成员和三个不同 owner 的 Runner；
- 三个 Runner 分别投影为 `native-linux/amd64`、`wsl2/amd64`、
  `lima/arm64`，共同使用同一冻结 wrapper SHA-256，并成功创建有关联的
  Bug、WorkItem 和 Bob owner assignment；
- 该夹具只验证 Gateway/RBAC/持久投影，不包含真实 Codex OAuth、飞书
  credential、Obsidian Vault 或三台物理电脑；
- 云端 Browser 在访问 `http://127.0.0.1:28891/dashboard/` 前被
  localhost 安全策略拒绝。未绕过策略，因此三独立 BrowserContext、
  Desktop/Mobile 与页面交互证据保持 `failed/pending现场复验`；
- 当前容器的真实 bwrap 探针失败关闭：
  `bwrap: open /proc/10/ns/ns failed: No such file or directory`；没有把
  单元测试误写为真机 bwrap、WSL2 或 Lima smoke；
- syscall 零出站 trace 仍因 ptrace policy 无法采集；没有以 socket
  轮询或应用日志替代 syscall 证据；
- `PILOT-W00` 保持 `active`。发布物是 `0.8.0-pilot.1` 技术候选，
  只有 credential owner、三台目标电脑、真实 Codex/飞书和现场浏览器
  Gate 完成后才能升级为“三人试点已上线”。

## 2026-07-27 — 跨平台构建与多项目运维手册

- 新增 `docs/CROSS_PLATFORM_BUILD_TEAM_OPERATIONS_CN.md`，从当前源码和
  实际 CLI 固化 Windows/macOS 控制端、Linux/WSL2/Lima 执行端的构建边界；
- 明确 Gateway Token、个人 Team Token、Runner device key 与 Codex OAuth
  四类凭据的保存、轮换、撤销及离职处理顺序；
- 明确一个 Runner ID 只有一个活动 lease；同机多项目并发必须使用独立
  Runner ID、key、work root 和进程；
- 增加项目 RBAC、Repository 映射、Issue/WorkItem/Assignment、四审、
  freeze、enqueue、Runner、Evidence、accept 和 link-pr 的完整操作说明；
- 本次只更新文档和 README 入口，不改变 Runtime 行为，也不把多项目常规
  模式误写为已完成三人现场 Pilot Gate。

## 2026-07-28 — r004 暂停并等待恢复基线

- 用户批准先执行源码恢复阶段以及 MVP；
- 已确认 `0.8.0-pilot.1` 来源归档完整，但不含原始 `.git`；
- 创建 `plan-r004.md`，把本 Wave 从 `active` 暂停为 `blocked`；
- `MVP-W00` 成为唯一 active Wave，只允许恢复文档、工具链清单和验证配置；
- 只有恢复 Gate 全部通过并生成新 base commit 后，才允许创建后继 Pilot
  revision 并重新绑定实机 Evidence。

## 2026-07-28 — r005 收敛机器依赖与范围

- 独立来源和文档复核发现 r004 正文要求等待 `MVP-W00`，但 frontmatter 与
  Registry 未机器化该依赖；
- Registry 仍残留 r003 的广泛产品 allowlist/product flag；
- 创建 r005，依赖 `FE-W00`、`MVP-W00`、`FE-W01`，并统一为 docs-only；
- 本 revision 即使依赖完成也不自动恢复实机 Pilot，必须新建 Plan revision。

## 2026-07-28 — r006 批准归因更正

- r005 把第一轮 BLOCK reviewer 误写入 `approved_by`；
- 新建 r006，只保留用户授权；
- blocked、FE-W00/MVP-W00/FE-W01 依赖、docs-only 范围和全部现场 Gate
  不变。

## 2026-07-28 — Recovery append-only 完整性恢复

- import tag 中本 Journal 的前 7145 bytes 已恢复为原 SHA-256
  `56c300e815e91170cbaffa145d02c1f9cec97bcf35632d649b2f81fa4f4c6d3e`；
- r004–r006 事件保留在冻结前缀之后；当前权威状态见 Registry 和
  `plan-r006.md`，不再改写本文件顶部历史文字。
