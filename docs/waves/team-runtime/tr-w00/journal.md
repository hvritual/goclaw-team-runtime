# TR-W00 Journal

本文件只在末尾追加事件。当前状态以 Registry 和 current Plan 为准。

## 2026-07-28 — r001 路线激活

- 用户授权自动规划并完整实现 Team Control 与 Runner；
- FE-W01–FE-W05 保留历史 Evidence，状态改为 `superseded`；
- 新路线顺序为 `TR-W00 → TC-W01 → RN-W01 → INT-W01 → REL-W01`；
- 每个 Wave 完成后必须把 Plan、实现和 Evidence 推送到私有 GitHub；
- GitHub 凭据保持在仓库外，仓库保存可恢复步骤而不是秘密。

| Step | 状态 | 结论 |
|---|---|---|
| `TR-W00-S01` | active | 等待 activation commit 后冻结 exact Task |
| `TR-W00-S02` | planned | 等待 S01 |
| `TR-W00-S03` | planned | 等待 S02 |
| `TR-W00-S04` | planned | 等待 S03 |
| `TR-W00-S05` | planned | 等待 S02–S04 |
