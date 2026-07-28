# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

## [0.8.0-pilot.1] - 2026-07-27

### Added

- Linux native、WSL2 和 Lima 统一 Linux substrate Runner 合同、
  `runner doctor`、Linux amd64/arm64 发布包及目标平台模板。
- 三人严格治理：`planner-service` 创建者、Wave/Step/plan/Registry
  哈希运行时绑定、三类人工职责分离和旁路/终态防线。
- `pilot check/backup/verify-backup/restore`，含维护锁、age 冷备、
  Git bundle、语义校验和仅恢复到新目录。
- 项目级 Web Console 会话、项目选择、Bug/WorkItem/Assignment 操作、
  重连/刷新与项目级聊天历史。

### Changed

- 原生 Windows/macOS 仅作为控制 CLI 目标；开发 Runner 必须运行在
  native Linux、WSL2 Linux guest 或 Lima Linux guest。
- Team `dev create` 只发送 `wave_step` 意图，Gateway 从冻结 Git base
  解析并覆盖全部 Wave 权威字段。
- 项目会话文件名改为版本化逐段 base64url 编码，模糊旧键迁移失败关闭。

### Security

- Runner 隔离 HOME/XDG/TMP，固定安全 PATH，拒绝危险 Git hook/filter/
  helper/driver 配置，并在取消时终止整个 Unix 进程组。
- WebSocket 项目广播要求已认证 principal、服务器端项目授权及精确
  `project_id/topic_id`；缺失或冲突 scope 失败关闭。
- 发布脚本恢复扫描源码归档中的链接、二进制和 credential-like material，
  并为全部可下载归档生成 SHA-256。

### Verification boundary

- 确定性 Go、race、vet、Web Console、Obsidian、交叉编译与归档 Gate
  已通过。
- 真实 bwrap/WSL2/Lima、三台电脑 Codex OAuth、飞书、Obsidian Desktop
  和现场浏览器仍是试点准入项；本版本不宣称已在生产上线。

## [0.7.0] - 2026-07-25

### Added
- Team Web Console：总览、对话、规格、记忆、审批、开发、团队、进度和 Harness 九个项目工作区。
- 浏览器 HttpOnly/SameSite 会话、CSRF 校验、同源 WebSocket 与安全响应头。
- 通用 `harness.knowledge_root`、`filesystem/git` 知识后端，以及 Git revision 乐观锁。
- Catalog `git+markdown://` 来源、来源类型和不可变 revision。

### Changed
- Team Web Console 成为默认团队控制面；Obsidian 插件降为可选适配器。
- 发布默认构建并嵌入 Web Console，默认不构建 Obsidian 插件。
- 旧 `harness.vault_path` 仅作为 `knowledge_root` 的兼容别名保留。

### Security
- Team Token 与 Gateway Token 不进入浏览器持久存储。
- WebSocket 拒绝跨站 Origin；浏览器 RPC 变更请求要求会话 CSRF Token。
- Git-backed Markdown 审批同时验证目标 SHA-256 与仓库 HEAD revision。

### Added
- `agent` command: Add `--agent` flag for specifying agent ID
- `agent` command: Add `--max-iterations` flag to limit agent loop iterations (default: 15)
- `agent` command: Add `--stream` flag for streaming output support
- `agent` command: Add validation requiring either `--agent` or `--session-id` flag
- Agent loop: Add context cancellation check to prevent hanging on timeout
- Agent loop: Add max iterations check to prevent infinite loops
- Agent: Add streaming events (`EventStreamContent`, `EventStreamThinking`, `EventStreamFinal`, `EventStreamDone`)

### Changed
- `agent` command: `--timeout` default changed from 120 to 600 seconds
- `agent` command: `--thinking` changed from boolean to string level (off|minimal|low|medium|high)
- `agent` command: `--message` now supports `-m` shorthand
- `agent` command: Updated flag descriptions to match openclaw CLI format
- Orchestrator: `streamAssistantResponse` now uses `ChatStream` for streaming providers

## 2026-02-27

### Added
- Feishu plugin: Add cron job support

### Changed
- Improve prompts

## 2026-02-26

### Added
- ACP (Agent Communication Protocol) support
- Feishu plugin: Add typing indicator
- Feishu plugin: Add image upload support
- Telegram: Add typing indicator (#19)

### Changed
- Feishu plugin: Improve markdown rendering
- Feishu plugin: Only response to its messages
- Feishu plugin: Improve logging and set log-level

### Fixed
- Fix wrong tool messages
- Fix `run_shell` name issue (#25)

## 2026-02-25

### Changed
- Improve config handling
- Improve SOUL.md template
- Refactor skills loading

### Fixed
- Fix bindings issue

## 2026-02-24

### Added
- Support feishu channel (#7)

### Changed
- Refactor agent architecture
- Improve agent performance

### Fixed
- Fix Windows issues (#23, #24)
- Merge PR from @qiangmzsx (#7)

## 2026-02-13

### Added
- Add infoflow support
- Add more logging

### Changed
- TUI and channels use the same logic
- Improve skills handling
- Improve web fetch
- Improve goreleaser configuration

### Fixed
- Fix Chinese issue in readline
- Fix readline issues
- Fix tool_call_id issues
- Fix goreleaser config

## 2026-02-12

### Added
- Add go-releaser for release v0.1.0
- Add "hi robot" feature
- Integrate qmd (#3)

### Changed
- Re-implement the agent like pi-mono (#10)
- Refactor skill package
- Improve agent/skills integration

### Fixed
- Fix agent command execution
- Fix feishu issue (#7)
- Fix logger race error (#9)
- Fix channels status command

## 2026-02-11

### Added
- Add tests
- Add history for readline
- Add `/status` command
- Support 8 types of config files
- Add find-skills feature
- Gemini image generation support

### Changed
- Improve channels
- Improve skills
- Improve QQ channel
- Set maxTokens configuration

### Fixed
- Fix go mod name
- Improve logics for tool failures (#2)
- Drop '400 input token limit is 97280' error

## 2026-02-10

### Added
- Add CLI commands

### Changed
- Improve agent loop
- Improve QQ channel

### Fixed
- Fix sending messages to QQ

## 2026-02-09

### Changed
- Re-implement smart_search with crawl4ai
- Improve browser tool usage

## 2026-02-08

### Changed
- Re-implement the browser tool

## 2026-02-06

### Added
- Initial commit
- Initial workable code
- Skills subcommands
- Clawhub subcommand for skill registry management
- Browser authentication for clawhub login
- QQ Official Bot API channel
- QQ and WeWork channel configurations
- Browser tool
- Slash commands
- Sandbox implementation

### Changed
- Change the logo

### Removed
- Remove legacy QQ WebSocket implementation
- Remove clawhub

## earlier

- Basic agent functionality
- Multi-channel support (Telegram, Discord, etc.)
- Tool system implementation
- Session management
- Memory store
