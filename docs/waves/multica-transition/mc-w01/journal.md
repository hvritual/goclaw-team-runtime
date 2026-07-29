# MC-W01 Journal

本文件只追加。当前状态以 Registry 和 `plan-r001.md` 为准。

## 2026-07-29 — 用户确认 Multica 方案 A

- 用户先要求以 Multica 为主体融合 Team Control，随后确认先拆分并落地
  Workspace、Member、Project、Issue、Task、Skill 六域；
- 用户明确选择保留完整 Multica 上游及原生依赖，不裁剪成六域最小应用；
- 用户授权把当前 GoClaw 源码与实施计划保存在 backup 分支，并在验证可恢复
  后替换目标分支的整个 tracked tree；
- 本授权不包含改写/推送 main、远端、部署、真实数据或凭据访问；
- `TC-W02` 在 final independent acceptance 前转为 superseded，历史和
  Evidence 只读保留，不冒充 complete。

## 状态事件

| Seq | 时间 | Actor | From | To | 原因 | Evidence |
|---:|---|---|---|---|---|---|
| 1 | 2026-07-29 | user directive / Codex root | `proposed` | `active` | 用户确认 Multica-first G3 转换 | `MC-EVID-W01-001` planned |

## 进度事件

| 时间 | Step ID | 状态变化 | 实际结果/阻塞 | 下一动作 |
|---|---|---|---|---|
| 2026-07-29 | `MC-W01-S01` | `planned → active` | Plan/Registry/Policy activation 编制中 | deterministic docs Gate |

## 2026-07-29 — exact Task Freeze

- activation commit：
  `dfe8aa4a5d100d5225582c902dfe0f5537790e11`；
- activation tree：
  `697cf292bdf0a205a272f6e5dd0817389b6c808c`；
- Task：`MC-W01-BASELINE-001 r001`；
- Policy manifest SHA-256：
  `430ae2d96db863bc261a4544f7bfc30fada092cd7677442af1ee4f3bca8b1324`；
- target branch：`codex/multica-six-domain-baseline`；
- backup branch：`codex/backup-goclaw-pre-multica-20260729`；
- replacement 前必须通过 `BACKUP-VERIFIED`。

| 2026-07-29 | `MC-W01-S01` | `active → complete` | activation exact commit 已冻结 | 创建 backup |
| 2026-07-29 | `MC-W01-S02` | `planned → active` | Task frozen | BACKUP-VERIFIED |

## 2026-07-29 — transition validator split

- 历史 `validate-wave-docs.mjs` 明确检查 `TC-W02 must be the active Wave`，
  因此对新路线返回预期 FAIL，不将其改写为 MC-W01 Evidence；
- 新增 `validate-multica-transition.mjs`，只验证 MC-W01 唯一 active、
  TC-W02 superseded、approved Plan/Policy、exact freeze tuple、非空
  base→candidate docs scope 和 `git diff --check`；
- 旧验证器与失败输出保留为旧路线历史，新验证器不得为产品 tree 或六域
  功能验收背书。

## 2026-07-29 — BACKUP-VERIFIED PASS

- backup ref：`codex/backup-goclaw-pre-multica-20260729`；
- verified commit：
  `d6211a45bb99aa98cb645ff1cf1ddf0747ac7346`；
- verified tree：
  `f87a806f325bae0e8229afcfc33ccadcac3b2e41`；
- backup 与替换前 HEAD diff 为空；
- 独立 detached worktree 干净检出，并确认存在 MC-W01 Plan 与
  `teamcontrol/service.go`；
- `git fsck --no-dangling` PASS；
- `BACKUP-VERIFIED` 硬门禁通过，允许进入 MC-W01-S03；main、origin、
  远端均未修改或推送。

| 2026-07-29 | `MC-W01-S02` | `active → complete` | BACKUP-VERIFIED PASS | 冻结 upstream 后替换目标 tree |
| 2026-07-29 | `MC-W01-S03` | `planned → active` | 等待 exact upstream tuple | 创建目标分支 |
