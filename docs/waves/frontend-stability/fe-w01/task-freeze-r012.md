# FE-W01 MVP Browser Task Freeze r012

状态：`frozen`

| 字段 | 冻结值 |
|---|---|
| Task-ID | `FE-W01-MVP-BROWSER-012` |
| Task-Revision | `r012` |
| Work-Item | `recovered-base-browser-credential-closure` |
| Project-ID | `goclaw-team-runtime` |
| Repository-ID | `goclaw-team-runtime-recovered` |
| Repository | local Git repository on branch `recovery/0.8.0-pilot.1` |
| Assignee | `Codex root agent` |
| Recovered source tag | `v0.8.0-pilot.1-recovered.1` |
| Base commit | `683a008caf3c99642f5ec32a71443c63092f7a4c` |
| Base tree | `b02027fe379cd8bc48e870af09f9568f954f4f61` |
| Issues | `FE-ISSUE-003`, `004`, `005`, `007`, `010` |
| Wave-ID | `FE-W01` |
| Wave revision | `r012` |
| Steps | `FE-W01-S13B`–`FE-W01-S13E` |
| Policy-Bundle | `FE-W01-R012-POLICY` |
| Policy manifest | `docs/waves/frontend-stability/fe-w01/POLICY_BUNDLE_SHA256SUMS-r012.txt` |
| Policy manifest SHA-256 | `0d1645ea6c8c6b347ce1514ad5962ee9cd289840f92e4f008d4d0cfa0b4c0595` |
| Frozen at | `2026-07-28` |

## Base 可解析性

从 exact base `683a008` 必须能读取：

- Registry `active_wave=FE-W01`、`current_release=0.8.0-pilot.1-recovered.1`；
- current document `frontend-stability/fe-w01/plan-r012.md`；
- r012 Policy manifest 及其 AGENTS/Plan/Registry 三项 SHA；
- Recovery `MVP-W00 r006 complete`。

任一项不可解析即禁止启动 Gateway、Vite、Browser 或 Playwright。

## 授权范围

继承 `plan-r012.md` 的 exact allowlist。预检、测试与证据阶段不得修改产品
代码；只有新 Issue 具备可重复环境、期望/实际、脱敏日志并先创建后继 Plan
revision，才允许实施修复。

本 Task 不授权真实 provider/API key、Codex OAuth、飞书凭据、生产项目、
自动 commit/push/PR/merge/release 或跨项目数据。

## Acceptance criteria

1. Task base、Plan、Registry、Policy 和 recovered tag 可复算。
2. Go 1.25.5、Node 24.14.0、npm 11.9.0、clean tree 通过。
3. UI test/build 与 Gateway/session/agent race 通过。
4. Browser plugin 优先路径有明确结果；fallback 仅按 r012 合同执行。
5. Desktop `1440×1000`、Mobile `390×844`、三个 BrowserContext 的目标 flow
   有 DOM/Console/交互/脱敏截图证据。
6. ptrace `/bin/true` capability 和 Gateway syscall 零出站通过。
7. credential owner closure 通过。
8. code/security/docs final review P0=0/P1=0。
9. 未泄露 credential，未修改无复现证据的产品代码。

## Deterministic verification

```bash
sha256sum -c \
  docs/waves/frontend-stability/fe-w01/POLICY_BUNDLE_SHA256SUMS-r012.txt
(cd ui && npm ci && npm test && npm run build)
go test -race -count=1 ./gateway ./session ./agent
head -c 26641 docs/waves/frontend-stability/fe-w01/journal.md | sha256sum
git diff --check
git status --short
```

Browser/runtime 命令必须在 S13B 记录 Browser availability、绑定 host、
synthetic fixture 和清理计划后确定；不得在冻结文件中伪造未来动态端口或
凭据路径。

## 独立验收与回滚

- reviewers：`recovery_code_review`、`recovery_security_review`、
  `recovery_docs_review`；
- assignee 不得替代独立 final；
- Browser、ptrace、owner 任一失败时，输出 blocked Evidence，FE-W01 不得
  complete；
- 回滚只清理本次 synthetic runtime/临时 profile，不删除历史 Evidence，
  不移动 recovered tag。
