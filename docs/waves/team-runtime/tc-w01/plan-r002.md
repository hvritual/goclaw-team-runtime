---
schema: goclaw.wave/v1
wave_id: TC-W01
track_id: TEAM-RUNTIME-2026-07
title: Team Control registries, budgets, and deterministic context compiler
revision: 2
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-29-independent-agents-and-auto-continue
owner: Codex root agent
reviewers:
  - independent_code_reviewer
  - independent_security_reviewer
  - independent_documentation_reviewer
depends_on:
  - TR-W00
supersedes:
  - docs/waves/team-runtime/tc-w01/plan-r001.md
created_at: 2026-07-29
updated_at: 2026-07-29
steps:
  - TC-W01-S01
  - TC-W01-S02
  - TC-W01-S03
  - TC-W01-S04
  - TC-W01-S05
allowed_change_scope:
  - docs/waves/**
  - docs/**
  - teamcontrol/types.go
  - teamcontrol/inputs.go
  - teamcontrol/store.go
  - teamcontrol/validation.go
  - teamcontrol/policy.go
  - teamcontrol/queries.go
  - teamcontrol/controlplane.go
  - teamcontrol/controlplane_test.go
  - teamcontrol/service_test.go
  - gateway/team_control.go
  - gateway/team_runtime_test.go
  - gateway/server_auth_test.go
  - cli/team.go
  - cli/system.go
  - ui/src/team/TeamPage.tsx
  - ui/src/team/types.ts
  - ui/src/team/client.ts
  - ui/tests/**
product_code_changes_allowed: true
---

# TC-W01 r002 — 中央 Registry、预算账本与 Context Compiler

本 Wave 把现有 TeamControl 的成员、Token、项目、仓库、文档、组件和策略
能力扩展为 Runner 可消费的中央治理合同。它不执行任务、不下载 Runner
binary、不直接读取 Obsidian Vault，也不接触成员本机 Codex OAuth。

## 已确认缺口

现有 `teamcontrol` 已有用户、个人访问凭据、项目 RBAC、Repository、
Document、Component 与 PolicyBundle，但没有以下中央权威状态：

1. 可按团队、项目和成员配置并发与 Token 上限的预算，及幂等 usage ledger；
2. 带版本、checksum、批准状态和项目范围的 Knowledge Source Registry；
3. 带版本、checksum、兼容约束和批准状态的 Skill Registry；
4. Runner release/channel 元数据 Registry；
5. 将项目、仓库、Policy、批准知识/Skill 和预算快照编译为不可变、
   可校验 Context Bundle 的确定性合同；
6. 对上述状态的项目 RBAC、审计字段、并发安全和 Gateway/CLI 查询入口。

## 数据与安全合同

- 所有资源保存 `project_id`，服务端先按存储资源解析项目，再授权；
- Budget 使用整数 token 单位，不用浮点；limit `0` 表示未配置而非无限；
- usage event 必须携带稳定 `event_id`，相同 payload 幂等，不同 payload
  复用 ID 冲突；累计使用不得溢出或超过硬上限；
- Knowledge/Skill 只保存 URI、revision、SHA-256、metadata 和批准状态，
  不复制 Vault 内容、Token 或 OAuth；
- Runner release 保存平台、架构、版本、artifact URI、SHA-256、最小协议
  和 channel；不会在本 Wave 下载或执行；
- Context Bundle 按稳定排序和 canonical JSON 计算 SHA-256，包含输入资源
  ID/revision/hash、resolved policy、预算 snapshot 和 compiler version；
- 编译只纳入 `approved` 且 checksum 完整的 Knowledge/Skill；缺失、
  跨项目引用或 unsupported compiler version 失败关闭；
- mutation 要求 `project.manage` 或对应 write action；普通成员只读自己
  有权限的项目，跨项目 ID 返回 not found/forbidden，不泄露存在性；
- state schema 以前向兼容方式初始化新 map；写入继续 atomic rename/fsync。

## Steps

| Step | 内容 | 状态 |
|---|---|---|
| `TC-W01-S01` | 冻结 exact Task、Policy manifest、数据/RBAC/幂等矩阵 | `active` |
| `TC-W01-S02` | 实现预算、Knowledge、Skill、Runner release Registry | `planned` |
| `TC-W01-S03` | 实现 deterministic Context Compiler 与 hash/provenance | `planned` |
| `TC-W01-S04` | 接入 Gateway RPC、CLI 和最小 Team Control projection | `planned` |
| `TC-W01-S05` | deterministic Gate、迁移/并发/拒绝测试与三路复核 | `planned` |

## Acceptance

- [ ] Task base 已包含本 Plan、Registry 和 Policy manifest；
- [ ] 旧 schema fixture 可无损加载，新 map 初始化后原状态保持；
- [ ] Budget limit、幂等 usage、超限、整数溢出和并发写测试通过；
- [ ] Knowledge/Skill/Runner release 的 CRUD/list、项目隔离、状态与 checksum
      验证矩阵通过；
- [ ] Context Bundle 对相同输入 byte/hash 稳定，任一输入 revision/hash 改变
      都产生新 bundle hash；
- [ ] 未批准、无 checksum、跨项目、缺失资源和越权编译均失败关闭；
- [ ] Gateway/CLI 不回显 access token、device key、Codex OAuth 或知识正文；
- [ ] Team Control projection 显示预算与 Registry 摘要，并有 loading/empty/
      denied/error 状态，不把浏览器作为权威状态；
- [ ] `go test ./...`、关键包 race、`go vet ./...`、UI test/build 通过；
- [ ] exact review commit 的 code/security/docs 均 P0=0/P1=0。

## Rollback

- schema、canonical hash 或幂等合同失败：停止 Gateway mutation，保留旧
  state backup，不尝试部分写入；
- budget ledger 发现重复/溢出/超限：整次 transaction 回滚；
- context input 不完整：不生成 bundle、不让 Runner 降级为本地自行拼装；
- 任一 P1 未关闭：TC-W01 保持 active，RN-W01 不激活。

## 后继边界

- Runner 注册、兼容协商、下载、原子更新与回滚属于 `RN-W01`；
- project-scoped MCP server 和 Runner/Codex 注入属于 `INT-W01`；
- native installer、签名/公证、GitHub Actions 与三机试点属于 `REL-W01`。
