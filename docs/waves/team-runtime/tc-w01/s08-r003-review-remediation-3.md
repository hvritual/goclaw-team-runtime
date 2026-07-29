# TC-W01 S08 r003 Review Remediation 3

Evidence ID：`TC-EVID-W01-007`

状态：`collecting`

针对 `TC-EVID-W01-006` 的唯一 P1：

- `file:` URI 在 parse 和 percent-unescape 后再次执行 UNC/device boundary；
- 拒绝 `file:////server/share`、`file://localhost//server/share`、
  `file:////./PIPE/name`、`file:////%3F/C:/...`；
- raw path、Windows drive、HTTPS/git+HTTPS allowlist 保持原合同；
- 直接父目录必须是真实目录；Unix 下不得允许 group/other 写。

新 exact candidate 的以下 Gate 全部通过：

- Policy manifest `3/3 OK`；
- `go test -count=1 ./teamcontrol ./gateway`；
- `go test -count=1 ./...`；
- `go vet ./...`；
- `go test -race -count=1 ./teamcontrol ./gateway`；
- UI `10/10` 与 production build；
- `git diff --check`。

等待推送新的 exact SHA 后重新执行 code/security/docs 三路验收。
