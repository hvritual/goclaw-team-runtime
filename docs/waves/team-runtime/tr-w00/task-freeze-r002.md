# TR-W00 Independent Acceptance Remediation Task Freeze r002

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TR-W00-ACCEPTANCE-002` |
| Task-Revision | `r002` |
| Work-Item | `tr-w00-independent-acceptance-remediation` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/tr-w00-acceptance-fixes-002` |
| Base commit | `4625c16e01f03f3b8f90c3b22ef71fa56388e6e0` |
| Base tree | `a088e1fbe120ab1819c0c71f9baf4b181631a922` |
| Issue | `TR-ISSUE-002` |
| Wave-ID | `TR-W00` |
| Wave revision | `r002` |
| Steps | `TR-W00-S07`–`TR-W00-S08` |
| Policy-Bundle | `TR-W00-R002-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tr-w00/POLICY_BUNDLE_SHA256SUMS-r002.txt` |
| Policy manifest SHA-256 | `c8a370406594da4b7f5f6bfa1dbc1bcfb9a775bf989521291fdfde143f5e4815` |
| Prior merged PR | `https://github.com/hvritual/goclaw-team-runtime/pull/1` |
| Frozen at | `2026-07-29` |

## Exact scope

```text
README.md
docs/**
scripts/build-release.sh
scripts/build-apps.sh
Makefile
cli/application_test.go
workstation/localexec.go
workstation/localexec_test.go
deploy/systemd/**
```

不得提前实现 TC-W01 的 Registry/预算/Context Compiler、RN-W01 的升级
生命周期或 INT-W01 的 MCP。

## Acceptance

1. source archive 包含并可解包编译两个 dedicated entrypoint；
2. cross-build 在独立 staging 生成且只发布精确 18 个 binary 和 checksum；
3. credential Gate 覆盖冻结的 Team/Gateway/Reviewer/Runner/Codex/GitHub
   secret 变量，前导连字符安全，命中时不发布；
4. Runner 的 Codex named permission profile 对真实 `CODEX_HOME` 为
   `deny`，worktree 为 `write`、命令网络关闭；
5. 每次模型执行前运行无模型负向 sandbox canary；profile、OS sandbox 或
   read-deny 不受支持时失败关闭；
6. actual entrypoint、clean、build provenance 的 code P2 回归补齐；
7. current Wave/state machine/root README/提前合并偏差一致；
8. code/security/docs 新一轮 exact-commit review 为 P0=0/P1=0。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tr-w00/POLICY_BUNDLE_SHA256SUMS-r002.txt
go test -count=1 ./cli ./cmd/team-control ./cmd/runner ./workstation
go test -race -count=1 ./cli ./teamcontrol ./workstation ./gateway
go vet ./cli ./cmd/team-control ./cmd/runner ./workstation
bash -n scripts/build-apps.sh scripts/build-release.sh
./scripts/build-apps.sh --output "$(mktemp -d)/apps"
SOURCE_ONLY=1 RELEASE_VERSION=0.9.0-wave.3 ./scripts/build-release.sh
(cd ui && npm ci && npm test && npm run build)
(cd plugins/obsidian-goclaw && npm ci && npm test && npm run build)
git diff --check
git status --short
```

真实 Codex sandbox canary 必须由 Runner 目标 Linux/WSL2/Lima 环境再次执行；
仓库单元测试使用 fake Codex 验证参数与失败关闭控制流，不声称替代 OS
enforcement evidence。

## Security and rollback

- 不复制、不解析、不记录 `CODEX_HOME` 中的凭据内容；
- canary 只测试目录/文件不可读，不调用模型，不回显目标内容；
- named permission profile 为 beta 能力，CLI 不支持时 Runner 停止；
- 构建只在 task-specific stage 产生候选，验证成功后才发布；
- 任一 P1 或 credential finding 未关闭，TR-W00 保持 active。
