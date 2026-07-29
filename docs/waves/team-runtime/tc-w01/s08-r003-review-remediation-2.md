# TC-W01 S08 r003 Review Remediation 2

Evidence ID：`TC-EVID-W01-005`

状态：`collecting`（确定性 Gate 通过；等待新 exact SHA 三路复核）

本轮前向修复 `TC-EVID-W01-004` 的全部 P1，不修改已失败的
`86f6823544...`。

## 修复与回归

| Finding | 修复 | Evidence |
|---|---|---|
| approved 可降级 draft 后删除 | 显式 Registry transition matrix；approved 只能转 disabled | Knowledge/Skill/Runner downgrade 与 approved-delete 负例 |
| legacy budget 超 JS-safe | normalize/load 聚合校验；Gateway 二次防御 | legacy overflow load 失败 |
| metadata canonical collision | 规范化重复 key 直接拒绝 | usage 与 Registry 大小写 collision 负例 |
| legacy usage metadata RPC 泄露 | list 前校验 event/budget/typed metadata | core 与真实 RPC fail-closed |
| UNC/伪 drive URI | 拒绝 scheme-relative、UNC、device、`x://` | 四类跨平台 URI 负例 |
| migration Evidence 缺口 | 六类 map success、collision/source unchanged、持久化复核 | reopen 与 on-disk composite key |
| CRUD Evidence 缺口 | 三 Registry get/delete/approved guard/RBAC/wrong-project/reopen | Service + Gateway tests |
| JSON 合同不完整 | 补齐请求文件、get/list/delete、safe presenter、ContextBundle | `TEAM_CONTROL_REGISTRY_CN.md` |

同时关闭初审 P2 中的：

- 重复 Context compile 自动检测无状态变化，不修改 revision/file；
- normalized state 成功后 atomic 持久化，冲突不覆盖源文件；
- Policy/Context canonical hash 与 Context ID 在读取前复算；
- Unix existing state 必须 regular file 且权限不宽于 `0600`。

## 确定性 Gate

环境：Go `1.25.5`，Node `24.14.0`，npm `11.9.0`。

| Gate | 结果 |
|---|---|
| Policy manifest | `3/3 OK` |
| `go test -count=1 ./teamcontrol ./gateway ./cli` | passed |
| `go test -count=1 ./...` | passed |
| `go vet ./...` | passed |
| `go test -race -count=1 ./teamcontrol ./gateway` | passed |
| UI test | `10/10 passed` |
| UI production build | passed |
| `git diff --check` | passed |

测试只使用 synthetic placeholder；没有真实 token、OAuth、device key 或
外部 secret。

## 待完成

推送本 Evidence 所在的新 exact commit 后，code/security/docs 三名独立
reviewer 必须重新审查同一 SHA。只有 P0=0/P1=0 才能关闭 TC-W01。
