# TC-W01 S09 r004 Independent Review 4

Evidence ID：`TC-EVID-W01-010`

状态：`failed`

Reviewed exact remote commit：
`e879b0e2b7194d256d29955d554c464fd12d72bf`

Reviewed tree：
`7caff593f58b0d6bfeb6358e88eecaac45ccc521`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 1 | 2 | BLOCK |
| security | 0 | 2 | 2 | BLOCK |
| docs/governance | 0 | 1 | 1 | BLOCK |

## P1

- code/security：`..` 和 percent-encoded `..` 可绕过 `/dev`、`/proc`、
  `/sys` boundary；
- security：Windows `COM¹/LPT²/COM³` DOS device name 未覆盖；
- docs/governance：r002 修改 `teamcontrol/service.go`，但 r002 Plan/Task
  scope 未授权，后续 Evidence 也未登记该 provenance 偏差。

## P2

- Context compiler schema/version、state ancestor/owner/TOCTOU 仍需后续
  独立加固；
- Registry URI 缺长度上限，部分 parse/source-kind error 仍回显输入；
- `TC-EVID-W01-005/007/009` 的 collecting 状态需要由后续 failed Evidence
  明确 supersede。

## 处理决定

- 本 exact review 保留为失败 Evidence，不修改已推送 commit；
- 创建 r004，先激活与冻结，再修改产品代码；
- r004 显式授权 `teamcontrol/service.go`，统一 raw/decoded lexical path
  boundary 并补 superscript device 回归；
- r004 新 exact SHA 重新执行 code/security/docs 三路完整验收。
