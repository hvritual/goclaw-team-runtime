# TR-EVID-W00-002 — r001 Independent Review

状态：`blocked / remediation-planned`

审查对象为 PR #1 exact head
`9d6a25276c9eda32360b801aa2c5fde1bd46e863`。三名 reviewer 均为只读
独立 agent，没有修改文件、分支或 PR。

| Review | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 2 | 3 | BLOCK |
| security | 0 | 2 | 3 | BLOCK |
| docs/governance | 0 | 4 | 1 | BLOCK |

## P1 findings

### Code

- source release allowlist 未包含两个 `cmd` 入口；
- cross-build 目标目录不是封闭集合，陈旧文件可能进入 checksum。

### Security

- 没有负向证据证明 Codex 工具不能读取真实 `CODEX_HOME`；
- binary secret scan 漏掉 Reviewer、device key、Codex access/refresh token，
  且 `grep` 前导连字符处理不安全。

### Docs / governance

- Wave current projection 仍指向已 superseded 的 FE-W01；
- 状态机未定义本次使用的 `active -> superseded`；
- PR 在自身三路验收 Gate 前合并；
- 根 README 仍引导用户克隆上游单应用源码。

## Forward handling

所有 P1 由 `TR-W00 r002` / `TR-W00-ACCEPTANCE-002` 前向修复。P2 中的
真实入口测试、clean、build provenance 和 systemd hardening 在不扩大
产品合同的前提下一并处理；完整原生 Windows/macOS packaging 仍属于
REL-W01。TR-W00 在新 exact commit 通过三路复核前保持 active。
