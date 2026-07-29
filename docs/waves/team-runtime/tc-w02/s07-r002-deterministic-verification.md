# TC-W02 S07 r002 Deterministic Verification

状态：`passed`

## Exact tuple

| 字段 | 值 |
|---|---|
| Frozen base | `d2df5fd59fffb2d38838364614974ebce62bbda1` |
| Base tree | `af6d487039df7086daa4224d341bfdbb9422772d` |
| Remediation candidate | `df47a8b4719185db40215c9fe028c665897cdf34` |
| Candidate tree | `284a60fdccbca9273ad140132ad1fce6a77e7264` |
| Task | `TC-W02-REPLAN-001` r002 |
| Wave/Step | `TC-W02 r002` / `TC-W02-S07` |
| Policy manifest SHA-256 | `04b76061b2dd61fe995568e1c8646f5eb5d7684cd0f3fcb12c2f7eee4033536d` |

## Results

| Gate | 结果 |
|---|---|
| `sha256sum -c ...r002.txt` | 3/3 OK |
| base→candidate validator | PASS；18 Waves；唯一 active TC-W02；10 changed files |
| changed scope | 全部 `docs/waves/**` |
| product code diff | empty |
| Registry JSON/state/dependencies | PASS |
| RN/INT supersession trace | PASS |
| proposed TC-W03–W06 product-code false/template sections | PASS |
| Policy/Knowledge/Context/MCP/Evidence required contracts | PASS |
| current RPC/command mapping | PASS |
| Markdown links | PASS |
| journal append-only | PASS |
| `git diff --check` | PASS |
| working tree at exact run | clean |

执行命令：

```bash
node docs/waves/validate-wave-docs.mjs \
  --base d2df5fd59fffb2d38838364614974ebce62bbda1 \
  --candidate df47a8b4719185db40215c9fe028c665897cdf34
sha256sum -c docs/waves/team-runtime/tc-w02/POLICY_BUNDLE_SHA256SUMS-r002.txt
git diff --check d2df5fd59fffb2d38838364614974ebce62bbda1...df47a8b4719185db40215c9fe028c665897cdf34
```

本 Evidence 证明文档候选和治理结构通过确定性 Gate；不证明产品实现、真实
数据迁移或运行环境。包含本 Evidence 的 final review candidate 仍需用同一
base→candidate validator 重验，并由独立 reviewer 验收。
