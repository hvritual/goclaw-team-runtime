---
schema: goclaw.wave/v1
wave_id: MVP-W00
track_id: MVP-RECOVERY-2026-07
title: Authoritative source and reproducible recovery release
revision: 2
supersedes: plan-r001.md
plan_status: approved
wave_state: active
approved_by:
  - user-directive-2026-07-28
  - recovery-code-review-block
  - recovery-security-review-block
  - recovery-docs-review-block
owner: Codex root agent
reviewers:
  - recovery_code_review
  - recovery_security_review
  - recovery_docs_review
depends_on: []
created_at: 2026-07-28
updated_at: 2026-07-28
steps:
  - MVP-W00-S01
  - MVP-W00-S02
  - MVP-W00-S03
  - MVP-W00-S04
  - MVP-W00-S05A
  - MVP-W00-S05B
  - MVP-W00-S05C
  - MVP-W00-S05D
  - MVP-W00-S05E
allowed_change_scope:
  - docs/waves/**
  - docs/recovery/**
  - .tool-versions
  - scripts/recovery/**
  - scripts/build-release.sh
  - ui/package.json
  - ui/package-lock.json
  - plugins/obsidian-goclaw/package.json
  - plugins/obsidian-goclaw/package-lock.json
product_code_changes_allowed: true
---

# MVP-W00 r002 — 可重放、原子且身份明确的恢复发布

本修订继承 [`plan-r001`](plan-r001.md) 的来源、非目标、已通过确定性 Gate
和停止条件。三路独立复核一致阻止创建恢复标签，因此扩展恢复范围；本修订
不授权任何运行时业务行为修改。

## 修订原因

| Finding | 严重度 | r001 缺口 | r002 处理 |
|---|---|---|---|
| `REC-P1-001` | P1 | 来源比较指向可变工作树 | 新增 import tag 固定树验证器 |
| `REC-P1-002` | P1 | 发布非原子、非位级重现 | 锁、stage、规范化归档、版本目录原子发布 |
| `REC-P1-003` | P1 | runtime/plugin 归档回读不足 | 统一安全 archive validator |
| `REC-P1-004` | P1 | source 包缺工具链身份 | 纳入 `.tool-versions` 和 npm packageManager |
| `REC-P1-005` | P1 | Obsidian 外部/内部版本冲突 | 独立组件版本命名和 release manifest 映射 |
| `REC-P1-006` | P1 | 原始/重建归档同名异物 | input/rebuilt manifest 分离、内容寻址 |
| `REC-P1-007` | P1 | Registry/Plan/Index 状态与范围冲突 | 新 Plan revisions 原子收敛投影 |
| `REC-P1-008` | P1 | Evidence/Decision ID 语义错位 | 追加式更正并恢复 001–006 定义 |

## 目标

在不改变 GoClaw 运行时行为的前提下，使恢复输入、Git import tree、工具链
和重建制品能够由另一台机器安全重放；同一 commit 连续两次洁净构建的最终
归档 SHA-256 必须完全一致。

## 范围

### 包含

- 固定比较 `v0.8.0-pilot.1-import^{commit}` 与原始归档。
- 修复 Wave 状态、依赖、范围、Evidence 和 Decision 投影。
- 为发布构建增加互斥锁、隔离 staging、规范化 tar/gzip 元数据。
- 最终制品只通过版本目录原子发布；已存在同版本只能内容相同。
- 所有归档在解压前验证路径、类型、链接、重复项和精确成员合同。
- source 包含 `.tool-versions`；npm 版本进入两个 package manifest。
- Obsidian 使用自身组件版本，release manifest 显式映射 runtime/component。
- 区分原始输入 manifest、S04 快照和 recovered 最终 manifest。

### 不包含

- 修改 TeamControl、Runner、Gateway、Session、Provider 或 UI 运行时。
- 修改 Obsidian 插件功能或把其内部版本伪升为 runtime 版本。
- 处理 GitHub Actions、Docker digest、SBOM、签名和外部 WORM；这些 P2
  进入后续供应链 Wave。
- 覆盖或删除 2026-07-27 原始归档。

## 分步计划

| Step ID | 前置 | 计划动作 | 验证 | 状态 |
|---|---|---|---|---|
| `MVP-W00-S05A` | r002 active | 收敛 Registry/Plan/Index/Evidence/Decision | JSON/Plan 一致性审查 | `planned` |
| `MVP-W00-S05B` | S05A | 增加固定 import tree 来源验证器 | 611/611，内容/执行位/extra=0 | `planned` |
| `MVP-W00-S05C` | S05B | 原子、规范化发布和版本映射 | archive validator + clean tree | `planned` |
| `MVP-W00-S05D` | S05C | 同一 commit 连续两次洁净构建 | 全部最终 SHA 完全一致 | `planned` |
| `MVP-W00-S05E` | S05D | 三路独立复核与恢复标签 | P0/P1=0，clean tree | `planned` |

## 新增验收

```bash
scripts/recovery/verify-source-import.sh \
  /immutable/input/goclaw-team-runtime-source-0.8.0-pilot.1.tar.gz

RELEASE_VERSION=0.8.0-pilot.1-recovered.1 \
  INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh

# 对同一 commit 再执行一次；脚本只接受与已发布版本目录完全相同的结果。
RELEASE_VERSION=0.8.0-pilot.1-recovered.1 \
  INCLUDE_OBSIDIAN_PLUGIN=1 ./scripts/build-release.sh
```

## 退出门禁补充

- [ ] 固定 import tree 验证器在干净 checkout 通过。
- [ ] Registry、Plan、Index 的 active/blocked、依赖、scope、product flag 一致。
- [ ] MVP Evidence ID 严格保持 001 provenance、002 Go、003 Web、
  004 Obsidian、005 Release、006 Review。
- [ ] 原始输入和 recovered 输出使用不同 locator、版本和 manifest。
- [ ] 两次 recovered 构建全部 SHA 位级一致。
- [ ] 三路复核关闭全部 P1。

## 回滚

- 发布脚本变更失败时回到 r001 的 import/provenance 提交，保留 BLOCK 结论；
- 不覆盖 `...-library/` 下的原始归档；
- 未通过双构建和复核前不创建 recovered tag。
