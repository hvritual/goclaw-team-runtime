# TC-W01 S11 r004 Independent Review 5

Evidence ID：`TC-EVID-W01-012`

状态：`failed`

Reviewed exact remote commit：
`325448250b0fd5279b02fb6c2db95912cfe83466`

Reviewed tree：
`bdf472767dbba2d4fb2e68fbe1c3c00b316e84fd`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 0 | 3 | PASS |
| security | 0 | 1 | 5 | BLOCK |
| docs/governance | 0 | 0 | 0 | PASS |

## P1

跨平台 rooted path 的 DOS device 扫描仍以前置 Windows drive-letter 为条件。
因此 `/NUL`、`/vault/COM1.txt`、`file:///NUL`、
`file:///vault/LPT².log` 和旧式 `file:///C|/NUL` 可在非 Windows
Team Control 宿主上被接受。

## 同范围 P2

- raw absolute Registry URI 在 parse 前返回，控制字符可进入 durable state；
- Owner/Maintainer 在验证 Action 是否为已知常量前直接允许未知 Action。

Context compiler 版本、Policy canonical legacy、state path TOCTOU 和实际
Runner dereference boundary 留作后续独立加固，不阻断本轮。

## 处理决定

- 本 exact review 保留为失败 Evidence；
- 所有 rooted raw/decoded local path 都逐段扫描 DOS device，不依赖 drive；
- 拒绝旧式 `C|` drive separator；
- Registry URI 拒绝控制字符且错误不回显输入；
- project authorization 先验证 Action 是已知枚举，再应用 role；
- 新 exact SHA 重新执行三路验收。
