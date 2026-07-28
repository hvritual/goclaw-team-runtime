# FE-EVID-W01-012 — Memory Catalog Provenance Gate Reproduction

- 日期：2026-07-26
- Wave：`FE-W01`
- Step：`FE-W01-S04`
- Issue：`FE-ISSUE-008`
- Project：`goclaw-team-runtime`
- Fixture project：synthetic `alpha`
- Actor：synthetic `importer`
- Runtime principal/role：不适用；本次为本地 package test，不经过鉴权路径
- 发现 Task：`FE-W01-TRANSPORT-R1` revision 3
- 环境：Go 1.25.5；共享只读 module cache；任务专用 worktree
- 日志脱敏：只保留命令、断言和脱敏摘要；不含凭据或生产数据
- 结论：确定性 root cause；不在 r005 allowlist，禁止顺手修复

## 复现

Revision 3 的 channels、Gateway、UI Gate 通过后执行：

```text
go test ./... -count=1 -timeout=5m
```

除 `memory/catalog` 外，命令已输出的其他 package 均通过。
`TestIngestStableRootDistinguishesSameNamedItems` 失败：默认 Markdown
ingestion 的 provenance `SourceKind` 实际为 `markdown-markdown`，冻结合同
要求 `markdown`。

随后只读执行目标测试两次：

```text
go test ./memory/catalog \
  -run '^TestIngestStableRootDistinguishesSameNamedItems$' \
  -count=1 -timeout=30s
```

两次都在远低于一秒内以同一断言失败；仅 UUID 与时间戳等动态字段不同。

## 根因与基线归属

`IngestPath` 先把空 `SourceScheme` 默认成 `markdown`，随后在
`SourceKind` 为空时对非 `git+markdown` scheme 统一追加 `-markdown`，
因而得到 `markdown-markdown`。

该行为同时冲突于：

- `DefaultConfig()` 的默认 `SourceKind: markdown`；
- 现有稳定根目录测试的默认 Markdown provenance 合同；
- CLI 对空 source kind 由 scheme 推断的语义。

以下路径在累计 base、Revision 3 Task Base 与当前 HEAD 内容一致，Revision 3
transport/channel patch 未修改它们：

| 路径 | SHA-256 |
|---|---|
| `memory/catalog/ingest.go` | `7d3cfc9721f46b587f7a59e9705a7cdcc425666b3e5a066395e9716db8ab3068` |
| `memory/catalog/service_test.go` | `f86038a3bf8cadb51bc7ba3edc05738788dcb1f3b499351958b0c0fcfe480403` |

## 影响与边界

- 直接调用 `IngestPath` 且不显式提供 `SourceKind` 时，会写入错误 provenance
  kind。
- startup 默认配置通常显式传入 `markdown`，所以默认 auto-ingest 通常不触发。
- Source URI、稳定根目录区分和默认 collection 未发现异常。
- 修复只影响后续 ingestion；现有错误记录的迁移不在本任务范围。
- r005 不包含 Memory 路径。必须登记新 Issue、Plan revision、Task revision
  和独立 worktree 后才能修改。
