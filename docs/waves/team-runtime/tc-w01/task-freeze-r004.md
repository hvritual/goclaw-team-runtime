# TC-W01 Path and RBAC Remediation Task Freeze r004

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TC-W01-ACCEPTANCE-006` |
| Task-Revision | `r004` |
| Work-Item | `tc-w01-path-rbac-remediation` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/tc-w01-acceptance-006` |
| Base commit | `2afbe53545ea1203d9cabbe95a2fde23e1318e04` |
| Base tree | `0617c07410151ed56841723b41f467ae254d859d` |
| Issue | `TC-ISSUE-001` |
| Wave-ID | `TC-W01` |
| Wave revision | `r004` |
| Steps | `TC-W01-S10`–`TC-W01-S11` |
| Policy-Bundle | `TC-W01-R004-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r004.txt` |
| Policy manifest SHA-256 | `38b77d31e09806aafbc6b301e82f2f46b77aa9fa03de93c3283ee44d7ddeb41b` |
| Prior failed Evidence | `TC-EVID-W01-010` |
| Frozen at | `2026-07-29` |

## Exact scope

```text
docs/**
teamcontrol/service.go
teamcontrol/service_test.go
teamcontrol/validation.go
teamcontrol/controlplane_test.go
```

## Acceptance

1. `service.go` read actions 在本 Task 中显式前向实现，授权语义不回退；
2. raw、`file:`、percent-encoded local path 在统一 lexical clean 后检查；
3. `/dev`、`/proc`、`/sys` traversal 全部失败关闭；
4. Windows `COM/LPT` ASCII 与 superscript device name 全部失败关闭；
5. Registry URI 限长，parse/source-kind error 不回显原始输入；
6. 既有多项目隔离、RBAC、migration、Context、usage 与 UI 回归通过；
7. 后继 commit trailers 精确使用本 Freeze 的 tuple；
8. code/security/docs final exact review P0=0/P1=0。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r004.txt
go test -count=1 ./teamcontrol ./gateway ./cli
go test -race -count=1 ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
(cd ui && npm test && npm run build)
git diff --check
git status --short
```

## Rollback

- 不确定的 local path 或超长 URI：拒绝且不写入 state；
- RBAC 前向重构回归失败：恢复到本 Task base 的等价授权语义；
- 任一 P1 未关闭：TC 保持 active，RN 不激活。
