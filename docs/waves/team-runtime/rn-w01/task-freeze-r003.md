# RN-W01 Trusted Profile Selection Task Freeze r003

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `RN-W01-LIFECYCLE-001` |
| Task-Revision | `r003` |
| Work-Item | `runner-dual-profile-lifecycle` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/rn-w01-lifecycle-001` |
| Base commit | `ebced0ac55176e67a9ae28351c43255d170bab86` |
| Base tree | `bfa32ace1e6ec1af3c8f7aa006b21524842c73b6` |
| Issue | `RN-ISSUE-001` |
| Wave-ID | `RN-W01` |
| Wave revision | `r003` |
| Steps | `RN-W01-S03B`–`RN-W01-S08` |
| Policy-Bundle | `RN-W01-R003-POLICY` |
| Policy manifest | `docs/waves/team-runtime/rn-w01/POLICY_BUNDLE_SHA256SUMS-r003.txt` |
| Policy manifest SHA-256 | `5d06456cd8357c8050ff7972f96c721c4f649e166c7b0a025e85dfcf67d65490` |
| Frozen at | `2026-07-29` |

## Exact scope

本 Freeze 采用 `plan-r003.md` 的完整 `allowed_change_scope`。相对 r002 只新增
受信 Team enqueue 与 CLI 选择路径；双 profile、更新、并发、三平台和安全
合同不变。

## Acceptance

1. `ExecutionPack.execution_profile` 缺省严格规范为 `strict`；
2. delegated 只有 resolved Team policy allow 且 Runner 显式 capability
   匹配时可 claim；
3. client execution pack、capability 或 metadata 不能绕过服务端选择；
4. 本机 executor profile 必须与签名 ExecutionPack 完全相同；
5. r002 全部 Acceptance、deterministic verification 与 review Gate 继续适用；
6. 所有后继 commit 使用 r003 tuple 与 Policy hash。

## Deterministic verification

沿用 r002 Freeze，并增加：

```bash
go test -count=1 ./gateway ./cli ./workstation
go test -race -count=1 ./gateway ./workstation
```

## Rollback

任何 policy 缺失、JSON 非法、profile 未知或 capability 不匹配都拒绝；
不允许把失败解释为自动回退 strict。
