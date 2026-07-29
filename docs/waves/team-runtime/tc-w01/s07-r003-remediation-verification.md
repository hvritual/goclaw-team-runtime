# TC-W01 S07 r003 Remediation Verification

Evidence ID：`TC-EVID-W01-003`

状态：`collecting`（确定性 Gate 通过；等待 exact-commit 三路独立验收）

## 冻结身份

| 字段 | 值 |
|---|---|
| Task-ID | `TC-W01-ACCEPTANCE-004` |
| Task-Revision | `r003` |
| Work-Item | `tc-w01-acceptance-remediation` |
| Wave-Step | `TC-W01-S07`–`TC-W01-S08` |
| Base commit | `6cc3334c68cb6b45b4f43c688e0ac0e674e02a7f` |
| Freeze commit | `cd82015ca3b8a3ea2424739fcac2e4e6e550c469` |
| Policy manifest SHA-256 | `c9f6bb4e72166b9faf9511404e3ceca7f31b1aa2e8d73738581395756b7ec6c1` |

## 已关闭的 r002 blocker

- Budget、Usage Event、Knowledge、Skill、Runner Release、Context Bundle
  统一迁移并使用 `(project_id, id)` 复合存储 key；
- project-level budget 不再覆盖 Context `target_user_id`，目标成员进入
  canonical material；
- Registry URI、registry/usage metadata 与 Policy Rules 改为显式 schema，
  legacy unsafe state 在 read/resolve/compile 前失败关闭；
- Gateway Registry presenter 不返回 metadata；
- Knowledge、Skill、Runner Release 增加 get/delete，approved 必须先
  disabled；
- usage identical replay 对 revision、UpdatedAt 和文件字节为 no-op；
- 项目预算总量不超过 JavaScript safe integer；
- Team 页 control summary 独立呈现 loading、empty、denied、error、ready，
  五状态通过 Vite SSR 执行测试。

## 确定性 Gate

环境：Go `1.25.5`，Node `24.14.0`，npm `11.9.0`。

| Gate | 结果 |
|---|---|
| `sha256sum -c .../POLICY_BUNDLE_SHA256SUMS-r003.txt` | `3/3 OK` |
| `go test -count=1 ./teamcontrol ./gateway ./cli` | passed |
| `go test -count=1 ./...` | passed |
| `go vet ./...` | passed |
| `go test -race -count=1 ./teamcontrol ./gateway` | passed |
| `(cd ui && npm test)` | `10/10 passed` |
| `(cd ui && npm run build)` | passed |
| `git diff --check` | passed |

新增负例覆盖 URI userinfo/query/fragment/plain HTTP、未知 metadata/policy、
legacy unsafe read、跨项目同 ID、跨项目 get/delete、approved delete、
裸 key 迁移和持久化 reopen。测试只使用 synthetic 值，没有真实 token、
OAuth、device key 或外部 secret。

## 待完成

1. 推送本 Evidence 所在 exact implementation commit；
2. code/security/docs 三名独立 reviewer 只读复核同一 SHA；
3. 仅当三路均 P0=0/P1=0 时创建 final Evidence、关闭
   `TC-ISSUE-001` 并激活 `RN-W01`。
