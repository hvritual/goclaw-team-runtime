# TC-W01 S08 r003 Independent Review 3

Evidence ID：`TC-EVID-W01-008`

状态：`failed`

Reviewed exact remote commit：
`2bce7317f810fbe6ebf3a1874dccab02fb4f670a`

Reviewed tree：
`4f9daf57198b5e4cd71386c7c25fce88fd515a1c`

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 0 | 3 | PASS |
| security | 0 | 1 | 2 | BLOCK |
| docs/governance | — | — | — | 未启动；overall 已 BLOCK |

Code reviewer 确认第二轮 P1 与 deterministic Gate 已关闭；剩余 P2 为
Context compiler v1 兼容、Policy canonical trim 和路径祖先边界。

Security reviewer 确认 UNC、parsed file URI、父目录和权限主路径已关闭，
但 `C:\NUL`、`file:///C:/NUL` 与 `/dev/zero` 等操作系统设备或伪文件系统
路径仍可作为本地 Registry URI 接受。祖先目录替换/owner/TOCTOU 与错误中
回显不受信任字段是 P2。

## 处理决定

- 本 exact review 保留为失败 Evidence，不改写历史结论；
- 拒绝 Windows DOS/NT device path 与 Unix `/dev`、`/proc`、`/sys`；
- Registry/Policy 校验错误不回显不受信任 scheme、metadata key 或 policy
  key；
- canonical marshal 前实际写回 trim 后的 Policy 字段；
- 新 exact SHA 重新执行 code/security/docs 三路完整验收。
