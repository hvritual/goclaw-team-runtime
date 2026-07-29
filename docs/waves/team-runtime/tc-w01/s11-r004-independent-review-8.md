# TC-W01 S11 r004 Independent Review 8

Evidence ID：`TC-EVID-W01-018`

状态：`failed`

Reviewed exact remote commit：
`e07276efdbff81742bbbd5b31cb3f12c00588ca7`

Reviewed tree：
`3602f9ba749dbdfe2e84435435cbdcb2977514b2`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 1 | 0 | BLOCK |
| security | 0 | 0 | 3 | PASS |
| docs/governance | 0 | 0 | 2 | PASS |

Code reviewer 复现 Registry URI 与 repository relative path 在安全检查前
`TrimSpace`，因此 `C:\vault\safe `、`/vault/safe `、
`services\api `、`services/api ` 被静默改写后接受。

## 处理决定

- 本 exact review 保留为失败 Evidence；
- raw Registry URI 和 relative path 首尾空白直接失败关闭，不静默 trim；
- 增加 raw、`file:`、正反斜杠和 leading/trailing whitespace 回归；
- 新 exact SHA 重新执行三路验收。
