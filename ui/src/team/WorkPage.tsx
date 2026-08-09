import { FormEvent, useCallback, useState } from "react";
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
import { TeamAssignment, TeamIssue, TeamMember, TeamWorkItem } from "./types";
import { useAsyncData } from "./use-data";
import { nextWorkItemStatuses } from "./workflow-state";

interface WorkData {
  work: TeamWorkItem[];
  issues: TeamIssue[];
  members: TeamMember[];
  assignments: TeamAssignment[];
}

const ACTIVE_STATUSES: TeamWorkItem["status"][] = [
  "pending",
  "ready",
  "in_progress",
  "blocked",
  "verifying",
];

export function WorkPage() {
  const { client, projectID } = useTeam();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<"all" | TeamWorkItem["status"]>("all");
  const [priority, setPriority] = useState<
    "all" | NonNullable<TeamWorkItem["priority"]>
  >("all");
  const [createOpen, setCreateOpen] = useState(false);
  const [selectedID, setSelectedID] = useState("");
  const [busy, setBusy] = useState("");
  const [actionError, setActionError] = useState("");
  const [title, setTitle] = useState("");
  const [instructions, setInstructions] = useState("");
  const [sourceIssue, setSourceIssue] = useState("");
  const [workPriority, setWorkPriority] =
    useState<NonNullable<TeamWorkItem["priority"]>>("p2");
  const [businessDomain, setBusinessDomain] = useState("");
  const [estimate, setEstimate] = useState(0);
  const [dueAt, setDueAt] = useState("");
  const [nextStatus, setNextStatus] = useState<TeamWorkItem["status"] | "">("");
  const [assignee, setAssignee] = useState("");

  const load = useCallback(async (): Promise<WorkData> => {
    const [work, issues, members, assignments] = await Promise.all([
      client.rpc<TeamWorkItem[]>("work.items", { project_id: projectID }),
      client.rpc<TeamIssue[]>("issue.list", { project_id: projectID }),
      client.rpc<TeamMember[]>("team.members", { project_id: projectID }),
      client.rpc<TeamAssignment[]>("assignment.list", {
        project_id: projectID,
        target_type: "work_item",
      }),
    ]);
    return { work, issues, members, assignments };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  const mutate = async (
    key: string,
    action: () => Promise<unknown>,
    done?: () => void,
  ) => {
    setBusy(key);
    setActionError("");
    try {
      await action();
      done?.();
      state.reload();
    } catch (reason) {
      setActionError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  const create = (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim() || !instructions.trim()) return;
    void mutate(
      "create",
      () =>
        client.rpc("work.create", {
          project_id: projectID,
          issue_id: sourceIssue,
          title: title.trim(),
          instructions: instructions.trim(),
          priority: workPriority,
          business_domain: businessDomain.trim(),
          estimate_points: estimate,
          due_at: dueAt ? new Date(`${dueAt}T23:59:59`).toISOString() : null,
        }),
      () => {
        setTitle("");
        setInstructions("");
        setSourceIssue("");
        setBusinessDomain("");
        setEstimate(0);
        setDueAt("");
        setCreateOpen(false);
      },
    );
  };

  if (state.loading && !state.data) return <Loading label="加载统一任务图…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;

  const { work, issues, members, assignments } = state.data;
  const names = new Map(
    members.map((member) => [member.id, member.display_name]),
  );
  const owners = new Map(
    assignments
      .filter((item) => item.status === "active" && item.role === "owner")
      .map((item) => [item.target_id, item]),
  );
  const issueNames = new Map(issues.map((issue) => [issue.id, issue]));
  const normalized = query.trim().toLowerCase();
  const filtered = work.filter((item) => {
    if (status !== "all" && item.status !== status) return false;
    if (priority !== "all" && item.priority !== priority) return false;
    if (!normalized) return true;
    return `${item.id} ${item.title} ${item.instructions ?? ""} ${item.business_domain ?? ""}`
      .toLowerCase()
      .includes(normalized);
  });
  const selected = work.find((item) => item.id === selectedID) ?? null;
  const selectedOwner = selected ? owners.get(selected.id) : undefined;

  const transition = () => {
    if (!selected || !nextStatus) return;
    void mutate(
      `transition:${selected.id}`,
      () =>
        client.rpc("work.transition", {
          project_id: projectID,
          work_item_id: selected.id,
          status: nextStatus,
        }),
      () => setNextStatus(""),
    );
  };

  const assign = () => {
    if (!selected || !assignee) return;
    void mutate(
      `assign:${selected.id}`,
      () =>
        client.rpc("assignment.create", {
          project_id: projectID,
          target_type: "work_item",
          target_id: selected.id,
          user_id: assignee,
          role: "owner",
        }),
      () => setAssignee(""),
    );
  };

  return (
    <div className="collection-page">
      <PageHeader
        title="任务"
        description="所有任务必须追溯到需求、Bug、风险、Review Finding 或知识改进。"
        actions={
          <Button
            tone="accent"
            icon="arrow"
            onClick={() => setCreateOpen((value) => !value)}
          >
            新建任务
          </Button>
        }
      />
      <div className="summary-strip">
        <span>
          <strong>{work.length}</strong>
          <small>全部</small>
        </span>
        <span>
          <strong>
            {
              work.filter((item) => ACTIVE_STATUSES.includes(item.status))
                .length
            }
          </strong>
          <small>活动</small>
        </span>
        <span>
          <strong>
            {work.filter((item) => item.status === "blocked").length}
          </strong>
          <small>阻塞</small>
        </span>
        <span>
          <strong>
            {work.filter((item) => item.status === "verifying").length}
          </strong>
          <small>验证中</small>
        </span>
        <span>
          <strong>
            {work.filter((item) => item.status === "done").length}
          </strong>
          <small>完成</small>
        </span>
      </div>
      {createOpen ? (
        <form className="create-surface" onSubmit={create}>
          <div className="surface-heading">
            <div>
              <strong>创建 WorkItem</strong>
              <small>创建后仍需负责人、验收和证据才能完成。</small>
            </div>
            <Button type="button" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>标题</span>
              <input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                maxLength={300}
                required
              />
            </label>
            <label>
              <span>来源 Issue</span>
              <select
                value={sourceIssue}
                onChange={(event) => setSourceIssue(event.target.value)}
              >
                <option value="">独立任务</option>
                {issues.map((issue) => (
                  <option key={issue.id} value={issue.id}>
                    {issue.id} · {issue.title}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span>优先级</span>
              <select
                value={workPriority}
                onChange={(event) =>
                  setWorkPriority(
                    event.target.value as NonNullable<TeamWorkItem["priority"]>,
                  )
                }
              >
                {["p0", "p1", "p2", "p3", "p4"].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label>
              <span>业务域</span>
              <input
                value={businessDomain}
                onChange={(event) => setBusinessDomain(event.target.value)}
              />
            </label>
            <label>
              <span>估算点</span>
              <input
                type="number"
                min={0}
                max={10000}
                value={estimate}
                onChange={(event) => setEstimate(Number(event.target.value))}
              />
            </label>
            <label>
              <span>截止日期</span>
              <input
                type="date"
                value={dueAt}
                onChange={(event) => setDueAt(event.target.value)}
              />
            </label>
            <label className="field-span-full">
              <span>执行说明 / 验收约束</span>
              <textarea
                rows={4}
                value={instructions}
                onChange={(event) => setInstructions(event.target.value)}
                required
              />
            </label>
          </div>
          <div className="button-row">
            <Button
              tone="accent"
              busy={busy === "create"}
              disabled={!title.trim() || !instructions.trim()}
            >
              创建
            </Button>
          </div>
        </form>
      ) : null}
      {actionError ? (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      ) : null}
      <div className={`master-detail ${selected ? "has-detail" : ""}`}>
        <section className="collection-surface">
          <div className="collection-toolbar">
            <input
              value={query}
              onChange={(event) => setQuery(event.target.value)}
              placeholder="搜索任务、ID、业务域…"
              aria-label="搜索任务"
            />
            <select
              value={status}
              onChange={(event) =>
                setStatus(event.target.value as typeof status)
              }
            >
              <option value="all">全部状态</option>
              {[
                "pending",
                "ready",
                "in_progress",
                "blocked",
                "verifying",
                "done",
                "cancelled",
              ].map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <select
              value={priority}
              onChange={(event) =>
                setPriority(event.target.value as typeof priority)
              }
            >
              <option value="all">全部优先级</option>
              {["p0", "p1", "p2", "p3", "p4"].map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <small>{filtered.length} 条</small>
          </div>
          {filtered.length === 0 ? (
            <Empty
              title="没有匹配任务"
              detail="调整筛选，或创建一个有明确来源和验收条件的任务。"
            />
          ) : (
            <div className="table-wrap">
              <table className="interactive-table">
                <thead>
                  <tr>
                    <th>任务</th>
                    <th>来源</th>
                    <th>负责人</th>
                    <th>优先级</th>
                    <th>状态</th>
                    <th>更新</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((item) => {
                    const owner = owners.get(item.id);
                    return (
                      <tr
                        key={item.id}
                        className={selectedID === item.id ? "is-selected" : ""}
                        onClick={() => {
                          setSelectedID(item.id);
                          setNextStatus("");
                          setAssignee("");
                        }}
                      >
                        <td>
                          <strong>{item.title}</strong>
                          <small>
                            {item.id} · {item.business_domain || "未设置业务域"}
                          </small>
                        </td>
                        <td>
                          {item.issue_id
                            ? (issueNames.get(item.issue_id)?.type ?? "issue")
                            : "—"}
                          <small>{item.issue_id || "无关联"}</small>
                        </td>
                        <td>
                          {names.get(owner?.user_id ?? "") ||
                            owner?.user_id ||
                            item.assignee_id ||
                            "未分配"}
                        </td>
                        <td>
                          <Status
                            tone={
                              ["p0", "p1"].includes(item.priority ?? "")
                                ? "danger"
                                : item.priority === "p2"
                                  ? "warning"
                                  : "neutral"
                            }
                          >
                            {item.priority || "p2"}
                          </Status>
                        </td>
                        <td>
                          <Status tone={toneForState(item.status)}>
                            {item.status}
                          </Status>
                        </td>
                        <td>{formatRelative(item.updated_at)}</td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </section>
        {selected ? (
          <aside className="detail-panel" aria-label="任务详情">
            <div className="detail-header">
              <div>
                <small>{selected.id}</small>
                <h2>{selected.title}</h2>
              </div>
              <button
                className="icon-button"
                onClick={() => setSelectedID("")}
                aria-label="关闭任务详情"
              >
                ×
              </button>
            </div>
            <div className="detail-body">
              <div className="detail-status-row">
                <Status tone={toneForState(selected.status)}>
                  {selected.status}
                </Status>
                <Status>{selected.priority || "p2"}</Status>
              </div>
              <section>
                <h3>执行说明</h3>
                <p>{selected.instructions || "未提供"}</p>
              </section>
              <dl className="property-list">
                <div>
                  <dt>来源</dt>
                  <dd>
                    {selected.issue_id
                      ? `${issueNames.get(selected.issue_id)?.type ?? "issue"} · ${selected.issue_id}`
                      : "独立任务"}
                  </dd>
                </div>
                <div>
                  <dt>负责人</dt>
                  <dd>
                    {names.get(selectedOwner?.user_id ?? "") ||
                      selectedOwner?.user_id ||
                      "未分配"}
                  </dd>
                </div>
                <div>
                  <dt>估算</dt>
                  <dd>{selected.estimate_points ?? 0} pts</dd>
                </div>
                <div>
                  <dt>截止</dt>
                  <dd>
                    {selected.due_at
                      ? new Date(selected.due_at).toLocaleDateString("zh-CN")
                      : "未设置"}
                  </dd>
                </div>
                <div>
                  <dt>开发任务</dt>
                  <dd>{selected.task_id || "尚未绑定"}</dd>
                </div>
              </dl>
              <section>
                <h3>状态流转</h3>
                <div className="control-row">
                  <select
                    value={nextStatus}
                    onChange={(event) =>
                      setNextStatus(event.target.value as typeof nextStatus)
                    }
                  >
                    <option value="">选择下一状态</option>
                    {nextWorkItemStatuses(selected.status).map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                  <Button
                    busy={busy === `transition:${selected.id}`}
                    disabled={!nextStatus}
                    onClick={transition}
                  >
                    流转
                  </Button>
                </div>
              </section>
              <section>
                <h3>负责人</h3>
                {selectedOwner ? (
                  <div className="control-row">
                    <span>
                      {names.get(selectedOwner.user_id) ||
                        selectedOwner.user_id}
                    </span>
                    <Button
                      tone="danger"
                      busy={busy === `release:${selected.id}`}
                      onClick={() =>
                        void mutate(`release:${selected.id}`, () =>
                          client.rpc("assignment.release", {
                            project_id: projectID,
                            assignment_id: selectedOwner.id,
                          }),
                        )
                      }
                    >
                      解除
                    </Button>
                  </div>
                ) : (
                  <div className="control-row">
                    <select
                      value={assignee}
                      onChange={(event) => setAssignee(event.target.value)}
                    >
                      <option value="">选择成员</option>
                      {members
                        .filter((member) => member.status !== "disabled")
                        .map((member) => (
                          <option key={member.id} value={member.id}>
                            {member.display_name}
                          </option>
                        ))}
                    </select>
                    <Button
                      busy={busy === `assign:${selected.id}`}
                      disabled={!assignee}
                      onClick={assign}
                    >
                      分配
                    </Button>
                  </div>
                )}
              </section>
              <section>
                <h3>证据要求</h3>
                {(selected.verification_commands ?? []).length ? (
                  <ul className="plain-list">
                    {selected.verification_commands?.map((command) => (
                      <li key={command.join("\u0000")}>
                        <code>{command.join(" ")}</code>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p>尚未登记冻结验证命令；开发任务 Freeze 时必须补齐。</p>
                )}
              </section>
            </div>
          </aside>
        ) : null}
      </div>
    </div>
  );
}
