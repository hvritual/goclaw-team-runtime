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

## Evidence ledger

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `MVP-EVID-001A` | 2026-07-28 | `MVP-W00-S01` | source archive | `cf327169e7654d2284c98482e4d885085ed6068152f5ae9cbd103ea5ffd78c8f` | 发布源码归档完整 | `pass` | Codex root agent | pending |

## 终态声明

尚未进入终态。

## 2026-07-28 — r001 后续门禁

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
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

### Evidence ledger 追加（首次全量门禁）

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
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

## 2026-07-28 — r002 第二轮复核与 r003 激活

- code/source：`PASS`，P0=0、P1=0；
- security/supply-chain：`PASS`，P0=0、P1=0；
- docs/governance：`BLOCK`，P0=0、P1=3；
- 三个治理 P1 稳定登记为 `MVP-ISSUE-001`，不修改 r002 或历史 commit；
- 新建 `plan-r003`，`approved_by` 只保留用户授权；
- 从 exact base `6fa9607f97715660271ea1356797d4dffaf78f62` 冻结
  `MVP-W00-RECOVERY-003 r003`，包含 Repository、Issue、policy hash、
  acceptance 和 verification；
- r002 候选只作为历史实现证据，必须在 r003 Task 下完整重验。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S05E` | `active → blocked` | docs review P1=3；禁止 tag | 激活 r003 |
| 2026-07-28 | `MVP-W00-S06A` | `planned → complete` | Issue、Task、Repository/base、policy bundle 和 r003 Registry 已冻结 | 修正 projection/approval/ID |

## 2026-07-28 — S06B current projection 与批准语义

- 新建 FE-W00 r005、FE-W01 r011、PILOT-W00 r006；当前 revisions 的
  `approved_by` 只包含用户授权，BLOCK reviewer 只保留为 reviewer/finding；
- 总 README、Track 和 Registry 全部指向 r003/r005/r011/r006；
- MVP r003 为 docs-only、`product_code_changes_allowed=false`，r002 的
  发布工具授权只作为不可变历史保留；
- Wave/Evidence/Issue ID 规则泛化到 FE、PILOT、MVP；
- MVP Evidence reviewer 改为稳定 agent identity + round；
- Review round 2 精确绑定 commit/tree，旧 working-tree locator 明确记录为
  第一轮缺陷，不再作为当前权威对象。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S06B` | `planned → complete` | 三个 docs P1 的实现已完成，`MVP-ISSUE-001 fixed` | 完整重验与最终独立复核 |

## 2026-07-28 — S06C r003 完整重验

- Policy manifest 自校验通过；
- 固定 import tree 重放为 611/611，内容/执行位/extra 全为 0；
- archive negative suite 全部 fail closed；
- Go 全包、六关键包 race/vet 全部通过；
- Web 8/8、Obsidian 6/6 及两项 production build 通过；
- commit `3c209a411333d31bdac44896a40d256bde33e3b0` 连续 release 构建两次，
  第二次整目录 identical；
- manifest commit/tree 与该候选完全一致，工作树无产品 diff。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S06C` | `planned → review` | frozen Task 下全量 Gate 通过；最终三路复核待执行 | exact HEAD code/security/docs review |

### Evidence ledger 追加

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `MVP-EVID-005B` | 2026-07-28 | `MVP-W00-S06C` / `MVP-ISSUE-001` / `MVP-W00-RECOVERY-003 r003` | r003 release manifest | `16ea5c12c1bdc9334f3eef8ee444148e50cd4aabade11a1deb60c8adcfe81965` | exact Task/Policy 下全量重验与双构建一致 | `pass` | Codex root agent | final review pending |

## 2026-07-28 — r003 final BLOCK 与 r004 前向修复

- r003 exact review target：
  `21bdfc5d9852143b10102ff3a804033dd29904a8` /
  `854262bd79e9e73ff8d3d9de6d96d94d068ac1e0`；
- code：`BLOCK`，P0=0/P1=1；security：`PASS`，P0=0/P1=0；
  docs：`BLOCK`，P0=0/P1=1；
- `MVP-ISSUE-001` 扩展记录 self-authorizing freeze，新增
  `MVP-ISSUE-002` 记录 Journal 历史完整性；
- r004 activation `91d47a8` 已包含 active Plan/Registry/Policy，
  后继 commit `7c78f52` 才冻结 `MVP-W00-RECOVERY-004`；
- FE-W00、FE-W01、PILOT-W00 和本 Journal 的冻结前缀均已恢复；本文件
  首次提交的前 1177 bytes SHA-256 为
  `3ef4b2fbfcf7d5926300c03c35943d190eaf07a2216e9f25bde44dbc1805e709`。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S07A` | `planned → complete` | activation base 含 r004 Plan/Registry/Policy | 从 base 冻结 Task |
| 2026-07-28 | `MVP-W00-S07B` | `planned → complete` | Task base commit/tree/policy 可复算 | 恢复 Journal |
| 2026-07-28 | `MVP-W00-S07C` | `planned → blocked` | 四个历史前缀已恢复，但 r004 Plan/Task 把权威 FE-W01 SHA `33a50e1bbd...` 误抄为 `33a50e8f3a...` | 登记 `MVP-ISSUE-003`，创建后继 revision |
| 2026-07-28 | `MVP-W00-S07D` | `planned → blocked` | 冻结 acceptance 不可满足，未执行全量 Gate 或 final review | 修正验收常量并重新冻结 |

## 2026-07-28 — r005 正确常量与 Task 冻结

- r005 将 FE-W01 26641-byte SHA 修正为 r009 Plan、R7 Evidence、import tree
  与当前文件共同复算的
  `33a50e1bbd028ca06adcee3e18df0ea62f405ff72a6e982b318720c11bccf997`；
- activation `df8fe9f` 已包含 active r005 Plan/Registry/Policy；
- 后继 commit `96de00a` 冻结 `MVP-W00-RECOVERY-005`，base commit/tree 与
  Policy manifest 均可复算；
- 五个冻结前缀长度/SHA 全部通过，current projection 唯一切换到 r005。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S08A` | `planned → complete` | r005 activation 只包含 Plan/Registry/Issue/Policy | 从 activation base 冻结 |
| 2026-07-28 | `MVP-W00-S08B` | `planned → complete` | exact base 含 active r005 Plan/Registry/Policy | 复算投影与前缀 |
| 2026-07-28 | `MVP-W00-S08C` | `planned → complete` | 五个冻结前缀和 current projection 一致 | 全量确定性 Gate |
| 2026-07-28 | `MVP-W00-S08D` | `planned → active` | Gate 与 final review 尚未执行 | exact clean HEAD revalidation |

## 2026-07-28 — S08D r005 全量重验

- exact candidate：`e262b8c3be6a42d3b86e13fe48b34c055dccb9db` /
  `bdf8c76ca2fc0b992e4a9d403c0ae1a0cbcf1b78`；
- Policy、来源 611/611、archive negative、Go 全包/race/vet、Web 8/8、
  Obsidian 6/6、两项 production build 与 tracked bundle diff 全部通过；
- release 首次原子发布，第二次明确为
  `Verified identical existing release`；
- manifest SHA-256：
  `0379cf736ac1fb6a3770be39ccb8156c877a8dfd1ec50777c90c2b7896fdc2b8`；
- final review 与 review 结论写回后的最终双构建/tag 尚未执行。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S08D` | `active → review` | frozen Task 下全量 Gate 与双构建通过 | exact evidence HEAD 三路复核 |

### Evidence ledger 追加

| Evidence ID | 时间 | Step/Issue/Task | Artifact/Trace | SHA-256 | 声明 | 结果 | 生成者 | 复核者 |
|---|---|---|---|---|---|---|---|---|
| `MVP-EVID-005C` | 2026-07-28 | `MVP-W00-S08D` / `MVP-ISSUE-001`–`003` / `MVP-W00-RECOVERY-005 r005` | r005 candidate release manifest | `0379cf736ac1fb6a3770be39ccb8156c877a8dfd1ec50777c90c2b7896fdc2b8` | 正确 SHA authority、完整 Gate 与同 commit 双构建一致 | `pass` | Codex root agent | final review pending |

## 2026-07-28 — r005 三路最终复核

- review target：`526e14b7214403d3b1bfbc9a660abf960a364a4e` /
  `524c764eb8203441cfb9c50e4219861b126275bd`，worktree clean；
- code/source：`PASS`，P0=0/P1=0，self-authorizing freeze P1 已关闭；
- security/supply-chain：`PASS`，P0=0/P1=0，保留 6 个非阻断 P2；
- docs/governance：`PASS`，P0=0/P1=0/P2=0，Journal P1 与 Gate report P2
  已关闭；
- `MVP-ISSUE-001`–`003` 独立验证完成；Recovery 仍等待 review 写回 commit
  上的最终双构建、manifest/tag 对齐。

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-28 | `MVP-W00-S08D` | `review → complete` | 三路 exact target 复核 P0=0/P1=0 | 写回复核结论 |
| 2026-07-28 | `MVP-W00-S08E` | `planned → active` | review 通过；最终双构建/tag 尚未执行 | 从 review-writeback clean commit 构建两次 |
