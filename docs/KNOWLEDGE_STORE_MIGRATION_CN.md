# 从 Obsidian Vault 迁移到通用 Markdown / Git 知识根

版本：GoClaw Team Runtime `0.7.0`。

## 结论

迁移不改变知识正文格式，也不重建 Catalog 概念身份。变化的是来源适配器：

```text
旧：Obsidian Vault → Markdown → Catalog
新：普通目录或 Git 工作树 → Markdown → Catalog
```

Obsidian 可以继续打开同一目录，但 GoClaw 不再要求安装插件，也不依赖 `.obsidian/`。

## 保留不变的资产

- `01-goals` 到 `05-knowledge` 的 Markdown 正文；
- `08-reviews` 的知识提案投影；
- Frontmatter 中的 `project_id`、`type`、`goclaw.work_id` 和 `goclaw.expression_id`；
- Catalog SQLite 中的 Record、Authority、Relation、Decision 和 Circulation；
- Harness Trace 中的 Memory ID；
- Governance 审批记录；
- Document Registry 的 URI、revision、checksum 与 supersedes 关系。

禁止只复制 Markdown 后丢弃 Catalog SQLite；那会丢失批准状态、权威词、关系、复核期限和流通历史。

## 配置

### 普通文件目录

```json
{
  "harness": {
    "knowledge_root": "/srv/goclaw/knowledge",
    "knowledge_backend": "filesystem"
  },
  "memory": {
    "catalog": {
      "source_root": "/srv/goclaw/knowledge",
      "source_scheme": "markdown",
      "source_kind": "managed-markdown"
    }
  }
}
```

知识提案批准时校验目标文件 SHA-256。

### Git 工作树

```json
{
  "harness": {
    "knowledge_root": "/srv/goclaw/knowledge-repo",
    "knowledge_backend": "git"
  },
  "memory": {
    "catalog": {
      "source_root": "/srv/goclaw/knowledge-repo",
      "source_scheme": "git+markdown",
      "source_kind": "git-markdown"
    }
  }
}
```

Git 模式下，知识提案冻结：

- 目标文件 SHA-256；
- 创建提案时的 `git rev-parse HEAD`；
- `git+markdown://<project_id>/<relative_path>` 来源 URI。

批准时任一条件漂移都会失败关闭。批准只写工作树，不自动 commit、push 或 merge；人类仍需按知识仓库规则提交。

旧 `harness.vault_path` 会继续映射为 `knowledge_root`，用于无停机升级，但新配置不应继续使用。

## 迁移步骤

1. 对运行时和 Catalog 做一致性备份。
2. 冻结新的知识审批，等待所有 pending 提案处理或明确作废。
3. 复制原 Vault 中的受治理 Markdown；不要复制 Token、SQLite、JSONL、lease 或 OAuth。
4. 若使用 Git，初始化仓库并提交迁移基线：

   ```bash
   git init
   git add 00-index 01-goals 02-decisions 03-constraints \
     04-requirements 05-knowledge 06-test-plans 07-runbooks \
     08-reviews 09-releases
   git commit -m "Import governed knowledge baseline"
   ```

5. 修改 `knowledge_root`、`knowledge_backend` 和 Catalog 来源配置。
6. 保持 `memory.catalog.auto_ingest=false`，先手动扫描：

   ```bash
   goclaw memory catalog ingest /srv/goclaw/knowledge-repo \
     --project project-alpha \
     --source-root /srv/goclaw/knowledge-repo \
     --source-scheme git+markdown \
     --source-kind git-markdown \
     --collection governed-markdown
   ```

7. 抽样比对 title、kind、project、source URI、revision 与 checksum。
8. 检查新扫描只生成 pending 候选，没有自动进入 active。
9. 对同一概念使用原 `work_id` / `expression_id`，通过人工审批形成新 Manifestation，不要创建平行概念。
10. 完成 Web Console 的知识搜索、候选审批、知识提案、冲突拒绝和 Trace 引用验收。
11. 再决定是否启用 `auto_ingest`。
12. 保留原 Vault 只读备份，直到至少完成一次恢复演练。

## Catalog 身份策略

来源 URI 是 Item 身份的一部分。若历史记录使用
`obsidian://project-alpha/02-decisions/ADR-0001.md`，改为
`git+markdown://project-alpha/02-decisions/ADR-0001.md` 会形成新的来源 Item。

这不应自动合并。迁移时采用以下顺序：

1. Frontmatter 显式保留历史 `goclaw.work_id`；
2. 保留 `goclaw.expression_id`；
3. 新来源作为同一 Work/Expression 的新 Manifestation/Item 候选；
4. 人工确认后批准；
5. 必要时以 `supersedes` 关联旧 Manifestation；
6. 不删除旧记录，保持可审计来源链。

## 回滚

如果 Git 知识根不可用：

1. 停止 GoClaw 单写者；
2. 恢复知识工作树到已知 Commit；
3. 恢复与该 Commit 同一备份点的 Catalog SQLite；
4. 把 `knowledge_root` 指向恢复目录；
5. 启动后先运行只读搜索与 Catalog 状态检查；
6. pending 提案必须重新生成，不能绕过 revision 冲突继续批准。

不要把 `knowledge_backend` 临时改成 `filesystem` 来绕过 Git revision 冲突。
