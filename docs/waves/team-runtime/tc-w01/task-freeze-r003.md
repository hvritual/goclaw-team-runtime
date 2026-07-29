# TC-W01 Acceptance Remediation Task Freeze r003

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `TC-W01-ACCEPTANCE-004` |
| Task-Revision | `r003` |
| Work-Item | `tc-w01-acceptance-remediation` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | `hvritual/goclaw-team-runtime` |
| Assignee | `Codex root agent` |
| Branch | `agent/tc-w01-control-003` |
| Base commit | `6cc3334c68cb6b45b4f43c688e0ac0e674e02a7f` |
| Base tree | `b78c6374101d3f6fa41ff47e39673eb75a9378e2` |
| Issue | `TC-ISSUE-001` |
| Wave-ID | `TC-W01` |
| Wave revision | `r003` |
| Steps | `TC-W01-S07`–`TC-W01-S08` |
| Policy-Bundle | `TC-W01-R003-POLICY` |
| Policy manifest | `docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r003.txt` |
| Policy manifest SHA-256 | `c9f6bb4e72166b9faf9511404e3ceca7f31b1aa2e8d73738581395756b7ec6c1` |
| Prior failed Evidence | `TC-EVID-W01-002` |
| Frozen at | `2026-07-29` |

## Exact scope

```text
docs/**
teamcontrol/types.go
teamcontrol/inputs.go
teamcontrol/store.go
teamcontrol/validation.go
teamcontrol/policy.go
teamcontrol/policy_test.go
teamcontrol/queries.go
teamcontrol/controlplane.go
teamcontrol/controlplane_test.go
gateway/team_control.go
gateway/team_runtime_test.go
cli/team.go
ui/src/team/TeamPage.tsx
ui/src/team/types.ts
ui/src/team/control-summary-state.ts
ui/tests/control-summary.test.mjs
```

## Acceptance

1. 两项目相同 external ID 的 Budget/Usage/Registry 不冲突且不可互相探测；
2. project budget 不丢失 target user，Context identity 语义完整；
3. URI、metadata、Policy 使用显式安全 schema，Gateway 不回显 metadata；
4. Knowledge/Skill/Runner get/delete 和 approved-delete guard 完成；
5. usage replay 对 state/file 严格 no-op，预算总量 JS-safe；
6. UI loading/empty/denied/error/ready 由可执行测试覆盖；
7. r002 candidate key 迁移、跨项目、secret 与 CRUD 正负例通过；
8. 后继 commit trailers 精确使用本 Freeze 的 Work-Item 和 step；
9. code/security/docs final exact review P0=0/P1=0。

## Deterministic verification

```bash
sha256sum -c docs/waves/team-runtime/tc-w01/POLICY_BUNDLE_SHA256SUMS-r003.txt
go test -count=1 ./teamcontrol ./gateway ./cli
go test -race -count=1 ./teamcontrol ./gateway
go test -count=1 ./...
go vet ./...
(cd ui && npm test && npm run build)
git diff --check
git status --short
```

## Rollback

- migration 复合 key 冲突或 legacy unsafe state：加载失败，不写回；
- unsafe URI/metadata/policy：mutation/read/compile 失败，不做猜测性脱敏；
- approved delete：失败，要求先 disabled；
- 任一 P1 未关闭：TC 保持 active，RN 不激活。
