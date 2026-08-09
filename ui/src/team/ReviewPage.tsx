import { useCallback, useState } from "react";
import { useTeam } from "./context";
import {
  Button,
  Empty,
  ErrorState,
  formatRelative,
  Loading,
  PageHeader,
  Status,
  toneForState,
} from "./primitives";
import { DevTask, TeamArtifact, TeamCorrelation, TeamIssue } from "./types";
import { useAsyncData } from "./use-data";

interface ReviewData {
  tasks: DevTask[];
  artifacts: TeamArtifact[];
  links: TeamCorrelation[];
  issues: TeamIssue[];
}

export function ReviewPage() {
  const { client, projectID } = useTeam();
  const [view, setView] = useState<"queue" | "artifacts" | "links">("queue");
  const [query, setQuery] = useState("");
  const load = useCallback(async (): Promise<ReviewData> => {
    const [tasks, artifacts, links, issues] = await Promise.all([
      client.rpc<DevTask[]>("dev.tasks", { project_id: projectID }),
      client.rpc<TeamArtifact[]>("artifact.list", { project_id: projectID }),
      client.rpc<TeamCorrelation[]>("correlation.list", {
        project_id: projectID,
      }),
      client.rpc<TeamIssue[]>("issue.list", { project_id: projectID }),
    ]);
    return { tasks, artifacts, links, issues };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);
  if (state.loading && !state.data)
    return <Loading label="加载代码评审、PR 与证据链…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;
  const { tasks, artifacts, links, issues } = state.data;
  const normalized = query.trim().toLowerCase();
  const reviewTasks = tasks.filter(
    (task) =>
      !normalized ||
      `${task.id} ${task.title} ${task.goal.objective}`
        .toLowerCase()
        .includes(normalized),
  );
  const reviewArtifacts = artifacts.filter((item) =>
    ["pull_request", "commit", "evidence", "report", "build", "diff"].includes(
      item.kind,
    ),
  );
  const failed = tasks.filter(
    (task) => task.last_gate && !task.last_gate.passed,
  ).length;
  const waiting = tasks.filter((task) =>
    ["review_pending", "awaiting_acceptance"].includes(task.status),
  ).length;
  const openReviewIssues = issues.filter(
    (issue) =>
      ["bug", "risk"].includes(issue.type) &&
      !["resolved", "closed"].includes(issue.status),
  );

  return (
    <div className="collection-page">
      <PageHeader
        title="代码评审"
        description="确定性检查先于模型 Review；Agent 只能提交 Finding，Go DoneGate 才能裁决完成。"
      />
      <div className="summary-strip">
        <span>
          <strong>{waiting}</strong>
          <small>待人工处理</small>
        </span>
        <span>
          <strong>
            {tasks.filter((task) => task.last_gate?.passed).length}
          </strong>
          <small>DoneGate 通过</small>
        </span>
        <span>
          <strong>{failed}</strong>
          <small>Gate 失败</small>
        </span>
        <span>
          <strong>
            {
              reviewArtifacts.filter((item) => item.kind === "pull_request")
                .length
            }
          </strong>
          <small>PR</small>
        </span>
        <span>
          <strong>{openReviewIssues.length}</strong>
          <small>质量事项</small>
        </span>
      </div>
      <div className="collection-surface">
        <div className="tab-toolbar">
          <div className="segmented-control" role="tablist">
            <button
              className={view === "queue" ? "is-active" : ""}
              onClick={() => setView("queue")}
            >
              评审队列
            </button>
            <button
              className={view === "artifacts" ? "is-active" : ""}
              onClick={() => setView("artifacts")}
            >
              PR 与证据
            </button>
            <button
              className={view === "links" ? "is-active" : ""}
              onClick={() => setView("links")}
            >
              关联链
            </button>
          </div>
          <input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="搜索任务或证据…"
          />
        </div>
        {view === "queue" ? (
          <div className="review-list">
            {reviewTasks.length === 0 ? (
              <Empty title="没有匹配的评审任务" />
            ) : (
              reviewTasks.map((task) => (
                <article className="review-row" key={task.id}>
                  <div className="review-main">
                    <div>
                      <strong>{task.title}</strong>
                      <small>
                        {task.id} · revision {task.compile.revision} ·{" "}
                        {formatRelative(task.updated_at)}
                      </small>
                    </div>
                    <Status tone={toneForState(task.status)}>
                      {task.status}
                    </Status>
                  </div>
                  <p>{task.goal.objective}</p>
                  <div className="review-checks">
                    {(["scenario", "capacity", "risk", "cost"] as const).map(
                      (kind) => (
                        <Status
                          key={kind}
                          tone={toneForState(task.reviews[kind]?.decision)}
                        >
                          {kind} · {task.reviews[kind]?.decision ?? "pending"}
                        </Status>
                      ),
                    )}
                  </div>
                  {task.last_gate ? (
                    <div
                      className={`gate-result ${task.last_gate.passed ? "is-passed" : "is-failed"}`}
                    >
                      <strong>
                        {task.last_gate.passed
                          ? "DoneGate passed"
                          : "DoneGate failed"}
                      </strong>
                      <span>{task.last_gate.verdict}</span>
                      {task.last_gate.reasons?.length ? (
                        <small>{task.last_gate.reasons.join("；")}</small>
                      ) : null}
                      <code>{task.last_gate.evidence_path}</code>
                    </div>
                  ) : (
                    <div className="safety-note">
                      尚无 EvidencePackage，不能进入最终验收。
                    </div>
                  )}
                  <div className="button-row">
                    <Button
                      onClick={() =>
                        void navigator.clipboard.writeText(task.id)
                      }
                    >
                      复制 Task ID
                    </Button>
                    {task.branch ? (
                      <Button
                        onClick={() =>
                          void navigator.clipboard.writeText(task.branch ?? "")
                        }
                      >
                        复制分支
                      </Button>
                    ) : null}
                  </div>
                </article>
              ))
            )}
          </div>
        ) : null}
        {view === "artifacts" ? (
          reviewArtifacts.length === 0 ? (
            <Empty
              title="尚无 PR 或评审证据"
              detail="Runner 与 link-pr 流程产出的真实 Artifact 会显示在这里。"
            />
          ) : (
            <div className="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Artifact</th>
                    <th>类型</th>
                    <th>资源</th>
                    <th>来源</th>
                    <th>时间</th>
                    <th />
                  </tr>
                </thead>
                <tbody>
                  {reviewArtifacts.map((item) => (
                    <tr key={item.id}>
                      <td>
                        <strong>{item.name}</strong>
                        <small>{item.id}</small>
                      </td>
                      <td>{item.kind}</td>
                      <td>{item.resource_type}</td>
                      <td>{item.created_by}</td>
                      <td>{formatRelative(item.created_at)}</td>
                      <td>
                        {/^https?:\/\//.test(item.uri) ? (
                          <a
                            className="text-action"
                            href={item.uri}
                            target="_blank"
                            rel="noreferrer"
                          >
                            打开
                          </a>
                        ) : (
                          <Button
                            onClick={() =>
                              void navigator.clipboard.writeText(item.uri)
                            }
                          >
                            复制 URI
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )
        ) : null}
        {view === "links" ? (
          links.length === 0 ? (
            <Empty title="尚无关联链" />
          ) : (
            <div className="link-list">
              {links.map((link) => (
                <article key={link.id}>
                  <span>{link.source_type}</span>
                  <strong>{link.source_id}</strong>
                  <i>→ {link.relation} →</i>
                  <span>{link.target_type}</span>
                  <strong>{link.target_id}</strong>
                </article>
              ))}
            </div>
          )
        ) : null}
      </div>
    </div>
  );
}
