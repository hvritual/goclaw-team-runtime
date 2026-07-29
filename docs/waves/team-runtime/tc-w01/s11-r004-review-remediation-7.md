# TC-W01 S11 r004 Review Remediation 7

Evidence ID：`TC-EVID-W01-015`

状态：`collecting`

## 修复

- `file:` PathUnescape 后拒绝 Unicode control character，不回显输入；
- DOS device segment 先取扩展/ADS 前 base，再 trim 尾部空格/点；
- repository relative path 将 `\` 统一为 `/`，用 `path.Clean` 做平台中立
  lexical fold；
- relative path 拒绝 POSIX absolute、Windows drive/drive-relative、UNC、
  `..` escape、DOS device 与 control character；
- 中文 Registry 合同适用版本更新为 `TC-W01 r004`。

## 回归

- `file:///vault/%00`、`%0a`、`%0d`、`%09`；
- `NUL .txt`、`COM1 .log` 的 raw/rooted/drive 形式；
- `..\outside`、`C:\outside`、`C:outside`、`\\server\share`、`NUL`；
- 正常 slash/backslash repository relative path canonicalization。

## Deterministic Gate

- target TeamControl/Gateway/CLI tests：通过；
- full `go test -count=1 ./...`：通过；
- `go vet ./...`：通过；
- TeamControl/Gateway race：通过；
- UI tests `10/10` 与 production build：通过；
- r004 Policy manifest `3/3` 与 `git diff --check`：通过。

新 exact remote candidate 与三路验收尚待产生，因此本 Evidence 保持
`collecting`。
