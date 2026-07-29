---
schema: goclaw.wave/v1
wave_id: TR-W00
track_id: TEAM-RUNTIME-2026-07
title: Team Control and Runner application boundary
revision: 1
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-28-auto-wave
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - MVP-W00
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - TR-W00-S01
  - TR-W00-S02
  - TR-W00-S03
  - TR-W00-S04
  - TR-W00-S05
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - cmd/team-control/**
  - cmd/runner/**
  - cli/application.go
  - cli/application_test.go
  - cli/root.go
  - main.go
  - scripts/build-release.sh
  - scripts/build-apps.sh
  - deploy/**
  - Makefile
product_code_changes_allowed: true
---

# TR-W00 r001 — Team Control 与 Runner 应用边界

本 Wave 把现有单一 `goclaw` 发行面收敛为两个可独立构建、安装和升级的
应用：

- `goclaw-team-control`：中央控制面、Gateway、团队治理、项目资源、知识与
  Harness 管理入口；
- `goclaw-runner`：工作站注册、doctor、任务领取、本地 Codex 执行、证据和
  自身状态入口。

本 Wave 只建立应用边界、命令面、构建产物、兼容入口和部署骨架。预算、
Context Compiler、Runner 版本策略和 MCP 知识接入分别由后继 Wave 实现。

## 路线替代

用户把目标从 FE-W01 浏览器门禁提升为完整 Team Control/Runner 产品路线。
因此 FE-W01–FE-W05 保留历史 Evidence，但状态改为 `superseded`，不伪造
`complete`。TR-W00 只依赖已完成的权威源码恢复 `MVP-W00`。

## 应用合同

### Team Control

- 默认命令名、帮助和版本输出使用 `goclaw-team-control`；
- 只暴露中央管理和运行所需命令，不能作为本地 Runner worker 启动；
- 继续使用既有 TeamControl、Workstation scheduler、Orchestrator Lite、
  Harness、Ouroboros、Memory Catalog、Web Console 与渠道实现；
- 保留原 `goclaw` 入口作为一个 release cycle 的兼容入口。

### Runner

- 默认命令名、帮助和版本输出使用 `goclaw-runner`；
- 只暴露 `runner`、连接诊断、配置查看和版本命令；
- 不暴露 bootstrap、成员/Token/项目/策略管理、Gateway server、开发任务
  审批或 Harness 提升命令；
- Runner 二进制中的隐藏代码不是授权边界；服务端个人 Token、项目 RBAC、
  device key、ExecutionPack 与目录/sandbox Gate 仍是安全边界。

## Steps

| Step | 内容 | 产物 | 状态 |
|---|---|---|---|
| `TR-W00-S01` | 激活路线并冻结应用合同 | Plan、Registry、Task Freeze | `active` |
| `TR-W00-S02` | 新增双入口与命令面限制 | 两个二进制、单元测试 | `planned` |
| `TR-W00-S03` | 增加跨平台构建与发行合同 | build script、archives | `planned` |
| `TR-W00-S04` | 更新部署模板和迁移说明 | Linux/macOS/Windows 文档 | `planned` |
| `TR-W00-S05` | 全量确定性验证和独立验收 | Evidence、review | `planned` |

## Acceptance criteria

- [ ] 可从同一 commit 构建 `goclaw-team-control` 与 `goclaw-runner`；
- [ ] 两个程序的名称、帮助、版本输出和允许命令可自动测试；
- [ ] Runner 命令面不能启动控制面或执行团队管理操作；
- [ ] Team Control 命令面不能启动 Runner worker；
- [ ] 原 `goclaw` 兼容入口行为不回归；
- [ ] Linux amd64/arm64、Windows amd64/arm64、macOS amd64/arm64 交叉构建；
- [ ] 发布包不含 Token、device key、Codex OAuth 或 GitHub 凭据；
- [ ] Go test/race/vet 与 UI/Obsidian 确定性 Gate 通过；
- [ ] code/security/docs final review P0=0/P1=0。

## GitHub 记忆合同

- Plan、Task、实现、Evidence、迁移和状态必须提交到
  `hvritual/goclaw-team-runtime`；
- 每个完成 Step 至少有一个带完整 trailers 的 commit 并推送；
- 每个完成 Wave 更新 Registry/Journal/Evidence 后推送；
- GitHub Token、OAuth、device key 和个人 Team Token 永不进入 Git；
- 授权丢失时停止 push，保留本地 commit，并按文档重新授权后续推。

## 停止与回滚

- 双入口导致原 CLI 命令回归：回滚新入口，不删除原 `goclaw`；
- Runner 暴露控制面写操作：按 P0 停止发布；
- 构建产物混入凭据或本地配置：删除候选归档并重新构建；
- 后继预算/MCP/升级范围不得提前混入本 Wave。
