# TC-W01 S08 r003 Independent Review 1

Evidence ID：`TC-EVID-W01-004`

状态：`failed`

Reviewed exact remote commit：
`86f6823544c72c4c8f070fe851ac3d4cd4e5f1b3`

Reviewed tree：
`b2d1675c1c98927e311d30e7255f90cc40ce3e0a`

三名 reviewer 均为只读独立 agent；未修改、提交或推送文件。

| Reviewer | P0 | P1 | P2 | 结论 |
|---|---:|---:|---:|---|
| code | 0 | 4 | 2 | BLOCK |
| security | 0 | 2 | 3 | BLOCK |
| docs/governance | 0 | 2 | 1 | BLOCK |

## Code findings

1. Registry 允许 `approved → draft → delete`，绕过必须先 disabled；
2. legacy 项目预算总量可超过 JavaScript safe integer；
3. metadata 大小写 canonical key collision 结果依赖 map 遍历顺序；
4. 六类裸 key migration 与三类 Registry CRUD/RBAC/persistence Evidence
   矩阵不完整。

P2：重复 Context compile 仍产生 revision/file 漂移；裸 key 迁移只在内存
发生。

## Security findings

1. legacy TokenUsageEvent 任意 metadata 未在 list 前复验，会经 RPC 回显；
2. `//host/share`、UNC、device path、`x://host/path` 可绕过本地路径快捷
   判断。

P2：metadata canonical collision；stored Policy/Context hash 未复算；existing
state 文件权限未检查。

## Docs/governance findings

1. 六类裸 key migration 与 collision 负例 Evidence 不足；
2. get/list/delete 请求、presenter-safe 响应和完整 ContextBundle JSON
   合同不完整。

P2：Issue Register 未链接 collecting Evidence。

## 处理决定

- 保留本次 exact review 为失败证据，不修改历史；
- 所有 P1 均属于 r003 Freeze 已授权的隔离、secret-safe schema、CRUD、
  migration 和文档合同范围；
- 在同一 Task/Work-Item 内做前向修复并生成新的 exact SHA；
- 新 SHA 必须重新执行全部 Gate 和三路独立 review；本次 reviewer 结论不能
  自动沿用。
