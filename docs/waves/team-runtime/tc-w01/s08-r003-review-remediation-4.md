# TC-W01 S08 r003 Review Remediation 4

Evidence ID：`TC-EVID-W01-009`

状态：`collecting`

本轮只处理第三次 exact review 在 r003 frozen scope 内确认的 P1，并顺带
关闭同一校验边界内的低风险 P2。

## 修复

- 本地路径拒绝 Windows DOS device name、NT device namespace；
- 本地路径拒绝 Unix `/dev`、`/proc`、`/sys` 及其 `file:` 形式；
- raw path 与 parse/unescape 后的 path 复用同一 boundary；
- 不支持的 URI scheme、metadata key、Policy key 不再在错误中回显；
- `style`、`code_style` 在 canonical marshal 前写回 trim 后的值。

## 回归

新增负例覆盖：

- `C:\NUL`、`C:\vault\COM1.txt`；
- `file:///C:/NUL`；
- `/dev/zero`、`file:///dev/zero`；
- `/proc/self/environ`、`file:///sys/kernel`。

## Deterministic Gate

- target TeamControl/Gateway tests：通过；
- full `go test -count=1 ./...`：通过；
- `go vet ./...`：通过；
- TeamControl/Gateway race：通过；
- Team Web Console tests `10/10` 与 production build：通过；
- `git diff --check`：通过。

下一步是推送新的 exact remote commit，并由独立 code/security/docs
reviewer 针对同一 SHA 验收。三路 P0/P1 均为 0 前，本 Evidence 不得标记
`passed`。
