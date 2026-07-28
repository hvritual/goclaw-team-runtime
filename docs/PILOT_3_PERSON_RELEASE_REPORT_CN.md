# GoClaw 三人试点交付报告

版本：`0.8.0-pilot.1`

结论：当前源码和发布物已达到“三人技术试点候选”，尚未达到“现场试点已
上线”。代码、确定性测试、交叉编译和发布归档已收敛；真实三台电脑、真实
Codex OAuth、飞书、浏览器/Obsidian Desktop、bwrap/WSL2/Lima 与历史凭据
责任人证明仍是准入 Gate。

## 当前可用能力

- 一个中央 Gateway 管理团队、项目、仓库、成员、策略、Issue、
  WorkItem、Assignment、开发任务、Runner、证据和项目会话。
- Alice、Bob、Carol 使用各自个人 Token 和 Reviewer Token；严格三人
  模式下 Alice 审 scenario/capacity，Bob 审 risk/cost，Carol 独立 final。
- `dev create --wave-step` 只发送 Step 意图。服务器从冻结 Git base 解析
  active Wave、plan、依赖、范围和 SHA-256，并在 freeze、enqueue、
  accept 重验。
- 每台电脑使用自己的 Codex 订阅 OAuth；中央控制面不收集 OAuth 文件，
  ExecutionPack 不含秘密。
- Runner 执行底座统一为 native Linux、WSL2 Linux guest 或 Lima Linux
  guest；Linux amd64/arm64 均有发布包。原生 Windows/macOS 只允许控制
  CLI，执行任务失败关闭。
- Runner 固定安全 PATH，隔离 HOME/XDG/TMP，拒绝危险 Git 执行配置，
  使用 root-owned bwrap wrapper，取消时终止整个 Unix 进程组。
- Web Console 支持授权项目选择、项目级聊天历史、Bug/任务/负责人流转、
  Runner/审批/进度看板、401 退出和断线刷新。
- Obsidian 插件继续作为可选交互与知识适配器；Vault 只同步 Markdown，
  不承载中央状态、锁、Token、device key 或 OAuth。
- `control.consistency.check` 在跨 TeamControl、Workstation、
  Orchestrator Lite 出现 critical finding 时阻止 enqueue/accept。
- `pilot backup/verify-backup/restore` 提供维护锁、age 加密、恢复语义
  校验、Git bundle 和仅恢复到新目录。

## 已通过的工程 Gate

- 发布脚本内的选定 Go 包测试全部通过。
- `session`、`orchestratorlite`、`teamcontrol`、`workstation`、
  `gateway`、`cli` 的 `go test -race -count=1` 通过。
- 同一关键包集合的 `go vet` 通过。
- Web Console 测试 `8/8`，TypeScript/Vite 生产构建通过。
- Obsidian adapter 测试 `6/6`，TypeScript/esbuild 构建通过。
- Linux amd64/arm64 Runner 构建通过。
- Darwin amd64/arm64、Windows amd64/arm64 控制 CLI 交叉编译通过。
- 三 owner、三任务并发 claim 隔离测试通过。
- 源码归档 allowlist、危险成员、链接、常见二进制和 credential-like
  material 恢复扫描通过。
- 四个归档的 `sha256sum -c` 通过。

## 尚未通过的现场 Gate

- 当前容器不能完成真实 bwrap probe，因此未证明目标机器的 bwrap、
  WSL2 或 Lima 隔离。
- 云端 Browser 安全策略拒绝访问本机 `127.0.0.1`；三独立
  BrowserContext、Desktop/Mobile 和真实页面交互必须在部署网络复验。
- 未提供三名成员和中央可选模型账号的真实 ChatGPT Workspace/Codex
  OAuth。
- 未提供真实飞书 App、回调、白名单、pairing 和项目路由。
- 未在真实 Obsidian Desktop、多机 Vault Sync 上运行兼容性和冲突验收。
- 历史 credential-shaped material 仍需责任人证明已撤销、轮换或从未
  有效。
- 当前环境 ptrace 被拒绝，旧 syscall 零出站 trace 不能在这里采集。

## 发布物

`dist/` 包含：

```text
goclaw-team-runtime-linux-amd64-0.8.0-pilot.1.tar.gz
goclaw-team-runtime-linux-arm64-0.8.0-pilot.1.tar.gz
goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz
obsidian-goclaw-plugin-0.8.0-pilot.1.tar.gz
SHA256SUMS-0.8.0-pilot.1.txt
```

先执行：

```bash
sha256sum -c SHA256SUMS-0.8.0-pilot.1.txt
```

完整配置、身份创建、三类 Runner、任务四审、备份恢复、飞书/Obsidian
边界和放行清单见
[`PILOT_3_PERSON_DEPLOYMENT_CN.md`](PILOT_3_PERSON_DEPLOYMENT_CN.md)。

## 建议的三人现场顺序

1. 在隔离环境部署单实例 Gateway 和 Caddy HTTPS/WSS。
2. 只创建 Alice、Bob、Carol，分别签发个人 Token 和 Reviewer Token。
3. 在 native Linux、WSL2 或 Lima 中分别完成自己的 `codex login`。
4. 三台安装同一 root-owned wrapper，注册 Runner 并通过
   `runner doctor --json`。
5. 创建一个只改文档的 smoke Issue/WorkItem/Assignment。
6. Alice/Bob 完成四审，Alice Runner 领取任务，Carol 独立验收。
7. 三人分别做 Web Console 登录、越权负例、刷新与断线恢复。
8. 配置并验证真实飞书；需要时再安装 Obsidian 插件。
9. 停 Gateway，创建 age 冷备，验证并恢复到新目录。
10. `control.consistency.check` 无 critical 且 `pilot check --json`
    返回 `ready=true` 后，才把状态改为“限时三人试点已开始”。
