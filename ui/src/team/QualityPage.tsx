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
import { nextIssueStatuses } from "./workflow-state";

interface QualityData {
  issues: TeamIssue[];
  members: TeamMember[];
  assignments: TeamAssignment[];
  work: TeamWorkItem[];
}

export function QualityPage() {
  const { client, projectID } = useTeam();
  const [query, setQuery] = useState("");
  const [type, setType] = useState<"all" | TeamIssue["type"]>("all");
  const [status, setStatus] = useState<"open" | "all" | TeamIssue["status"]>(
    "open",
  );
  const [selectedID, setSelectedID] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [busy, setBusy] = useState("");
  const [actionError, setActionError] = useState("");
  const [issueType, setIssueType] = useState<TeamIssue["type"]>("bug");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [severity, setSeverity] = useState<TeamIssue["severity"]>("medium");
  const [priority, setPriority] =
    useState<NonNullable<TeamIssue["priority"]>>("p2");
  const [module, setModule] = useState("");
  const [environment, setEnvironment] = useState("");
  const [reproduction, setReproduction] = useState("");
  const [expected, setExpected] = useState("");
  const [actual, setActual] = useState("");
  const [labels, setLabels] = useState("");
  const [nextStatus, setNextStatus] = useState<TeamIssue["status"] | "">("");
  const [resolution, setResolution] = useState("");
  const [assignee, setAssignee] = useState("");

  const load = useCallback(async (): Promise<QualityData> => {
    const [issues, members, assignments, work] = await Promise.all([
      client.rpc<TeamIssue[]>("issue.list", { project_id: projectID }),
      client.rpc<TeamMember[]>("team.members", { project_id: projectID }),
      client.rpc<TeamAssignment[]>("assignment.list", {
        project_id: projectID,
        target_type: "issue",
      }),
      client.rpc<TeamWorkItem[]>("work.items", { project_id: projectID }),
    ]);
    return { issues, members, assignments, work };
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
    if (!title.trim()) return;
    void mutate(
      "create",
      () =>
        client.rpc("issue.create", {
          project_id: projectID,
          type: issueType,
          title: title.trim(),
          description: description.trim(),
          severity,
          priority,
          module: module.trim(),
          environment: environment.trim(),
          labels: labels
            .split(",")
            .map((value) => value.trim())
            .filter(Boolean),
          reproduction: reproduction.trim(),
          expected: expected.trim(),
          actual: actual.trim(),
        }),
      () => {
        setTitle("");
        setDescription("");
        setModule("");
        setEnvironment("");
        setReproduction("");
        setExpected("");
        setActual("");
        setLabels("");
        setCreateOpen(false);
      },
    );
  };

  if (state.loading && !state.data)
    return <Loading label="加载 Bug、风险与修复任务…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;
  const { issues, members, assignments, work } = state.data;
  const names = new Map(
    members.map((member) => [member.id, member.display_name]),
  );
  const owners = new Map(
    assignments
      .filter((item) => item.status === "active" && item.role === "owner")
      .map((item) => [item.target_id, item]),
  );
  const normalized = query.trim().toLowerCase();
  const filtered = issues.filter((item) => {
    if (type !== "all" && item.type !== type) return false;
    if (
      status === "open" &&
      ["resolved", "closed", "cancelled"].includes(item.status)
    )
      return false;
    if (status !== "all" && status !== "open" && item.status !== status)
      return false;
    return (
      !normalized ||
      `${item.id} ${item.title} ${item.description ?? ""} ${item.module ?? ""} ${(item.labels ?? []).join(" ")}`
        .toLowerCase()
        .includes(normalized)
    );
  });
  const selected = issues.find((item) => item.id === selectedID) ?? null;
  const owner = selected ? owners.get(selected.id) : undefined;
  const linkedWork = selected
    ? work.filter((item) => item.issue_id === selected.id)
    : [];
  const needsResolution = nextStatus === "resolved" || nextStatus === "closed";

  const transition = () => {
    if (!selected || !nextStatus) return;
    void mutate(
      `transition:${selected.id}`,
      () =>
        client.rpc("issue.transition", {
          project_id: projectID,
          issue_id: selected.id,
          status: nextStatus,
          resolution: resolution.trim(),
        }),
      () => {
        setNextStatus("");
        setResolution("");
      },
    );
  };

  return (
    <div className="collection-page">
      <PageHeader
        title="Bug 与风险"
        description="Bug 是已发生问题，Risk 是可能发生的损失；共享任务和证据闭环，但不混用语义。"
        actions={
          <Button
            tone="accent"
            onClick={() => setCreateOpen((value) => !value)}
          >
            登记
          </Button>
        }
      />
      <div className="summary-strip">
        <span>
          <strong>
            {
              issues.filter(
                (item) =>
                  item.type === "bug" &&
                  !["resolved", "closed"].includes(item.status),
              ).length
            }
          </strong>
          <small>未关闭 Bug</small>
        </span>
        <span>
          <strong>
            {
              issues.filter(
                (item) =>
                  item.type === "risk" &&
                  !["resolved", "closed"].includes(item.status),
              ).length
            }
          </strong>
          <small>活动风险</small>
        </span>
        <span>
          <strong>
            {
              issues.filter(
                (item) =>
                  ["critical", "high"].includes(item.severity) &&
                  !["resolved", "closed"].includes(item.status),
              ).length
            }
          </strong>
          <small>高严重度</small>
        </span>
        <span>
          <strong>
            {issues.filter((item) => item.status === "blocked").length}
          </strong>
          <small>阻塞</small>
        </span>
        <span>
          <strong>
            {issues.filter((item) => item.status === "verifying").length}
          </strong>
          <small>验证中</small>
        </span>
      </div>
      {createOpen ? (
        <form className="create-surface" onSubmit={create}>
          <div className="surface-heading">
            <div>
              <strong>登记质量事项</strong>
              <small>风险必须记录影响和响应策略；Bug 应提供复现证据。</small>
            </div>
            <Button type="button" onClick={() => setCreateOpen(false)}>
              取消
            </Button>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>类型</span>
              <select
                value={issueType}
                onChange={(event) =>
                  setIssueType(event.target.value as TeamIssue["type"])
                }
              >
                {["bug", "risk", "improvement", "task"].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label className="field-span-two">
              <span>标题</span>
              <input
                value={title}
                onChange={(event) => setTitle(event.target.value)}
                maxLength={300}
                required
              />
            </label>
            <label>
              <span>严重度</span>
              <select
                value={severity}
                onChange={(event) =>
                  setSeverity(event.target.value as TeamIssue["severity"])
                }
              >
                {["critical", "high", "medium", "low"].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label>
              <span>优先级</span>
              <select
                value={priority}
                onChange={(event) =>
                  setPriority(
                    event.target.value as NonNullable<TeamIssue["priority"]>,
                  )
                }
              >
                {["p0", "p1", "p2", "p3", "p4"].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label>
              <span>模块</span>
              <input
                value={module}
                onChange={(event) => setModule(event.target.value)}
              />
            </label>
            <label>
              <span>环境 / 影响范围</span>
              <input
                value={environment}
                onChange={(event) => setEnvironment(event.target.value)}
              />
            </label>
            <label>
              <span>标签（逗号分隔）</span>
              <input
                value={labels}
                onChange={(event) => setLabels(event.target.value)}
              />
            </label>
            <label className="field-span-full">
              <span>描述</span>
              <textarea
                rows={3}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
              />
            </label>
            <label>
              <span>{issueType === "risk" ? "触发条件" : "复现步骤"}</span>
              <textarea
                rows={3}
                value={reproduction}
                onChange={(event) => setReproduction(event.target.value)}
              />
            </label>
            <label>
              <span>{issueType === "risk" ? "预期控制" : "期望结果"}</span>
              <textarea
                rows={3}
                value={expected}
                onChange={(event) => setExpected(event.target.value)}
              />
            </label>
            <label>
              <span>{issueType === "risk" ? "潜在影响" : "实际结果"}</span>
              <textarea
                rows={3}
                value={actual}
                onChange={(event) => setActual(event.target.value)}
              />
            </label>
          </div>
          <div className="button-row">
            <Button
              tone="accent"
              busy={busy === "create"}
              disabled={!title.trim()}
            >
              登记
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
              placeholder="搜索标题、ID、模块、标签…"
            />
            <select
              value={type}
              onChange={(event) => setType(event.target.value as typeof type)}
            >
              <option value="all">全部类型</option>
              <option value="bug">Bug</option>
              <option value="risk">Risk</option>
              <option value="improvement">改进</option>
              <option value="task">事项</option>
            </select>
            <select
              value={status}
              onChange={(event) =>
                setStatus(event.target.value as typeof status)
              }
            >
              <option value="open">未关闭</option>
              <option value="all">全部状态</option>
              {[
                "new",
                "triaged",
                "assigned",
                "in_progress",
                "blocked",
                "verifying",
                "resolved",
                "closed",
                "reopened",
                "cancelled",
              ].map((value) => (
                <option key={value}>{value}</option>
              ))}
            </select>
            <small>{filtered.length} 条</small>
          </div>
          {filtered.length === 0 ? (
            <Empty title="没有匹配事项" />
          ) : (
            <div className="table-wrap">
              <table className="interactive-table">
                <thead>
                  <tr>
                    <th>事项</th>
                    <th>类型</th>
                    <th>负责人</th>
                    <th>等级</th>
                    <th>状态</th>
                    <th>更新</th>
                  </tr>
                </thead>
                <tbody>
                  {filtered.map((item) => (
                    <tr
                      key={item.id}
                      className={selectedID === item.id ? "is-selected" : ""}
                      onClick={() => {
                        setSelectedID(item.id);
                        setNextStatus("");
                        setResolution("");
                        setAssignee("");
                      }}
                    >
                      <td>
                        <strong>{item.title}</strong>
                        <small>
                          {item.id} · {item.module || "未设置模块"}
                        </small>
                      </td>
                      <td>{item.type}</td>
                      <td>
                        {names.get(owners.get(item.id)?.user_id ?? "") ||
                          owners.get(item.id)?.user_id ||
                          item.owner_id ||
                          "未分配"}
                      </td>
                      <td>
                        <Status
                          tone={
                            ["critical", "high"].includes(item.severity)
                              ? "danger"
                              : item.severity === "medium"
                                ? "warning"
                                : "neutral"
                          }
                        >
                          {item.severity} · {item.priority || "p2"}
                        </Status>
                      </td>
                      <td>
                        <Status tone={toneForState(item.status)}>
                          {item.status}
                        </Status>
                      </td>
                      <td>{formatRelative(item.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </section>
        {selected ? (
          <aside className="detail-panel">
            <div className="detail-header">
              <div>
                <small>
                  {selected.type} · {selected.id}
                </small>
                <h2>{selected.title}</h2>
              </div>
              <button
                className="icon-button"
                onClick={() => setSelectedID("")}
                aria-label="关闭详情"
              >
                ×
              </button>
            </div>
            <div className="detail-body">
              <div className="detail-status-row">
                <Status tone={toneForState(selected.status)}>
                  {selected.status}
                </Status>
                <Status
                  tone={
                    ["critical", "high"].includes(selected.severity)
                      ? "danger"
                      : "warning"
                  }
                >
                  {selected.severity}
                </Status>
              </div>
              <section>
                <h3>描述</h3>
                <p>{selected.description || "未提供"}</p>
              </section>
              <dl className="property-list">
                <div>
                  <dt>负责人</dt>
                  <dd>
                    {names.get(owner?.user_id ?? "") ||
                      owner?.user_id ||
                      "未分配"}
                  </dd>
                </div>
                <div>
                  <dt>模块</dt>
                  <dd>{selected.module || "—"}</dd>
                </div>
                <div>
                  <dt>环境/范围</dt>
                  <dd>{selected.environment || "—"}</dd>
                </div>
                <div>
                  <dt>关联任务</dt>
                  <dd>
                    {linkedWork.length
                      ? linkedWork.map((item) => item.id).join("、")
                      : "尚未创建"}
                  </dd>
                </div>
                <div>
                  <dt>重开次数</dt>
                  <dd>{selected.reopen_count ?? 0}</dd>
                </div>
              </dl>
              <section>
                <h3>{selected.type === "risk" ? "触发条件" : "复现步骤"}</h3>
                <p>{selected.reproduction || "未提供"}</p>
              </section>
              <section>
                <h3>
                  {selected.type === "risk"
                    ? "控制目标 / 潜在影响"
                    : "期望 / 实际"}
                </h3>
                <p>{selected.expected || "—"}</p>
                <p>{selected.actual || "—"}</p>
              </section>
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
                    {nextIssueStatuses(selected.status).map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                  <Button
                    busy={busy === `transition:${selected.id}`}
                    disabled={
                      !nextStatus || (needsResolution && !resolution.trim())
                    }
                    onClick={transition}
                  >
                    流转
                  </Button>
                </div>
                {needsResolution ? (
                  <textarea
                    rows={2}
                    placeholder="解决说明或关联修复证据"
                    value={resolution}
                    onChange={(event) => setResolution(event.target.value)}
                  />
                ) : null}
              </section>
              <section>
                <h3>负责人</h3>
                {owner ? (
                  <div className="control-row">
                    <span>{names.get(owner.user_id) || owner.user_id}</span>
                    <Button
                      tone="danger"
                      busy={busy === `release:${selected.id}`}
                      onClick={() =>
                        void mutate(`release:${selected.id}`, () =>
                          client.rpc("assignment.release", {
                            project_id: projectID,
                            assignment_id: owner.id,
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
                      onClick={() =>
                        void mutate(
                          `assign:${selected.id}`,
                          () =>
                            client.rpc("assignment.create", {
                              project_id: projectID,
                              target_type: "issue",
                              target_id: selected.id,
                              user_id: assignee,
                              role: "owner",
                            }),
                          () => setAssignee(""),
                        )
                      }
                    >
                      分配
                    </Button>
                  </div>
                )}
              </section>
            </div>
          </aside>
        ) : null}
      </div>
    </div>
  );
}
