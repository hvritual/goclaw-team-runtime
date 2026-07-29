# TR-W00 Application Boundary Task Freeze r001

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TR-W00-APP-SPLIT-001` |
| Task-Revision | `r001` |
| Work-Item | `team-control-runner-application-boundary` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/tr-w00-team-control-runner-001` |
| Base commit | `e87aa3ac3330f3259a12d2a7f3198ee2726d6814` |
| Base tree | `35ec868bdc34261a4efc0e70c80d3e11c85ce3a3` |
| Issue | `TR-ISSUE-001` |
| Wave-ID | `TR-W00` |
| Wave revision | `r001` |
| Steps | `TR-W00-S02`–`TR-W00-S05` |
| Policy-Bundle | `TR-W00-R001-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tr-w00/POLICY_BUNDLE_SHA256SUMS-r001.txt` |
| Policy manifest SHA-256 | `61613faf29b804b6723afe9378aa15f31b67a962ac620b7bc53d4b509e9d9977` |
| Draft PR | `https://github.com/hvritual/goclaw-team-runtime/pull/1` |
| Frozen at | `2026-07-28` |

## Exact scope

产品代码只允许修改：

```text
cmd/team-control/**
cmd/runner/**
cli/application.go
cli/application_test.go
cli/root.go
main.go
scripts/build-release.sh
scripts/build-apps.sh
deploy/**
Makefile
```

治理、证据和使用文档只允许修改 `docs/**`。本 Task 不授权提前实现
TC-W01 的预算/Context Compiler、RN-W01 的自更新或 INT-W01 的 MCP。

## Acceptance

1. `goclaw-team-control`、`goclaw-runner` 和兼容 `goclaw` 可从同一 commit
   构建。
2. Team Control 和 Runner 的 root command、help、version 以及子命令
   allowlist 有确定性测试。
3. Runner 不能发现或调用 Team/Gateway/Dev/Harness/Ouroboros 管理命令。
4. Team Control 不能发现或调用 `runner work/register/update/rotate-key`。
5. 两个新入口在 Linux/Windows/macOS amd64/arm64 均能交叉编译。
6. 发行脚本生成独立命名产物并保留兼容产物；归档中无秘密或本地配置。
7. 现有 Go/UI/Obsidian 确定性测试不回归。
8. code/security/docs review P0=0/P1=0 后才可关闭 TR-W00。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tr-w00/POLICY_BUNDLE_SHA256SUMS-r001.txt
go test -count=1 ./cli ./cmd/team-control ./cmd/runner
go test -race -count=1 ./cli ./teamcontrol ./workstation ./gateway
go vet ./cli ./cmd/team-control ./cmd/runner
./scripts/build-apps.sh --output /tmp/goclaw-apps
(cd ui && npm ci && npm test && npm run build)
(cd plugins/obsidian-goclaw && npm ci && npm test && npm run build)
git diff --check
git status --short
```

发布脚本验证必须使用任务专用临时目录，不写入源码树；网络依赖失败时记录
environment-blocked，不跳过哈希或凭据扫描。

## Security and rollback

- 命令隐藏不替代服务端 RBAC、个人 Token、device key 与 ExecutionPack；
- 两个新 binary 不嵌入 GitHub、Team、Gateway、Reviewer、Codex OAuth；
- GitHub 授权保存在仓库外，不能复制进文档、日志或构建产物；
- 任一命令面越权或凭据扫描命中即停止发布；
- 回滚删除新入口/构建目标即可，兼容 `goclaw` 保持可构建。
