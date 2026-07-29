# TC-W01 S11 r004 Review Remediation 9

Evidence ID：`TC-EVID-W01-019`

状态：`collecting`

Registry URI 和 repository relative path 现在在任何 trim/canonical 操作前
比较原值与 `strings.TrimSpace`；不一致则返回不回显输入的错误。

回归覆盖：

- `C:\vault\safe `、`/vault/safe `、` /vault/safe`；
- `file:///vault/safe%20` decoded trailing space；
- `services/api `、`services\api `、` services/api`。

完整 deterministic Gate 已通过：

- `go test -count=1 ./...`；
- `go vet ./...`；
- `go test -race -count=1 ./teamcontrol ./gateway`；
- UI tests `10/10` 与 production build；
- Policy manifest `3/3` 与 `git diff --check`。

新 exact remote candidate 与三路验收尚待产生。
