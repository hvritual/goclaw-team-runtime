# TC-W01 Control Plane Task Freeze r002

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TC-W01-CONTROL-003` |
| Task-Revision | `r002` |
| Work-Item | `tc-w01-central-registries-budget-context` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/tc-w01-control-003` |
| Base commit | `c29dabdee2f0551ad57996611e61e40521e3f7ee` |
| Base tree | `9f08542676dec3f4680107613027392974dd3d50` |
| Issue | `TC-ISSUE-001` |
| Wave-ID | `TC-W01` |
| Wave revision | `r002` |
| Steps | `TC-W01-S02`–`TC-W01-S05` |
| Policy-Bundle | `TC-W01-R002-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r002.txt` |
| Policy manifest SHA-256 | `0950b7ee6a2a15a0d8b7093cb4f7dbb513a1691c7c4f38755710a91a25d52b2b` |
| Frozen at | `2026-07-29` |

## Exact scope

```text
docs/**
teamcontrol/types.go
teamcontrol/inputs.go
teamcontrol/store.go
teamcontrol/validation.go
teamcontrol/policy.go
teamcontrol/queries.go
teamcontrol/controlplane.go
teamcontrol/controlplane_test.go
teamcontrol/service_test.go
gateway/team_control.go
gateway/team_runtime_test.go
gateway/server_auth_test.go
cli/team.go
cli/system.go
ui/src/team/TeamPage.tsx
ui/src/team/types.ts
ui/src/team/client.ts
ui/tests/**
```

不得修改 Runner update/work loop、MCP bridge、Obsidian adapter、release
scripts、installer 或 GitHub Actions。

## Frozen contracts

1. Budget limit 和 usage ledger 使用整数，event ID 幂等，冲突/超限/溢出
   整笔失败；
2. Knowledge/Skill/Runner release 全部 project-scoped，URI 与 SHA-256
   只保存元数据，不保存正文或 secret；
3. approved Knowledge/Skill 才可进入 Context Bundle；
4. Context Bundle canonical JSON、稳定排序、compiler version 和 SHA-256
   可重复；
5. 服务端从存储资源解析 project，再执行 RBAC；请求 project ID 不可覆盖；
6. state 新 map 对旧文件惰性初始化，atomic write/fsync 合同保持；
7. UI 只投影 central state，不存 Team Token、device key 或 Codex OAuth。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r002.txt
go test -count=1 ./teamcontrol ./gateway ./cli
go test -race -count=1 ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
(cd ui && npm test && npm run build)
git diff --check
git status --short
```

新增测试必须覆盖：

- 旧 state normalize/migration；
- budget 幂等、payload conflict、超限、overflow 和并发；
- Registry checksum/status/RBAC/cross-project 拒绝矩阵；
- Context Bundle byte/hash determinism、输入变化和不完整输入失败；
- Gateway principal/project authorization；
- UI loading/empty/denied/error projection。

## Security and rollback

- 不接收、不扫描、不持久化 Codex OAuth、device key、个人 Token 原值；
- Knowledge URI 不自动 fetch；Runner artifact URI 不自动下载；
- mutation 失败时 file store state/revision 不变化；
- 任一 P1 未关闭，TC-W01 保持 active，RN-W01 不激活。
