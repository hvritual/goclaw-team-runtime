# RN-W01 Runner Lifecycle Task Freeze r002

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `RN-W01-LIFECYCLE-001` |
| Task-Revision | `r002` |
| Work-Item | `runner-dual-profile-lifecycle` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/rn-w01-lifecycle-001` |
| Base commit | `f7b30062468919db5ca8c4fcb5148f4493188832` |
| Base tree | `4eeb0a92368f29e9bfa5aa0a7cd9fb2ff62cd8b2` |
| Issue | `RN-ISSUE-001` |
| Wave-ID | `RN-W01` |
| Wave revision | `r002` |
| Steps | `RN-W01-S02`–`RN-W01-S08` |
| Policy-Bundle | `RN-W01-R002-POLICY` |
| Policy manifest | `docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r002.txt` |
| Policy manifest SHA-256 | `37c0f491b640c21bc8ea22d70f500f2047c8f46d5f0baab72baff45a5f71e657` |
| Frozen at | `2026-07-29` |

## Exact scope

```text
docs/**
teamcontrol/**
workstation/**
gateway/workstation.go
gateway/workstation_test.go
gateway/team_runtime_test.go
cli/runner.go
cli/runner_test.go
config/**
deploy/**
scripts/**
```

## Contract

1. `strict` 保持现有默认和失败关闭语义；
2. `codex-delegated` 必须由项目 policy 与 Runner capability 双向选择，不能
   隐式 fallback；
3. 两个 profile 均由 GoClaw 强制 repository/worktree/allowed path/diff
   边界；delegated 不宣称 OS network/process sandbox；
4. native Windows/macOS 仅允许 delegated；strict 继续要求受支持的 Linux
   substrate；
5. Runner 公共投影包含当前/目标版本、contract、profile/posture 和 rollout
   状态，不包含 device key、Team Token 或 Codex OAuth；
6. release stage 复验 immutable ID、platform、arch、size 与 SHA-256；
7. activate/rollback 原子且 crash-safe，运行中拒绝替换；
8. 多项目 claim、lease、Evidence 与 update state 不得串线。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r002.txt
go test -count=1 ./workstation ./teamcontrol ./gateway ./cli ./config
go test -race -count=1 ./workstation ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 go test -c ./workstation
GOOS=darwin GOARCH=arm64 go test -c ./workstation
bash -n scripts/*.sh
(cd ui && npm test && npm run build)
git diff --check
git status --short
```

## Independent acceptance

- code reviewer：compatibility、migration、更新状态机、多项目并发；
- security reviewer：目录边界、降级披露、artifact/update、凭据与 rollback；
- docs reviewer：Team Control/Runner 职责、三平台操作、Evidence/governance；
- 任一 P0/P1：不关闭 RN-W01，不激活 INT-W01。

## Rollback

- 未识别 profile/contract/release：拒绝 claim/update；
- delegated policy 缺失或 posture 不匹配：拒绝，不回退 strict 或继续执行；
- 更新 health confirm 失败：恢复最后已验证版本并保留失败 Evidence；
- 远程 fetch 策略未完成：只接受 operator 提供的本地 artifact。
