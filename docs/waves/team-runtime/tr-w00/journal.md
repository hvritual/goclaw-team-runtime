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

## 2026-07-28 — r001 Task Freeze

- activation base：`e87aa3ac3330f3259a12d2a7f3198ee2726d6814`；
- base tree：`35ec868bdc34261a4efc0e70c80d3e11c85ce3a3`；
- Task：`TR-W00-APP-SPLIT-001`；
- Policy manifest SHA-256：
  `61613faf29b804b6723afe9378aa15f31b67a962ac620b7bc53d4b509e9d9977`；
- Draft PR：`https://github.com/hvritual/goclaw-team-runtime/pull/1`；
- freeze 后才允许修改双入口、命令面、构建、部署和文档范围。

## 2026-07-28 — S02–S04 实现与 S05 确定性验证

- 新增 `goclaw-team-control`、`goclaw-runner`，保留兼容 `goclaw`；
- Team Control 不暴露 Runner worker；Runner 不暴露中央管理命令；
- 新增 18 目标跨平台构建、三 binary Linux release 包和 systemd 模板；
- Go unit/vet/race、UI 8/8、Obsidian 6/6、两项 build 全部通过；
- `0.9.0-wave.2` release Gate 退出码 0，四项发布 SHA 重验通过；
- 共享工作区曾重新物化 release stage；精确人工清理成功，归档和 Git tree
  未受影响；
- Evidence：`TR-EVID-W00-001`；
- `S02/S03/S04` complete，`S05` deterministic passed、independent final
  pending；TR-W00 仍 active。
