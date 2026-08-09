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
import {
  DeliveryIntegrity,
  DeliveryProjection,
  DeliveryRequest,
  DeliveryReviewKind,
  SolutionSpec,
} from "./types";
import { useAsyncData } from "./use-data";

interface RequirementData {
  projection: DeliveryProjection;
  integrity: DeliveryIntegrity;
}

const reviewKinds: DeliveryReviewKind[] = [
  "scenario",
  "capacity",
  "risk",
  "cost",
];

const lines = (value: string) =>
  value
    .split("\n")
    .map((item) => item.trim())
    .filter(Boolean);

const commandID = (kind: string) =>
  `ui-${kind}-${crypto.randomUUID().replace(/-/g, "")}`;

export function SpecPage() {
  const { client, projectID } = useTeam();
  const [selectedID, setSelectedID] = useState("");
  const [busy, setBusy] = useState("");
  const [actionError, setActionError] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [criteria, setCriteria] = useState("");
  const [constraints, setConstraints] = useState("");
  const [goal, setGoal] = useState("");
  const [solutionTitle, setSolutionTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [testStrategy, setTestStrategy] = useState("");
  const [rollbackPlan, setRollbackPlan] = useState("");
  const [reviewComment, setReviewComment] = useState("");
  const [workTitle, setWorkTitle] = useState("");
  const [workInstructions, setWorkInstructions] = useState("");
  const [verification, setVerification] = useState("go test ./...");
  const [evidence, setEvidence] = useState("test report");
  const [changeReason, setChangeReason] = useState("");
  const [changeImpact, setChangeImpact] = useState("");

  const load = useCallback(async (): Promise<RequirementData> => {
    const [projection, integrity] = await Promise.all([
      client.rpc<DeliveryProjection>("delivery.projection", {
        project_id: projectID,
      }),
      client.rpc<DeliveryIntegrity>("delivery.integrity", {
        project_id: projectID,
      }),
    ]);
    return { projection, integrity };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  const mutate = async (
    key: string,
    type: string,
    payload: Record<string, unknown>,
    done?: () => void,
  ) => {
    if (!state.data) return;
    setBusy(key);
    setActionError("");
    try {
      await client.rpc("delivery.command", {
        id: commandID(key),
        project_id: projectID,
        type,
        expected_revision: state.data.projection.revision,
        payload,
      });
      done?.();
      state.reload();
    } catch (reason) {
      setActionError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy("");
    }
  };

  const createRequest = (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim() || !description.trim()) return;
    void mutate(
      "request",
      "request.create",
      {
        title: title.trim(),
        description: description.trim(),
        acceptance_criteria: lines(criteria),
        constraints: lines(constraints),
      },
      () => {
        setTitle("");
        setDescription("");
        setCriteria("");
        setConstraints("");
        setCreateOpen(false);
      },
    );
  };

  if (state.loading && !state.data)
    return <Loading label="加载需求、方案与冻结计划…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;

  const { projection, integrity } = state.data;
  const requests = Object.values(projection.requests).sort((a, b) =>
    b.updated_at.localeCompare(a.updated_at),
  );
  const selected =
    projection.requests[selectedID] ?? requests[0] ?? null;
  const intent = selected
    ? Object.values(projection.intents).find(
        (item) =>
          item.request_id === selected.id && item.revision === selected.revision,
      )
    : undefined;
  const solution = selected
    ? Object.values(projection.solutions).find(
        (item) =>
          item.request_id === selected.id && item.revision === selected.revision,
      )
    : undefined;
  const plan = solution
    ? Object.values(projection.frozen_plans).find(
        (item) => item.solution_id === solution.id,
      )
    : undefined;
  const changes = selected
    ? Object.values(projection.change_intents).filter(
        (item) => item.request_id === selected.id,
      )
    : [];

  return (
    <div className="collection-page">
      <PageHeader
        title="需求、方案与任务"
        description="Request → IntentContract → SolutionSpec/ADR → 四审 → Freeze → WorkItem；冻结后变更只能生成 ChangeIntent。"
        actions={
          <Button tone="accent" onClick={() => setCreateOpen((value) => !value)}>
            新建需求
          </Button>
        }
      />
      <div className="summary-strip">
        <span>
          <strong>{requests.length}</strong>
          <small>需求</small>
        </span>
        <span>
          <strong>
            {
              Object.values(projection.solutions).filter(
                (item) => item.status === "approved",
              ).length
            }
          </strong>
          <small>待冻结方案</small>
        </span>
        <span>
          <strong>{Object.keys(projection.frozen_plans).length}</strong>
          <small>冻结计划</small>
        </span>
        <span>
          <strong>{projection.revision}</strong>
          <small>投影 Revision</small>
        </span>
        <span>
          <strong>{integrity.projection_stable ? "一致" : "异常"}</strong>
          <small>{integrity.event_count} 个事件</small>
        </span>
      </div>

      {actionError ? (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      ) : null}

      {createOpen ? (
        <form className="create-surface" onSubmit={createRequest}>
          <div className="surface-heading">
            <strong>登记 Request</strong>
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
                required
              />
            </label>
            <label className="span-two">
              <span>问题与目标</span>
              <textarea
                rows={3}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                required
              />
            </label>
            <label>
              <span>验收标准（每行一条）</span>
              <textarea
                rows={3}
                value={criteria}
                onChange={(event) => setCriteria(event.target.value)}
                required
              />
            </label>
            <label>
              <span>约束（每行一条）</span>
              <textarea
                rows={3}
                value={constraints}
                onChange={(event) => setConstraints(event.target.value)}
              />
            </label>
          </div>
          <Button
            tone="accent"
            busy={busy === "request"}
            disabled={!lines(criteria).length}
          >
            保存 Request
          </Button>
        </form>
      ) : null}

      <div className="master-detail">
        <div className="collection-list">
          {requests.length === 0 ? (
            <Empty
              title="还没有受治理需求"
              detail="登记 Request 后才能批准 IntentContract 和创建方案。"
            />
          ) : (
            requests.map((item) => (
              <button
                type="button"
                className={selected?.id === item.id ? "is-selected" : ""}
                key={item.id}
                onClick={() => setSelectedID(item.id)}
              >
                <span>
                  <strong>{item.title}</strong>
                  <small>
                    {item.id} · revision {item.revision} ·{" "}
                    {formatRelative(item.updated_at)}
                  </small>
                </span>
                <Status tone={toneForState(item.status)}>{item.status}</Status>
              </button>
            ))
          )}
        </div>

        {selected ? (
          <aside className="detail-panel">
            <div className="detail-heading">
              <div>
                <span className="eyebrow">Request</span>
                <h2>{selected.title}</h2>
                <small>{selected.id}</small>
              </div>
              <Status tone={toneForState(selected.status)}>
                {selected.status}
              </Status>
            </div>
            <p>{selected.description}</p>
            <RequirementSteps
              request={selected}
              solution={solution}
              intentID={intent?.id}
              planID={plan?.id}
            />

            {!intent && selected.status === "draft" ? (
              <section>
                <h3>批准 IntentContract</h3>
                <label>
                  <span>冻结目标</span>
                  <textarea
                    rows={3}
                    value={goal}
                    onChange={(event) => setGoal(event.target.value)}
                    placeholder={selected.description}
                  />
                </label>
                <Button
                  tone="accent"
                  busy={busy === "intent"}
                  disabled={!goal.trim() || !(selected.acceptance_criteria ?? []).length}
                  onClick={() =>
                    void mutate("intent", "intent.approve", {
                      request_id: selected.id,
                      goal: goal.trim(),
                      acceptance_criteria: selected.acceptance_criteria,
                      non_goals: selected.non_goals ?? [],
                      constraints: selected.constraints ?? [],
                    })
                  }
                >
                  人工批准 Intent
                </Button>
              </section>
            ) : null}

            {intent && !solution ? (
              <section>
                <h3>创建 SolutionSpec / ADR</h3>
                <div className="form-grid">
                  <label>
                    <span>方案标题</span>
                    <input
                      value={solutionTitle}
                      onChange={(event) => setSolutionTitle(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>方案摘要</span>
                    <textarea
                      rows={3}
                      value={summary}
                      onChange={(event) => setSummary(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>测试策略（每行一条）</span>
                    <textarea
                      rows={3}
                      value={testStrategy}
                      onChange={(event) => setTestStrategy(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>回滚方案</span>
                    <textarea
                      rows={3}
                      value={rollbackPlan}
                      onChange={(event) => setRollbackPlan(event.target.value)}
                    />
                  </label>
                </div>
                <Button
                  tone="accent"
                  busy={busy === "solution"}
                  disabled={
                    !solutionTitle.trim() ||
                    !summary.trim() ||
                    !testStrategy.trim() ||
                    !rollbackPlan.trim()
                  }
                  onClick={() =>
                    void mutate("solution", "solution.create", {
                      request_id: selected.id,
                      intent_id: intent.id,
                      title: solutionTitle.trim(),
                      summary: summary.trim(),
                      test_strategy: lines(testStrategy),
                      rollback_plan: rollbackPlan.trim(),
                    })
                  }
                >
                  提交方案
                </Button>
              </section>
            ) : null}

            {solution && solution.status !== "frozen" ? (
              <section>
                <h3>Scenario / Capacity / Risk / Cost 四审</h3>
                <input
                  value={reviewComment}
                  onChange={(event) => setReviewComment(event.target.value)}
                  placeholder="评审意见"
                />
                <div className="review-checks">
                  {reviewKinds.map((kind) => {
                    const review = solution.reviews[kind];
                    return (
                      <div className="control-row" key={kind}>
                        <Status tone={toneForState(review?.decision)}>
                          {kind} · {review?.decision ?? "pending"}
                        </Status>
                        <Button
                          busy={busy === `review-${kind}`}
                          disabled={review?.decision === "approved"}
                          onClick={() =>
                            void mutate(
                              `review-${kind}`,
                              "solution.review.decide",
                              {
                                solution_id: solution.id,
                                kind,
                                decision: "approved",
                                comment: reviewComment.trim(),
                              },
                            )
                          }
                        >
                          批准
                        </Button>
                      </div>
                    );
                  })}
                </div>
              </section>
            ) : null}

            {solution?.status === "approved" ? (
              <section>
                <h3>Freeze 并生成 WorkItem</h3>
                <div className="form-grid">
                  <label>
                    <span>任务标题</span>
                    <input
                      value={workTitle}
                      onChange={(event) => setWorkTitle(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>实施说明</span>
                    <textarea
                      rows={3}
                      value={workInstructions}
                      onChange={(event) => setWorkInstructions(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>验证命令</span>
                    <input
                      value={verification}
                      onChange={(event) => setVerification(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>证据要求</span>
                    <input
                      value={evidence}
                      onChange={(event) => setEvidence(event.target.value)}
                    />
                  </label>
                </div>
                <Button
                  tone="accent"
                  busy={busy === "freeze"}
                  disabled={!workTitle.trim() || !workInstructions.trim()}
                  onClick={() =>
                    void mutate("freeze", "plan.freeze", {
                      solution_id: solution.id,
                      work_items: [
                        {
                          title: workTitle.trim(),
                          instructions: workInstructions.trim(),
                          priority: "p2",
                          risk_level: "medium",
                          verification_commands: [
                            verification.trim().split(/\s+/).filter(Boolean),
                          ],
                          evidence_requirements: lines(evidence),
                        },
                      ],
                    })
                  }
                >
                  人工 Freeze
                </Button>
              </section>
            ) : null}

            {plan && selected.status === "frozen" ? (
              <section>
                <h3>冻结 Bundle</h3>
                <code>{plan.bundle_hash}</code>
                <small>{plan.work_item_ids.join(" · ")}</small>
                <div className="form-grid">
                  <label>
                    <span>变更原因</span>
                    <textarea
                      rows={2}
                      value={changeReason}
                      onChange={(event) => setChangeReason(event.target.value)}
                    />
                  </label>
                  <label>
                    <span>影响分析</span>
                    <textarea
                      rows={2}
                      value={changeImpact}
                      onChange={(event) => setChangeImpact(event.target.value)}
                    />
                  </label>
                </div>
                <Button
                  busy={busy === "change"}
                  disabled={!changeReason.trim() || !changeImpact.trim()}
                  onClick={() =>
                    void mutate("change", "change_intent.create", {
                      request_id: selected.id,
                      frozen_plan_id: plan.id,
                      reason: changeReason.trim(),
                      impact: changeImpact.trim(),
                    })
                  }
                >
                  创建 ChangeIntent
                </Button>
              </section>
            ) : null}

            {changes.length ? (
              <section>
                <h3>ChangeIntent</h3>
                {changes.map((item) => (
                  <article className="stack-item" key={item.id}>
                    <div>
                      <strong>{item.reason}</strong>
                      <small>{item.impact}</small>
                    </div>
                    <Status tone={toneForState(item.status)}>
                      {item.status}
                    </Status>
                  </article>
                ))}
              </section>
            ) : null}
          </aside>
        ) : null}
      </div>
    </div>
  );
}

function RequirementSteps({
  request,
  solution,
  intentID,
  planID,
}: {
  request: DeliveryRequest;
  solution?: SolutionSpec;
  intentID?: string;
  planID?: string;
}) {
  const steps = [
    ["Request", request.id, true],
    ["Intent", intentID ?? "待批准", Boolean(intentID)],
    ["Solution", solution?.id ?? "待创建", Boolean(solution)],
    ["四审", solution?.status ?? "pending", solution?.status === "approved" || solution?.status === "frozen"],
    ["Freeze", planID ?? "待冻结", Boolean(planID)],
  ] as const;
  return (
    <div className="review-checks">
      {steps.map(([label, value, complete]) => (
        <Status key={label} tone={complete ? "success" : "warning"}>
          {label} · {value}
        </Status>
      ))}
    </div>
  );
}
