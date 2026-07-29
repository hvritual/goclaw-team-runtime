# TC-W01 S11 r004 Review Remediation 6

Evidence ID：`TC-EVID-W01-013`

状态：`collecting`

## 修复

- rooted raw 与 `file:` decoded path 全部逐段扫描 DOS device name；
- `/NUL`、任意目录下的 `COM/LPT`、superscript 设备名统一失败关闭；
- `file:///C|/...` 旧式 drive separator 失败关闭；
- Registry URI 在任何 absolute-path 快捷返回前拒绝控制字符；
- `roleAllows` 先以完整 Action enum fail-closed，再执行 Owner/Maintainer/
  Developer/Reviewer/Viewer 权限矩阵。

## 回归

- `/NUL`、`/vault/COM1.txt`；
- `file:///NUL`、`file:///vault/LPT².log`；
- `file:///C|/NUL`；
- 含 CR/LF/NUL 且带 secret-shaped 内容的路径错误不回显；
- 五种 ProjectRole 对未知 future Action 均返回 false。

## Deterministic Gate

- target TeamControl/Gateway/CLI tests：通过；
- full `go test -count=1 ./...`：通过；
- `go vet ./...`：通过；
- TeamControl/Gateway race：通过；
- UI tests `10/10` 与 production build：通过；
- r004 Policy manifest `3/3` 与 `git diff --check`：通过。

新的 exact remote review 尚待执行，因此本 Evidence 保持 `collecting`。
