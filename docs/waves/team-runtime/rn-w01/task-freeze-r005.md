# RN-W01 Device Authentication Remediation Task Freeze r005

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `RN-W01-LIFECYCLE-001` |
| Task-Revision | `r005` |
| Work-Item | `runner-dual-profile-lifecycle` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/rn-w01-lifecycle-001` |
| Base commit | `40e79693cf25eb24d6b0728956684d2491f21247` |
| Base tree | `1e630b831808cec32ee0a89e7b4844de03ac3eed` |
| Issue | `RN-ISSUE-001` |
| Wave-ID | `RN-W01` |
| Wave revision | `r005` |
| Steps | `RN-W01-S07B`–`RN-W01-S08` |
| Policy-Bundle | `RN-W01-R005-POLICY` |
| Policy manifest | `docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r005.txt` |
| Policy manifest SHA-256 | `cf62089d3659605c2050949d26a73d16927039a91228edbf4497ea7f27c73b76` |
| Frozen at | `2026-07-29` |

## Exact scope and contract

采用 `plan-r005.md` 的累计 r002–r005 scope 与 Acceptance。设备凭据只能
调用其自身 `ping/claim/heartbeat/complete/fail`；work loop 必须拒绝成员
Token。claim、release executable/lock、Windows owner、policy write、
legacy migration 和 docs findings 全部修复后，必须对新 exact candidate
重新执行三路独立 review。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r005.txt
go test -count=1 ./workstation ./teamcontrol ./gateway ./cli ./config
go test -race -count=1 ./workstation ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -o /tmp/rn-workstation-windows.test.exe ./workstation
GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go test -c -o /tmp/rn-workstation-darwin.test ./workstation
bash -n scripts/*.sh
(cd ui && npm test && npm run build)
./scripts/build-apps.sh --output <EMPTY_DIR> --version 0.9.0-rn-w01-r005
(cd <OUTPUT_DIR> && sha256sum -c SHA256SUMS)
git diff --check
git status --short
```

## Stop condition

RN-W01 的新 exact candidate 通过 code/security/docs 三路
`P0=0/P1=0` 后，固化 Evidence、把 RN-W01 标记 `complete` 并暂停开发。
不激活 INT-W01 或 REL-W01，不自动 merge Draft PR。
