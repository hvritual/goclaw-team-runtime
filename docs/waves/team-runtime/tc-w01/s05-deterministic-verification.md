# TC-W01 r002 Deterministic Verification

Evidence ID：`TC-EVID-W01-001`

状态：`collecting`（确定性 Gate 通过；exact-commit independent review 待执行）

## Frozen authority

- Task：`TC-W01-CONTROL-003` r002；
- Base：`c29dabdee2f0551ad57996611e61e40521e3f7ee`；
- Base tree：`9f08542676dec3f4680107613027392974dd3d50`；
- Policy manifest SHA-256：
  `0950b7ee6a2a15a0d8b7093cb4f7dbb513a1691c7c4f38755710a91a25d52b2b`；
- Issue：`TC-ISSUE-001`。

## Implemented contracts

- project/member Token Budget、hard limit 与幂等 usage ledger；
- Knowledge Source、Skill Release、Runner Release Registry；
- `goclaw-context/v1` canonical Context Bundle；
- project RBAC、server-resolved resource scope 和旧 state map 初始化；
- Gateway RPC、CLI 管理入口和 Team Web Console central summary；
- 操作与迁移文档。

## Deterministic results

| Gate | 结果 |
|---|---|
| Policy manifest | PASS |
| `go test -count=1 ./teamcontrol ./gateway ./cli` | PASS |
| `go test -race -count=1 ./teamcontrol ./gateway` | PASS |
| `go test -count=1 ./...` | PASS |
| `go vet ./...` | PASS |
| UI tests | 9/9 PASS |
| UI production build | PASS |
| `git diff --check` | PASS |

关键负例：

- 相同 usage event/payload 不重复计费，不同 payload 复用 ID 冲突；
- 超 hard limit、降低 limit 到 used 以下及超全局安全上限失败；
- 并发 usage 经 atomic single-writer 累计为精确值；
- viewer mutation、developer compile 和跨项目 Registry 引用拒绝；
- draft Knowledge/Skill、缺 checksum、budget/user 不匹配失败；
- 旧 state 缺少六个新 map 时无损加载并初始化为空；
- 相同 Context 输入 ID/hash 稳定，预算 snapshot 改变生成新 hash。

## Security boundary

Registry 只保存 URI、revision/version、SHA-256、状态与 metadata；本 Wave
不 fetch URI、不下载 Runner artifact、不读取 Vault 正文，也不接收 Codex
OAuth、device key 或个人 Token 原值。Context Bundle 是中央元数据合同，
Runner 注入留给 `INT-W01`。

## Pending

实现提交推送 GitHub 后，独立 code/security/docs agents 必须绑定远端 exact
commit 完成 P0/P1 review。未通过前本 Evidence 保持 collecting，RN-W01
不得激活。
