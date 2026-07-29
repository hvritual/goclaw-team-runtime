# TC-W01 S11 r004 Independent Review 6

Evidence ID：`TC-EVID-W01-014`

状态：`failed`

Reviewed exact remote commit：
`74dde015e5afbbc4c904b03c1e9b5bfdd8cb1cd2`

Reviewed tree：
`381e0e44e5296ec3447b0935726f2334e4dfc99c`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 2 | 3 | BLOCK |
| security | 0 | 1 | 2 | BLOCK |
| docs/governance | 0 | 1 | 2 | BLOCK |

## P1

- `file:` percent-decode 后没有复查控制字符，`%00/%0a/%0d/%09` 可进入
  durable Registry；
- DOS device base 后带空格再接扩展名（`NUL .txt`、`COM1 .log`）可绕过；
- `validateRelativePath` 使用宿主 `filepath` 语义，Linux 控制面可接受
  Windows traversal、drive、UNC 和 device path；
- 当前中文 Registry 合同首行仍标记适用 r003，而正文已是 r004 合同。

## 处理决定

- 本 exact review 保留为失败 Evidence；
- `file:` decoded path 复用 control-character boundary；
- DOS base 先截扩展/ADS，再移除尾部空格/点并判断；
- repository relative path 使用平台中立 slash/path clean，拒绝两平台的
  absolute、drive-relative、UNC、traversal、device 与 control character；
- 中文合同适用版本更新为 r004；
- 新 exact SHA 重新执行三路验收。
