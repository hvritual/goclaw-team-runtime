# MVP-W00 执行 Journal

本文件只追加。当前计划由 Registry 指向 `plan-r002.md`；`plan-r001.md`
保留为已被替代的历史。

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
| `MVP-EVID-006` | 2026-07-28 | `MVP-W00-S05` | `docs/recovery/RECOVERY_REVIEW.md` | pending | 第一轮独立复核 | `failed` | three reviewers | r002 re-review pending |

## 证据元数据更正

第一轮登记曾把 Index 中的 003 合并为 Web+Obsidian、004 写成 Release、
005 写成 Review，违反 r001 的稳定定义。更正后的唯一映射为：

`001 provenance`、`002 Go`、`003 Web`、`004 Obsidian`、`005 Release`、
`006 Review`。`001A` 只是 `001` 的归档 SHA 子证据，不是新的 Gate。

## 2026-07-28 — S05A/S05B 治理与来源重放

- 新建 FE-W00 r004、FE-W01 r010、PILOT-W00 r005；
- Registry、当前 Plan、Track Index 的状态、依赖、scope 和 product flag
  已收敛，唯一 active 仍为 `MVP-W00`；
- Evidence 恢复为 001–006 稳定定义，Decision 补齐 supersedes、替代方案、
  影响、Evidence/Task 和 bootstrap 例外；
- 新增 `verify-source-import.sh`，固定比较 import tag，而不是可变工作树；
- 实际结果：611/611，内容差异 0、执行位差异 0、额外文件 0。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S05A` | `planned → complete` | Registry/Plan/Index/Evidence/Decision 投影一致 | 发布脚本修复 |
| 2026-07-28 | `MVP-W00-S05B` | `planned → complete` | 固定 import tree 来源验证通过 | 发布脚本修复 |

## 2026-07-28 — S05C/S05D 可重复恢复发布

- 发布改为全局互斥锁、版本隔离 stage 和版本目录原子 rename；
- 原始 `0.8.0-pilot.1` 输出名受保护，recovered runtime 使用
  `0.8.0-pilot.1-recovered.1`；
- tar 使用 commit epoch、稳定排序、固定 owner/group/mode 和 `gzip -n`；
- runtime、source、Obsidian 全部在解压前校验安全路径、类型、链接、重复
  成员和精确成员合同；
- Obsidian 保持组件版本 `0.6.0`，release manifest 显式映射 runtime；
- 负向自测拒绝 extra、duplicate、symlink、traversal；一次自测确实发现
  conditional function 的失败返回缺口，修复后四类样本全部 fail closed；
- commit `792b599c56852e26623bca83313f56b3a0693f2b` 连续构建两次，第二次
  整个版本目录与首次发布完全一致。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S05C` | `planned → complete` | 原子/规范化发布、精确 archive contract、工具链和组件映射通过 | 双构建 |
| 2026-07-28 | `MVP-W00-S05D` | `planned → complete` | 同 commit 两次构建逐字节一致；clean tree | 三路独立复核 |
| 2026-07-28 | `MVP-W00-S05E` | `planned → active` | P0/P1 清零复核尚未执行 | code/security/docs re-review |

### Evidence ledger 追加

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `MVP-EVID-005A` | 2026-07-28 | `MVP-W00-S05C/S05D` | recovered release manifest | `f2c4cfe786a0c562ed4d0e983002e10e653bcb1f77ecf3a4c2fbbf0e34253305` | r002 候选的原子发布、归档合同和双构建一致 | `pass` | Codex root agent | r002 re-review pending |

## 终态声明

尚未进入终态。
