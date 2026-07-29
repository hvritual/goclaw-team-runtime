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

## 2026-07-29 — r001 三路独立验收与提前合并偏差

- 用户授权独立 code/security/docs agents 完成验收并自动继续路线；
- 三路审查绑定 exact head `9d6a25276c9eda32360b801aa2c5fde1bd46e863`；
- code：P0=0/P1=2/P2=3，BLOCK；
- security：P0=0/P1=2/P2=3，BLOCK；
- docs/governance：P0=0/P1=4/P2=1，BLOCK；
- GitHub 显示 PR #1 已于独立验收前合并，merge commit 为
  `3a75c7376d73e41f33e2b94eb3bb1ca4c30219fd`；
- 该合并不构成 TR-W00 acceptance；r001 Evidence 保持 collecting，
  `TR-ISSUE-002` 进入 fixing；
- r002 只允许前向修复，三路 P1 清零前不激活 TC-W01。

## 2026-07-29 — r002 Task Freeze

- activation commit：`4625c16e01f03f3b8f90c3b22ef71fa56388e6e0`；
- activation tree：`a088e1fbe120ab1819c0c71f9baf4b181631a922`；
- Task：`TR-W00-ACCEPTANCE-002` r002；
- Policy manifest SHA-256：
  `c8a370406594da4b7f5f6bfa1dbc1bcfb9a775bf989521291fdfde143f5e4815`；
- branch：`agent/tr-w00-acceptance-fixes-002`；
- freeze 后才允许修改 build、credential Gate、Codex read-deny、入口测试、
  root README 和 systemd hardening 范围。
