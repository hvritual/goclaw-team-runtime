# TC-W01 S08 r003 Independent Review 2

Evidence ID：`TC-EVID-W01-006`

状态：`failed`

Reviewed exact remote commit：
`961a93291e6867adc977b930e6c483b49a1f7861`

Reviewed tree：
`c560545338cdde96dd7939e15d858273ba93f4f3`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 0 | 2 | PASS |
| security | 0 | 1 | 1 | BLOCK |
| docs/governance | — | — | — | 未启动；overall 已 BLOCK |

Code reviewer 确认初审四项 P1 及全套 deterministic Gate 已关闭。

Security reviewer 确认 raw UNC/device、legacy usage RPC、metadata、hash、RBAC
主路径已关闭，但 `file:////server/share`、
`file://localhost//server/share`、`file:////./PIPE/name` 和 percent-encoded
device path 在 URL parse 后仍具有 UNC/device 语义。父目录可写/路径替换是
P2。

## 处理决定

- 本 exact review 保留为失败 Evidence；
- 在 `file:` parse/unescape 后复用 UNC/device boundary 校验并补四类负例；
- 同时校验直接父目录不是 symlink 且 Unix 下不可被 group/other 写；
- 新 exact SHA 重新执行 code/security/docs 三路完整验收。
