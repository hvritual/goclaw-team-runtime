# MVP-W00 执行 Journal

本文件只追加。计划内容位于 `plan-r001.md`。

## 状态事件

| Seq | 时间 | Actor | From | To | 原因 | Evidence |
|---:|---|---|---|---|---|---|
| 1 | 2026-07-28 | user | `planned` | `active` | 批准先执行源码恢复和 MVP | 用户指令 |

## 进度事件

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S01` | `planned → complete` | 四个归档 SHA 通过；源码归档与解压树一致；新 import commit 为 `e4783a4f2bc7a6ce8df1405787c44ed636b195d3` | 提交 Wave 激活文档 |
| 2026-07-28 | `MVP-W00-S02` | `planned → active` | Node `v24.14.0`、npm `11.9.0` 可用；当前环境缺少 Go | 冻结工具链并补齐 Go Gate |
| 2026-07-28 | `MVP-W00-S02` | `active → complete` | 官方 Go `1.25.5` 归档校验通过并在临时目录安装；工具链已冻结 | 重放测试 |
| 2026-07-28 | `MVP-W00-S03` | `planned → complete` | Go 全包、关键包 race/vet、Web 8/8、Obsidian 6/6 及两项构建通过 | 重放发布 |
| 2026-07-28 | `MVP-W00-S04` | `planned → complete` | 双架构 Runner、跨平台控制端、归档恢复扫描及 SHA 校验通过；tracked bundle 无差异 | 独立复核 |
| 2026-07-28 | `MVP-W00-S05` | `planned → active` | 三路只读复核已启动 | 汇总 findings，决定标签 |
| 2026-07-28 | `MVP-W00-S05` | `active → blocked` | 三路复核均 BLOCK；P0=0，去重后 8 个 P1 | 激活 `plan-r002` 修复 |

## 2026-07-28 — r002 恢复发布修订

- 第一轮确定性 Gate 保持通过，不把审查失败改写成测试失败；
- 独立审查证明 import 内容正确，但来源命令、发布原子性/可重现性、归档回读、
  工具链/组件身份和治理投影不足以放行恢复标签；
- 创建 `plan-r002.md`，只扩展恢复脚本、package 工具链元数据和文档范围；
- 运行时产品代码、真实凭据和实机 Pilot 仍不在允许范围内。

## Evidence ledger

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `MVP-EVID-001A` | 2026-07-28 | `MVP-W00-S01` | source archive | `cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f` | 发布源码归档完整 | `pass` | Codex root agent | pending |
| `MVP-EVID-002` | 2026-07-28 | `MVP-W00-S03` | Go test/race/vet | 见 `docs/recovery/RECOVERY_GATE_REPORT.md` | 确定性 Go Gate 通过 | `pass` | Codex root agent | reviewing |
| `MVP-EVID-003` | 2026-07-28 | `MVP-W00-S03` | Web test/build | 见 `docs/recovery/RECOVERY_GATE_REPORT.md` | Web 8/8 与 build 通过 | `pass` | Codex root agent | reviewing |
| `MVP-EVID-004` | 2026-07-28 | `MVP-W00-S03` | Obsidian test/build | 见 `docs/recovery/RECOVERY_GATE_REPORT.md` | Adapter 6/6 与 build 通过 | `pass` | Codex root agent | reviewing |
| `MVP-EVID-005` | 2026-07-28 | `MVP-W00-S04` | release archives | `cb02be08f065274855ee6a0b9935a567b3a70b6aca23ea58be7f304047b26e7e` | 跨平台构建和安全扫描通过 | `pass` | Codex root agent | reviewing |

## 终态声明

尚未进入终态。
