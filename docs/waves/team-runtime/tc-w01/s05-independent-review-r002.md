# TC-W01 r002 Independent Review

Evidence ID：`TC-EVID-W01-002`

审阅绑定：

- Draft PR #3；
- base `c29dabdee2f0551ad57996611e61e40521e3f7ee`；
- exact implementation commit
  `6aab01f2508a4c162147da20b399172e01a83d7d`；
- exact tree `fcf1aee0f8d2c7b5ca9ab2d59e2b730d704360b8`。

| Reviewer | 结论 | P0 | P1 | P2 |
|---|---|---:|---:|---:|
| code | BLOCK | 0 | 2 | 3 |
| security | BLOCK | 0 | 2 | 0 |
| docs/governance | BLOCK | 0 | 3 | 1 |

## Blocking findings

1. Budget/Registry/usage 使用全局裸 ID map，Put 先按请求 project 授权再查
   existing，形成跨项目 existence oracle 和 ID 抢占；
2. project-level budget 覆盖掉显式 target user，不同成员可能得到同一
   Context hash/ID；
3. Registry URI 允许 userinfo/query/fragment/任意 scheme，自由 metadata
   与任意 Policy JSON 可把 secret 持久化并返回；
4. Knowledge/Skill/Runner release 只实现 put/list，未满足冻结的 CRUD/list；
5. UI test 只正则检查源码，没有执行 loading/empty/denied/error projection；
6. r002 implementation commit 的 Work-Item/Wave-Step trailer 与冻结 tuple
   不一致，不能作为最终 traceable acceptance head。

## Non-blocking findings carried into remediation

- usage event replay 不重复计费但仍递增 state revision 并重写文件；
- 多预算 int64 汇总可能超过 JavaScript safe integer；
- 操作文档引用 JSON 文件但样例 schema 不完整；
- GitHub 没有 status checks；最终仍需仓库 deterministic Evidence。

## Result

`TC-EVID-W01-001` 保持 collecting，`TC-ISSUE-001` 回到 fixing。r002
implementation 保留为失败 Evidence，不 amend/rebase。创建 r003 做前向
修复；任一 P1 未关闭前 RN-W01 不激活。
