# GoClaw Recovery 独立复核记录

日期：2026-07-28  
对象：`v0.8.0-pilot.1-import..working-tree`  
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
