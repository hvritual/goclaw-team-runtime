# FE-EVID-W01-016 — S10 runtime provider preflight

## 证据主体

| 字段 | 值 |
|---|---|
| 时间 | 2026-07-26 |
| Project-ID | `goclaw-team-runtime` |
| Wave / Plan | `FE-W01` / `plan-r007` |
| Task | `FE-W01-TRANSPORT-R1` revision `5` |
| Worktree | `repair/fe-w01-transport-r5` |
| Task Base | `2f9cd8289d7c05e44d30b70a07e7991036229bbf` |
| R5 HEAD | `5160273fb17502cf02cd10e1a17f5a47b7eb30be` |
| Step | `FE-W01-S10` |
| Actor / role | `Codex root agent` / assignee；静态启动路径 preflight |
| Environment | Linux `6.12.13` x86_64；Go `1.25.5 linux/amd64` |
| 发现阶段 | Playwright 下载完成、任何 synthetic Token/config/runtime 创建之前 |

## 动作、预期与实际

### 动作

在 R5 HEAD 上只读检查真实 CLI 启动链：

```text
goclaw gateway run
→ internal/start.StartAgent
→ config.Validate
→ providers.NewProvider
→ providers.NewSimpleProvider
→ providers.NewProviderFromModelsConfig
→ providers.NewOpenAIProviderWithTimeout
```

复核动作仅使用 `rg`、`sed` 和 `sha256sum` 读取源码；没有生成或执行临时
Gateway config，也没有运行 Gateway。该证据是静态 source preflight，不是
已启动 runtime 的动态复现。

### 预期

r007 冻结的 synthetic config 在全部 provider 关闭时，仍可由真实
`goclaw gateway run` 启动 Gateway，供本地 Playwright 只访问 Team 页面。

### 实际

- `config.Validate` 要求 `models.providers` 至少包含一个 provider；
- 即使跳过该校验，`providers.NewSimpleProvider` 也会返回
  `no LLM provider configured`；
- 加入 OpenAI-compatible provider 但保持空 key 时，
  `NewOpenAIProviderWithTimeout` 返回 `API key is required`；
- 显式 model 定义还必须有非空 `id` 与 `name`。

因此 r007 的“真实 binary + 全部 provider 关闭”合同不可执行。因为该结论在
credential/runtime 之前已由真实源码路径确定，未用一个预期失败的含配置
启动去制造额外敏感状态。

## 源码锚点

| 路径 | R5 内容 SHA-256 | 约束 |
|---|---|---|
| `internal/start/start.go` | `25b617933cba8e44fe545add03b267924555cf64485ce4490eca3c1fcd09ab98` | `StartAgent` 先验证 config，随后无条件构造 provider |
| `config/validator.go` | `388738975f01674c1a200ac2b8e9c49ff4d29dc1e70a5af053ebedcc0fb969db` | 至少一个 provider；显式 model 的 id/name 非空 |
| `providers/factory.go` | `362bae28a3caf16d03fe3673325c6c12e465de54cd2453b211ff5b4f6813e82d` | 无 provider 报错；OpenAI-compatible 分支进入 OpenAI constructor |
| `providers/openai.go` | `ad66922a8c85b8d939a9bb701c57efec4b5b5fe5134a06c50d793324a96a7e03` | 空 API key 在 constructor 阶段失败 |

## 可重复约束

真实 `goclaw gateway run` 走 `internal/start.StartAgent`。该启动路径在创建
Gateway 前无条件调用：

```text
providers.NewProvider(configFile)
```

`providers.NewSimpleProvider` 在 `models.providers` 为空时返回：

```text
no LLM provider configured
```

因此 r007 的两个冻结条件不能同时满足：

1. 使用真实 `goclaw gateway run`；
2. synthetic config 显式关闭全部 provider。

这不是页面缺陷，也未授权产品代码变更；它是 S10 浏览器夹具的启动前置约束。

进一步静态复核 `NewOpenAIProviderWithTimeout` 证明空 API key 也会在构造期
失败。因此 inert fixture 需要固定公开 marker
`test-inert-provider-key`；它不是生产 credential，仅允许写入 `0600`
synthetic config 与 `0600` sentinel，不通过环境变量/CLI 传递，且仍进入
sentinel scan，防止出现在截图、报告或日志。

## 日志与脱敏声明

本步骤未启动 Gateway，故没有 runtime log、Trace、HAR、网络快照或截图。
只读源码检查输出不含 Token/config 内容，未把 raw 命令输出保存为 Evidence；
本文仅记录错误类别、源码 SHA 和下载工件公开版本/内容摘要。

## 安全停止

- 未创建 Gateway/Team Token、config、TeamControl 数据库、browser context 或
  用户 profile；
- 未启动 Gateway、Vite、Chromium 或 Codex；
- 未访问生产 provider/channel/Vault/TeamControl；
- 已完成的下载阶段只有 `playwright@1.55.0` 和 Chromium build `1187`：
  Chromium `140.0.7339.16`，可执行文件 SHA-256
  `2fa605e3639b8cfbe8037d0b8e0324dbf7f9e6ad7beb345374ecd26764e2d92b`；
- 下载时的 `tooling/package-lock.json` SHA-256 为
  `53622035b305ccadd941f377f72f9231deb8394810387cea36196b2fb6a7e3fe`；
  Playwright node_modules regular-file manifest SHA-256 为
  `4a87376e407b7093d8dfa42b3051ffbac1b4f5ec86ac05fddc7da08654968988`，
  Chromium build regular-file manifest SHA-256 为
  `0f88026f00f407c0c858d3ed95da311baec3320450da01cf12fc97363c7b20e7`；
- 下载工件位于 `/tmp/goclaw-fe-w01-r5-playwright-MUTsW4`，不在仓库，
  且产生于任何 synthetic credential 之前。

## 决策输入

候选 r008 需在不改变产品代码的前提下冻结以下测试夹具：

- 配置唯一一个无真实 API key、仅含固定 synthetic marker、固定非空 model
  name、无 header、无 runtime command、指向不可用 loopback endpoint 的
  inert provider，只为满足 config validator 与启动构造器；
- Browser 流程禁止触发 chat/model RPC；
- Gateway 启动前必须开启 syscall 级 `connect()` 审计；Gateway 及子进程
  不得尝试任何出站连接，短暂/失败连接和 inert endpoint 尝试也判 Gate
  失败，不能用 socket 轮询替代；
- 真实 provider credential 和宿主 provider 环境继续由 `env -i` 排除。

r007 按失败关闭规则停止；Registry 在 r008 审批和原子激活前仍指向 r007。
