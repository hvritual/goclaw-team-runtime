# RN-W01 Complete Release Identity Task Freeze r004

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `RN-W01-LIFECYCLE-001` |
| Task-Revision | `r004` |
| Work-Item | `runner-dual-profile-lifecycle` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/rn-w01-lifecycle-001` |
| Base commit | `400c8005a9d3152f083fbd4b51ca2b6253590679` |
| Base tree | `78da2dfc56586057ac1c583331f77fd0f15659fc` |
| Issue | `RN-ISSUE-001` |
| Wave-ID | `RN-W01` |
| Wave revision | `r004` |
| Steps | `RN-W01-S03D`–`RN-W01-S08` |
| Policy-Bundle | `RN-W01-R004-POLICY` |
| Policy manifest | `docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r004.txt` |
| Policy manifest SHA-256 | `9f01bb43e7efc6ac928712999f5dc17abd4480f291d233624d915693911dae30` |
| Frozen at | `2026-07-29` |

## Exact scope and contract

采用 `plan-r004.md` 的完整 scope，以及 r002/r003/r004 的累计 Acceptance。
新 release 的中央 identity 必须包含正数 `size_bytes`；legacy zero-size
record 可读但不能用于本地 stage。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r004.txt
go test -count=1 ./workstation ./teamcontrol ./gateway ./cli ./config
go test -race -count=1 ./workstation ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 go test -c -o /tmp/rn-workstation-windows.test.exe ./workstation
GOOS=darwin GOARCH=arm64 go test -c -o /tmp/rn-workstation-darwin.test ./workstation
bash -n scripts/*.sh
(cd ui && npm test && npm run build)
git diff --check
git status --short
```

## Stop condition

按用户最新指令，RN-W01 完成并通过三路 exact review 后暂停开发；不激活
INT-W01 或 REL-W01，不自动合并 PR。
