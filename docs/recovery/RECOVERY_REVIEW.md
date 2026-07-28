# GoClaw Recovery 独立复核记录

日期：2026-07-28

审查对象：

| Round | Git range / commit | Tree | 说明 |
|---|---|---|---|
| 1 | import 后、r002 前历史 working tree | 不可精确复算 | 此 locator 缺陷触发 r002，保留为失败事实 |
| 2 | `6fa9607f97715660271ea1356797d4dffaf78f62` | `f3d30badaeb83c91622b4585d73923f6e787f0fb` | clean reviewed HEAD |
| 2 implementation | `792b599c56852e26623bca83313f56b3a0693f2b` | `6d7da8ee1d4c9f69fc4cc2aea8fade9b816e9a6c` | 双构建候选 |
| r003 final | `21bdfc5d9852143b10102ff3a804033dd29904a8` | `854262bd79e9e73ff8d3d9de6d96d94d068ac1e0` | exact clean reviewed HEAD；触发 r004 |

当前结论：`BLOCK`

## 审查角色与工具策略

| Reviewer | 范围 | 工具策略 | 编辑权限 |
|---|---|---|---|
| `recovery_code_review` | 来源、Git、范围、可重放性 | 本地只读文件、Git、校验命令 | 禁止 |
| `recovery_security_review` | 归档、凭据、供应链、发布原子性 | 本地只读扫描、构建脚本、lockfile | 禁止 |
| `recovery_docs_review` | Wave、Evidence、Decision、承诺一致性 | 本地只读文档和 Registry | 禁止 |

三名 reviewer 使用彼此独立的审查上下文。审查提示均要求：

- 不修改任何文件；
- findings 按严重性排列并给出文件/行号；
- 不把历史报告替代为本次验证；
- 明确 `pass` 或 `block`；
- 没有证据时不得推断通过。

## 第一轮结果

| Reviewer | P0 | P1 | 结论 |
|---|---:|---:|---|
| code/source | 0 | 2 | `BLOCK` |
| security/supply-chain | 0 | 5 | `BLOCK` |
| docs/governance | 0 | 4 | `BLOCK` |

去重后的阻断项登记在 `MVP-W00 plan-r002` 的 `REC-P1-001` 到
`REC-P1-008`。正向确认：

- import 的 611 个文件名、内容、执行位与原始 source archive 一致；
- import tree 为 `38f798c2a652eaf99d5ad1ca145e50c176ee4c58`；
- 本轮恢复变更未修改产品运行时代码；
- 当前 tracked 源码未发现真实 Token、私钥或 credential assignment；
- Go module 校验和 npm lockfile integrity 可验证；
- 未把 Codex OAuth、飞书、浏览器、bwrap、WSL2、Lima 或三台实机写成通过。

## 未关闭 P2

- import tag 和 checksum 尚未签名；
- GitHub Actions/Docker action/image 尚未全部固定 digest；
- SBOM、漏洞报告和外部 WORM Evidence 尚未实现；
- 历史 credential-shaped material 仍需 owner 证明已撤销、轮换或从未有效。

这些 P2 不允许在 MVP 实机前被遗忘，但不通过扩大 Recovery 运行时范围顺带
实现。

## 第二轮

状态：`plan-r002` 的 S05A–S05D 已实施，候选 commit
`792b599c56852e26623bca83313f56b3a0693f2b` 待三路只读复核。第一轮
`BLOCK` 在第二轮结果写入前保持有效，不能提前创建 recovered tag。

## 第二轮结果

| Reviewer | P0 | P1 | 结论 |
|---|---:|---:|---|
| `recovery_code_review` | 0 | 0 | `PASS` |
| `recovery_security_review` | 0 | 0 | `PASS` |
| `recovery_docs_review` | 0 | 3 | `BLOCK` |

docs/governance 的三个 P1 登记为 `MVP-ISSUE-001`：current projection
冲突、BLOCK reviewer 被误记为批准者、r002 实现缺完整 frozen Task tuple。
`MVP-W00 r003` 采用 forward-only 修复，不改写 r002 历史；最终三路复核
仍未执行。

## r003 最终复核结果

| Reviewer | P0 | P1 | 结论 |
|---|---:|---:|---|
| `recovery_code_review` | 0 | 1 | `BLOCK` |
| `recovery_security_review` | 0 | 0 | `PASS` |
| `recovery_docs_review` | 0 | 1 | `BLOCK` |

code P1：r003 Task 的 base 不含 active r003 Plan/Registry/Policy，形成
self-authorizing freeze。docs P1：FE-W00、FE-W01、PILOT-W00、MVP-W00
Journal 的冻结历史字节被改写或插入，FE-W01 独立冻结 SHA 失效。

两项分别登记为 `MVP-ISSUE-001` 和 `MVP-ISSUE-002`。r004 采用
activation → freeze → implementation 三提交前向修复，并恢复 Journal
历史前缀；不 amend/rebase r003 或更早历史。

## r004 最终复核

状态：S07C 在 final review 前发现 frozen SHA acceptance 抄写错误并失败
关闭；登记 `MVP-ISSUE-003`。r004 不执行 S07D/S07E，也不提交 final
review；后继 revision 修正常量并重新冻结前当前 `BLOCK` 不变。
