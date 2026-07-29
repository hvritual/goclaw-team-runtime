# TC-W01 S11 r004 Review Remediation 8

Evidence ID：`TC-EVID-W01-017`

状态：`collecting`

## 修复

- raw/file local absolute path 使用与宿主 OS 无关的 slash/drive helper；
- local path 拒绝 NTFS ADS、单反斜杠 rooted、尾空格/尾点 segment；
- raw、file decoded 与 HTTPS/git+HTTPS decoded path 拒绝控制字符和非法
  UTF-8；
- repository relative path 逐段拒绝 Win32 trailing-space/dot alias 和非法
  UTF-8。

## 回归

- `C:\safe.txt:stream`、`safe.txt::$DATA` 及 `file:` 形式；
- `safe/.. /outside`、`safe/. /outside`、反斜杠形式；
- `services./api`、`services/api.`、invalid UTF-8；
- POSIX/Windows raw absolute 与 `file:///C:/...` 的平台中立判定；
- HTTPS/git+HTTPS percent-decoded control。

## Deterministic Gate

- target TeamControl/Gateway/CLI tests：通过；
- full `go test -count=1 ./...`：通过；
- `go vet ./...`：通过；
- TeamControl/Gateway race：通过；
- UI tests `10/10` 与 production build：通过；
- r004 Policy manifest `3/3` 与 `git diff --check`：通过。

新 exact remote candidate 与三路验收尚待产生。
