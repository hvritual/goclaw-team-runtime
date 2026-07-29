# TC-W01 S10 r004 Remediation Verification

Evidence ID：`TC-EVID-W01-011`

状态：`collecting`

## 绑定

- activation remote commit：
  `2afbe53545ea1203d9cabbe95a2fde23e1318e04`；
- activation tree：`0617c07410151ed56841723b41f467ae254d859d`；
- frozen Task remote commit：
  `f2434ca1438cced2851cd841af1b7e48342de0ab`；
- Task：`TC-W01-ACCEPTANCE-006` r004；
- Policy manifest：
  `38b77d31e09806aafbc6b301e82f2f46b77aa9fa03de93c3283ee44d7ddeb41b`。

## 前向修复

- raw、`file:` 和 percent-decoded path 在保留 UNC/namespace 原始检查后，
  使用 slash-normalized lexical clean 折叠 `.`/`..`；
- lexical clean 后拒绝 `/dev`、`/proc`、`/sys`；
- Windows DOS device name 按 rune 识别 ASCII `1..9` 与 `¹²³`；
- Registry URI 限制为 4096 bytes，URL parse 与无效 `source_kind` 错误
  不回显原始输入；
- `teamcontrol/service.go` 在 r004 exact scope 内将项目 read-action 集合
  前向重构为明确 helper，并补五类中央控制 read authorization 回归。

## 回归输入

- `/tmp/../dev/zero`、`file:///tmp/../dev/zero`；
- `/var/../proc/self/environ`、`file:///opt/../sys/kernel`；
- `file:///tmp/%2e%2e/proc/self/environ`；
- `C:\vault\COM¹.txt`、`C:\vault\LPT².log`；
- `file:///C:/vault/COM%C2%B9.txt`；
- malformed/overlong URI 和 secret-shaped `source_kind`。

## Deterministic Gate

- `go test -count=1 ./teamcontrol ./gateway ./cli`：通过；
- `go test -count=1 ./...`：通过；
- `go vet ./...`：通过；
- `go test -race -count=1 ./teamcontrol ./gateway`：通过；
- UI tests `10/10` 与 production build：通过；
- r004 Policy manifest `3/3`：通过；
- `git diff --check`：通过。

新 exact remote candidate 及三路独立验收尚未产生，因此本 Evidence 保持
`collecting`。
