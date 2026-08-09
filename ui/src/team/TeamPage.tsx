import { FormEvent, useCallback, useEffect, useState } from "react";
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
import {
  TeamComponentRecord,
  TeamDocumentRecord,
  TeamMember,
  TeamPolicyBundle,
  TeamPolicyStatus,
  TeamRepository,
  TeamRunner,
  TeamRunnerTask,
} from "./types";
import { useAsyncData } from "./use-data";

type ControlTab = "people" | "runners" | "assets" | "policy";

interface ControlData {
  members: TeamMember[];
  runners: TeamRunner[];
  runnerTasks: TeamRunnerTask[];
  repositories: TeamRepository[];
  documents: TeamDocumentRecord[];
  components: TeamComponentRecord[];
  policies: TeamPolicyBundle[];
  policyStatus: TeamPolicyStatus;
}

export function TeamPage() {
  const { client, projectID, projects } = useTeam();
  const [tab, setTab] = useState<ControlTab>("people");
  const [form, setForm] = useState<
    "member" | "repository" | "document" | "component" | "policy" | ""
  >("");
  const [busy, setBusy] = useState("");
  const [actionError, setActionError] = useState("");
  const [memberID, setMemberID] = useState("");
  const [memberRole, setMemberRole] = useState("developer");
  const [memberDomains, setMemberDomains] = useState("");
  const [memberCapacity, setMemberCapacity] = useState(10);
  const [repositoryName, setRepositoryName] = useState("");
  const [repositoryURL, setRepositoryURL] = useState("");
  const [repositoryPath, setRepositoryPath] = useState("");
  const [repositoryBranch, setRepositoryBranch] = useState("main");
  const [documentTitle, setDocumentTitle] = useState("");
  const [documentKey, setDocumentKey] = useState("");
  const [documentURI, setDocumentURI] = useState("");
  const [documentKind, setDocumentKind] =
    useState<TeamDocumentRecord["kind"]>("adr");
  const [componentName, setComponentName] = useState("");
  const [componentKind, setComponentKind] =
    useState<TeamComponentRecord["kind"]>("service");
  const [componentRepo, setComponentRepo] = useState("");
  const [componentPath, setComponentPath] = useState("");
  const [policyName, setPolicyName] = useState("");
  const [policyScope, setPolicyScope] =
    useState<TeamPolicyBundle["scope"]>("project");
  const [policyScopeID, setPolicyScopeID] = useState(projectID);
  const [policyVersion, setPolicyVersion] = useState(1);
  const [policyRules, setPolicyRules] = useState(
    '{\n  "review.required": true\n}',
  );

  const load = useCallback(async (): Promise<ControlData> => {
    const [
      members,
      runners,
      runnerTasks,
      repositories,
      documents,
      components,
      policies,
      policyStatus,
    ] = await Promise.all([
      client.rpc<TeamMember[]>("team.members", { project_id: projectID }),
      client.rpc<TeamRunner[]>("runner.list", { project_id: projectID }),
      client.rpc<TeamRunnerTask[]>("runner.tasks", { project_id: projectID }),
      client.rpc<TeamRepository[]>("repository.list", {
        project_id: projectID,
      }),
      client.rpc<TeamDocumentRecord[]>("document.list", {
        project_id: projectID,
      }),
      client.rpc<TeamComponentRecord[]>("component.list", {
        project_id: projectID,
      }),
      client.rpc<TeamPolicyBundle[]>("policy.list", { project_id: projectID }),
      client.rpc<TeamPolicyStatus>("policy.status", { project_id: projectID }),
    ]);
    return {
      members,
      runners,
      runnerTasks,
      repositories,
      documents,
      components,
      policies,
      policyStatus,
    };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  useEffect(() => {
    if (policyScope === "project") setPolicyScopeID(projectID);
  }, [policyScope, projectID]);

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

  if (state.loading && !state.data) return <Loading label="加载团队控制面…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;
  const {
    members,
    runners,
    runnerTasks,
    repositories,
    documents,
    components,
    policies,
    policyStatus,
  } = state.data;
  const currentProject = projects.find((project) => project.id === projectID);
  const names = new Map(
    members.map((member) => [member.id, member.display_name]),
  );
  const activeRunnerTasks = runnerTasks.filter(
    (item) => !["completed", "cancelled"].includes(item.status),
  );

  const addMember = (event: FormEvent) => {
    event.preventDefault();
    if (!memberID.trim()) return;
    void mutate(
      "member",
      () =>
        client.rpc("project.member.add", {
          project_id: projectID,
          user_id: memberID.trim(),
          role: memberRole,
          business_domains: memberDomains
            .split(",")
            .map((value) => value.trim())
            .filter(Boolean),
          capacity_points: memberCapacity,
        }),
      () => {
        setMemberID("");
        setMemberDomains("");
        setForm("");
      },
    );
  };

  const addRepository = (event: FormEvent) => {
    event.preventDefault();
    if (!repositoryName.trim()) return;
    void mutate(
      "repository",
      () =>
        client.rpc("repository.create", {
          project_id: projectID,
          name: repositoryName.trim(),
          remote_url: repositoryURL.trim(),
          local_path: repositoryPath.trim(),
          default_branch: repositoryBranch.trim() || "main",
        }),
      () => {
        setRepositoryName("");
        setRepositoryURL("");
        setRepositoryPath("");
        setForm("");
      },
    );
  };

  const addDocument = (event: FormEvent) => {
    event.preventDefault();
    if (!documentTitle.trim() || !documentKey.trim() || !documentURI.trim())
      return;
    void mutate(
      "document",
      () =>
        client.rpc("document.register", {
          project_id: projectID,
          key: documentKey.trim(),
          title: documentTitle.trim(),
          kind: documentKind,
          status: "active",
          uri: documentURI.trim(),
        }),
      () => {
        setDocumentTitle("");
        setDocumentKey("");
        setDocumentURI("");
        setForm("");
      },
    );
  };

  const addComponent = (event: FormEvent) => {
    event.preventDefault();
    if (!componentName.trim()) return;
    void mutate(
      "component",
      () =>
        client.rpc("component.register", {
          project_id: projectID,
          repository_id: componentRepo,
          name: componentName.trim(),
          kind: componentKind,
          root_path: componentPath.trim(),
        }),
      () => {
        setComponentName("");
        setComponentRepo("");
        setComponentPath("");
        setForm("");
      },
    );
  };

  const addPolicy = (event: FormEvent) => {
    event.preventDefault();
    let rules: Record<string, unknown>;
    try {
      rules = JSON.parse(policyRules) as Record<string, unknown>;
    } catch {
      setActionError("Policy Rules 必须是有效 JSON 对象");
      return;
    }
    const projectScopeID =
      policyScope === "team" ? currentProject?.team_id : projectID;
    const scopeID = policyScopeID.trim() || projectScopeID;
    if (!policyName.trim() || !scopeID) return;
    void mutate(
      "policy",
      () =>
        client.rpc("policy.put", {
          name: policyName.trim(),
          scope: policyScope,
          scope_id: scopeID,
          version: policyVersion,
          priority: 100,
          enabled: true,
          rules,
        }),
      () => {
        setPolicyName("");
        setForm("");
      },
    );
  };

  const actionLabel =
    tab === "people"
      ? "添加成员"
      : tab === "assets"
        ? "登记资产"
        : tab === "policy"
          ? "新增策略"
          : "";

  return (
    <div className="collection-page">
      <PageHeader
        title="团队控制"
        description="项目成员、Runner、仓库、文档、组件与策略都由中央单写者按项目授权返回。"
        actions={
          actionLabel ? (
            <Button
              tone="accent"
              onClick={() =>
                setForm(
                  tab === "people"
                    ? "member"
                    : tab === "policy"
                      ? "policy"
                      : "repository",
                )
              }
            >
              {actionLabel}
            </Button>
          ) : undefined
        }
      />
      <div className="summary-strip">
        <span>
          <strong>{members.length}</strong>
          <small>成员</small>
        </span>
        <span>
          <strong>
            {
              runners.filter((item) => ["online", "busy"].includes(item.status))
                .length
            }
            /{runners.length}
          </strong>
          <small>Runner 在线</small>
        </span>
        <span>
          <strong>{repositories.length}</strong>
          <small>仓库</small>
        </span>
        <span>
          <strong>{documents.length + components.length}</strong>
          <small>工程资产</small>
        </span>
        <span>
          <strong>
            {policyStatus.compliant ? "一致" : policyStatus.drift_count}
          </strong>
          <small>策略状态</small>
        </span>
      </div>
      <div className="segmented-control control-tabs" role="tablist">
        {(
          [
            ["people", "成员与容量"],
            ["runners", "Runner"],
            ["assets", "工程资产"],
            ["policy", "策略"],
          ] as Array<[ControlTab, string]>
        ).map(([id, label]) => (
          <button
            key={id}
            className={tab === id ? "is-active" : ""}
            onClick={() => {
              setTab(id);
              setForm("");
            }}
          >
            {label}
          </button>
        ))}
      </div>
      {actionError ? (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      ) : null}
      {form === "member" ? (
        <form className="create-surface compact-create" onSubmit={addMember}>
          <div className="surface-heading">
            <strong>加入项目成员</strong>
            <Button type="button" onClick={() => setForm("")}>取消</Button>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>User ID</span>
              <input
                value={memberID}
                onChange={(event) => setMemberID(event.target.value)}
                required
              />
            </label>
            <label>
              <span>项目角色</span>
              <select
                value={memberRole}
                onChange={(event) => setMemberRole(event.target.value)}
              >
                {["owner", "maintainer", "developer", "reviewer", "viewer"].map(
                  (value) => (
                    <option key={value}>{value}</option>
                  ),
                )}
              </select>
            </label>
            <label>
              <span>业务域</span>
              <input
                value={memberDomains}
                onChange={(event) => setMemberDomains(event.target.value)}
                placeholder="device, cloud"
              />
            </label>
            <label>
              <span>容量点</span>
              <input
                type="number"
                min={0}
                max={10000}
                value={memberCapacity}
                onChange={(event) =>
                  setMemberCapacity(Number(event.target.value))
                }
              />
            </label>
          </div>
          <div className="button-row">
            <Button tone="accent" busy={busy === "member"}>
              添加
            </Button>
          </div>
        </form>
      ) : null}
      {form === "repository" ? (
        <form
          className="create-surface compact-create"
          onSubmit={addRepository}
        >
          <div className="surface-heading">
            <strong>登记仓库</strong>
            <div className="button-row">
              <Button type="button" onClick={() => setForm("document")}>改为文档</Button>
              <Button type="button" onClick={() => setForm("component")}>改为组件</Button>
              <Button type="button" onClick={() => setForm("")}>取消</Button>
            </div>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>名称</span>
              <input
                value={repositoryName}
                onChange={(event) => setRepositoryName(event.target.value)}
                required
              />
            </label>
            <label>
              <span>Remote URL</span>
              <input
                value={repositoryURL}
                onChange={(event) => setRepositoryURL(event.target.value)}
              />
            </label>
            <label>
              <span>受管本地路径</span>
              <input
                value={repositoryPath}
                onChange={(event) => setRepositoryPath(event.target.value)}
              />
            </label>
            <label>
              <span>默认分支</span>
              <input
                value={repositoryBranch}
                onChange={(event) => setRepositoryBranch(event.target.value)}
              />
            </label>
          </div>
          <div className="button-row">
            <Button tone="accent" busy={busy === "repository"}>
              登记
            </Button>
          </div>
        </form>
      ) : null}
      {form === "document" ? (
        <form className="create-surface compact-create" onSubmit={addDocument}>
          <div className="surface-heading">
            <strong>登记文档</strong>
            <div className="button-row">
              <Button type="button" onClick={() => setForm("repository")}>仓库</Button>
              <Button type="button" onClick={() => setForm("component")}>组件</Button>
              <Button type="button" onClick={() => setForm("")}>取消</Button>
            </div>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>Key</span>
              <input
                value={documentKey}
                onChange={(event) => setDocumentKey(event.target.value)}
                required
              />
            </label>
            <label>
              <span>标题</span>
              <input
                value={documentTitle}
                onChange={(event) => setDocumentTitle(event.target.value)}
                required
              />
            </label>
            <label>
              <span>类型</span>
              <select
                value={documentKind}
                onChange={(event) =>
                  setDocumentKind(
                    event.target.value as TeamDocumentRecord["kind"],
                  )
                }
              >
                {[
                  "prd",
                  "adr",
                  "design",
                  "runbook",
                  "api",
                  "test_plan",
                  "report",
                  "knowledge",
                  "other",
                ].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label>
              <span>URI</span>
              <input
                value={documentURI}
                onChange={(event) => setDocumentURI(event.target.value)}
                required
              />
            </label>
          </div>
          <div className="button-row">
            <Button tone="accent" busy={busy === "document"}>
              登记
            </Button>
          </div>
        </form>
      ) : null}
      {form === "component" ? (
        <form className="create-surface compact-create" onSubmit={addComponent}>
          <div className="surface-heading">
            <strong>登记共享组件</strong>
            <div className="button-row">
              <Button type="button" onClick={() => setForm("repository")}>仓库</Button>
              <Button type="button" onClick={() => setForm("document")}>文档</Button>
              <Button type="button" onClick={() => setForm("")}>取消</Button>
            </div>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>名称</span>
              <input
                value={componentName}
                onChange={(event) => setComponentName(event.target.value)}
                required
              />
            </label>
            <label>
              <span>类型</span>
              <select
                value={componentKind}
                onChange={(event) =>
                  setComponentKind(
                    event.target.value as TeamComponentRecord["kind"],
                  )
                }
              >
                {["service", "library", "app", "module", "device", "other"].map(
                  (value) => (
                    <option key={value}>{value}</option>
                  ),
                )}
              </select>
            </label>
            <label>
              <span>仓库</span>
              <select
                value={componentRepo}
                onChange={(event) => setComponentRepo(event.target.value)}
              >
                <option value="">未绑定</option>
                {repositories.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label>
              <span>Root path</span>
              <input
                value={componentPath}
                onChange={(event) => setComponentPath(event.target.value)}
              />
            </label>
          </div>
          <div className="button-row">
            <Button tone="accent" busy={busy === "component"}>
              登记
            </Button>
          </div>
        </form>
      ) : null}
      {form === "policy" ? (
        <form className="create-surface compact-create" onSubmit={addPolicy}>
          <div className="surface-heading">
            <strong>新增 Policy Bundle</strong>
            <Button type="button" onClick={() => setForm("")}>取消</Button>
          </div>
          <div className="form-grid form-grid-wide">
            <label>
              <span>名称</span>
              <input
                value={policyName}
                onChange={(event) => setPolicyName(event.target.value)}
                required
              />
            </label>
            <label>
              <span>层级</span>
              <select
                value={policyScope}
                onChange={(event) => {
                  const next = event.target.value as TeamPolicyBundle["scope"];
                  setPolicyScope(next);
                  setPolicyScopeID(
                    next === "team"
                      ? (currentProject?.team_id ?? "")
                      : next === "project"
                        ? projectID
                        : "",
                  );
                }}
              >
                {["team", "project", "repository", "component"].map((value) => (
                  <option key={value}>{value}</option>
                ))}
              </select>
            </label>
            <label>
              <span>Scope ID</span>
              <input
                value={policyScopeID}
                onChange={(event) => setPolicyScopeID(event.target.value)}
                required
              />
            </label>
            <label>
              <span>版本</span>
              <input
                type="number"
                min={1}
                value={policyVersion}
                onChange={(event) =>
                  setPolicyVersion(Number(event.target.value))
                }
              />
            </label>
            <label className="field-span-full">
              <span>Rules JSON</span>
              <textarea
                rows={7}
                value={policyRules}
                onChange={(event) => setPolicyRules(event.target.value)}
              />
            </label>
          </div>
          <div className="button-row">
            <Button tone="accent" busy={busy === "policy"}>
              保存策略层
            </Button>
          </div>
        </form>
      ) : null}

      {tab === "people" ? (
        <section className="collection-surface">
          {members.length === 0 ? (
            <Empty title="尚无项目成员" />
          ) : (
            <div className="member-directory">
              {members.map((member) => {
                const utilization = Math.max(
                  0,
                  Math.min(100, member.capacity?.utilization_percent ?? 0),
                );
                return (
                  <article key={member.id}>
                    <div className="avatar">
                      {(member.display_name || member.id)
                        .slice(0, 1)
                        .toUpperCase()}
                    </div>
                    <div className="member-content">
                      <div>
                        <strong>{member.display_name || member.id}</strong>
                        <Status tone={toneForState(member.status)}>
                          {member.status}
                        </Status>
                      </div>
                      <small>
                        {member.role} ·{" "}
                        {(member.business_domains ?? []).join(" / ") ||
                          "未设置业务域"}
                      </small>
                      <progress
                        className="progress-track"
                        max={100}
                        value={utilization}
                      />
                      <p>
                        {member.capacity?.active_work ?? 0} 进行中 ·{" "}
                        {member.capacity?.queued_work ?? 0} 排队 ·{" "}
                        {member.capacity?.blocked_work ?? 0} 阻塞 ·{" "}
                        {utilization}%
                      </p>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </section>
      ) : null}
      {tab === "runners" ? (
        <section className="collection-surface">
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Runner</th>
                  <th>Owner</th>
                  <th>能力</th>
                  <th>当前任务</th>
                  <th>状态</th>
                  <th>心跳</th>
                </tr>
              </thead>
              <tbody>
                {runners.map((runner) => (
                  <tr key={runner.id}>
                    <td>
                      <strong>{runner.display_name || runner.id}</strong>
                      <small>{runner.id}</small>
                    </td>
                    <td>
                      {names.get(runner.member_id ?? "") ||
                        runner.member_id ||
                        "未绑定"}
                    </td>
                    <td>{(runner.capabilities ?? []).join(" · ") || "—"}</td>
                    <td>{runner.current_work_id || "空闲"}</td>
                    <td>
                      <Status tone={toneForState(runner.status)}>
                        {runner.status}
                      </Status>
                    </td>
                    <td>{formatRelative(runner.last_seen_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {runners.length === 0 ? (
            <Empty
              title="没有已登记 Runner"
              detail="Runner 注册和 device key 操作必须在成员自己的工作站执行。"
            />
          ) : null}
          <div className="subsection">
            <div className="surface-heading">
              <strong>活动队列</strong>
              <small>{activeRunnerTasks.length} 个任务</small>
            </div>
            {activeRunnerTasks.length ? (
              <div className="stack-list">
                {activeRunnerTasks.map((task) => (
                  <article className="stack-item" key={task.id}>
                    <div>
                      <strong>{task.id}</strong>
                      <small>
                        {task.runner_id || "尚未领取"} · attempt{" "}
                        {task.attempt ?? 0}
                      </small>
                    </div>
                    <Status tone={toneForState(task.status)}>
                      {task.status}
                    </Status>
                  </article>
                ))}
              </div>
            ) : (
              <Empty title="队列为空" />
            )}
          </div>
        </section>
      ) : null}
      {tab === "assets" ? (
        <div className="asset-columns">
          <section className="collection-surface">
            <div className="surface-heading">
              <strong>Repositories</strong>
              <Button onClick={() => setForm("repository")}>登记</Button>
            </div>
            {repositories.length ? (
              <div className="stack-list">
                {repositories.map((item) => (
                  <article className="stack-item" key={item.id}>
                    <div>
                      <strong>{item.name}</strong>
                      <small>
                        {item.remote_url || item.local_path || item.id}
                      </small>
                    </div>
                    <Status>{item.default_branch}</Status>
                  </article>
                ))}
              </div>
            ) : (
              <Empty title="尚无仓库" />
            )}
          </section>
          <section className="collection-surface">
            <div className="surface-heading">
              <strong>Documents</strong>
              <Button onClick={() => setForm("document")}>登记</Button>
            </div>
            {documents.length ? (
              <div className="stack-list">
                {documents.map((item) => (
                  <article className="stack-item" key={item.id}>
                    <div>
                      <strong>{item.title}</strong>
                      <small>
                        {item.key} · {item.uri}
                      </small>
                    </div>
                    <Status tone={toneForState(item.status)}>
                      {item.kind} · {item.status}
                    </Status>
                  </article>
                ))}
              </div>
            ) : (
              <Empty title="尚无方案文档" />
            )}
          </section>
          <section className="collection-surface">
            <div className="surface-heading">
              <strong>Components</strong>
              <Button onClick={() => setForm("component")}>登记</Button>
            </div>
            {components.length ? (
              <div className="stack-list">
                {components.map((item) => (
                  <article className="stack-item" key={item.id}>
                    <div>
                      <strong>{item.name}</strong>
                      <small>
                        {item.repository_id || "未绑定仓库"} ·{" "}
                        {item.root_path || "/"}
                      </small>
                    </div>
                    <Status>{item.kind}</Status>
                  </article>
                ))}
              </div>
            ) : (
              <Empty title="尚无共享组件" />
            )}
          </section>
        </div>
      ) : null}
      {tab === "policy" ? (
        <div className="asset-columns">
          <section className="collection-surface">
            <div className="surface-heading">
              <strong>有效策略</strong>
              <Status tone={policyStatus.compliant ? "success" : "danger"}>
                {policyStatus.compliant
                  ? "一致"
                  : `${policyStatus.drift_count} 项漂移`}
              </Status>
            </div>
            <dl className="property-list policy-summary">
              <div>
                <dt>Effective hash</dt>
                <dd>
                  <code>{policyStatus.effective_version || "未解析"}</code>
                </dd>
              </div>
              <div>
                <dt>Layers</dt>
                <dd>{policyStatus.layers?.length ?? 0}</dd>
              </div>
              <div>
                <dt>Checked</dt>
                <dd>{formatRelative(policyStatus.checked_at)}</dd>
              </div>
            </dl>
          </section>
          <section className="collection-surface span-two">
            <div className="surface-heading">
              <strong>Policy Bundles</strong>
              <Button onClick={() => setForm("policy")}>新增</Button>
            </div>
            {policies.length ? (
              <div className="table-wrap">
                <table>
                  <thead>
                    <tr>
                      <th>策略</th>
                      <th>层级</th>
                      <th>版本</th>
                      <th>优先级</th>
                      <th>状态</th>
                      <th>Hash</th>
                    </tr>
                  </thead>
                  <tbody>
                    {policies.map((item) => (
                      <tr key={item.id}>
                        <td>
                          <strong>{item.name}</strong>
                          <small>{item.id}</small>
                        </td>
                        <td>
                          {item.scope} · {item.scope_id}
                        </td>
                        <td>{item.version}</td>
                        <td>{item.priority}</td>
                        <td>
                          <Status tone={item.enabled ? "success" : "neutral"}>
                            {item.enabled ? "enabled" : "disabled"}
                          </Status>
                        </td>
                        <td>
                          <code>{item.hash.slice(0, 12)}</code>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <Empty title="尚无策略层" />
            )}
          </section>
        </div>
      ) : null}
    </div>
  );
}
