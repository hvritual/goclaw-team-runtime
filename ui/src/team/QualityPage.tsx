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
  DeliveryDefect,
  DeliveryProjection,
  DeliveryRisk,
  ResourceType,
} from "./types";
import { useAsyncData } from "./use-data";

type Lane = "defect" | "risk";

const commandID = (kind: string) =>
  `ui-${kind}-${crypto.randomUUID().replace(/-/g, "")}`;
const entityID = (kind: string) =>
  `${kind}-${crypto.randomUUID().replace(/-/g, "")}`;

export function QualityPage() {
  const { client, projectID, session } = useTeam();
  const [lane, setLane] = useState<Lane>("defect");
  const [selectedID, setSelectedID] = useState("");
  const [createOpen, setCreateOpen] = useState(false);
  const [busy, setBusy] = useState("");
  const [actionError, setActionError] = useState("");
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [severity, setSeverity] =
    useState<DeliveryDefect["severity"]>("medium");
  const [priority, setPriority] =
    useState<DeliveryDefect["priority"]>("p2");
  const [environment, setEnvironment] = useState("");
  const [module, setModule] = useState("");
  const [probability, setProbability] =
    useState<DeliveryRisk["probability"]>("medium");
  const [impact, setImpact] = useState<DeliveryRisk["impact"]>("medium");
  const [trigger, setTrigger] = useState("");
  const [reproduction, setReproduction] = useState("");
  const [expected, setExpected] = useState("");
  const [actual, setActual] = useState("");
  const [rootCause, setRootCause] = useState("");
  const [resolution, setResolution] = useState("");
  const [workItemIDs, setWorkItemIDs] = useState("");
  const [response, setResponse] =
    useState<NonNullable<DeliveryRisk["response"]>>("mitigate");
  const [responsePlan, setResponsePlan] = useState("");
  const [acceptanceReason, setAcceptanceReason] = useState("");
  const [reviewAt, setReviewAt] = useState("");
  const [evidenceURI, setEvidenceURI] = useState("");
  const [lastEvidenceID, setLastEvidenceID] = useState("");

  useEffect(() => {
    setLastEvidenceID("");
  }, [lane, selectedID]);

  const load = useCallback(
    () =>
      client.rpc<DeliveryProjection>("delivery.projection", {
        project_id: projectID,
      }),
    [client, projectID],
  );
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
        expected_revision: state.data.revision,
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

  const create = (event: FormEvent) => {
    event.preventDefault();
    if (!title.trim() || !description.trim()) return;
    const payload =
      lane === "defect"
        ? {
            title: title.trim(),
            description: description.trim(),
            severity,
            priority,
            environment: environment.trim(),
            module: module.trim(),
            owner_id: session?.principal_id,
          }
        : {
            title: title.trim(),
            description: description.trim(),
            probability,
            impact,
            trigger: trigger.trim(),
            owner_id: session?.principal_id,
          };
    void mutate(
      `create-${lane}`,
      lane === "defect" ? "defect.create" : "risk.create",
      payload,
      () => {
        setTitle("");
        setDescription("");
        setEnvironment("");
        setModule("");
        setTrigger("");
        setCreateOpen(false);
      },
    );
  };

  if (state.loading && !state.data)
    return <Loading label="加载 Defect、Risk 与证据…" />;
  if (state.error || !state.data)
    return <ErrorState error={state.error} onRetry={state.reload} />;

  const projection = state.data;
  const defects = Object.values(projection.defects).sort((a, b) =>
    b.updated_at.localeCompare(a.updated_at),
  );
  const risks = Object.values(projection.risks).sort((a, b) =>
    b.updated_at.localeCompare(a.updated_at),
  );
  const items: Array<DeliveryDefect | DeliveryRisk> =
    lane === "defect" ? defects : risks;
  const selected =
    items.find((item) => item.id === selectedID) ?? items[0] ?? null;

  const recordEvidence = (
    resourceType: ResourceType,
    resourceID: string,
  ) => {
    if (!evidenceURI.trim()) return;
    const id = entityID("evidence");
    void mutate(
      "evidence",
      "delivery.evidence.record",
      {
        id,
        resource_type: resourceType,
        resource_id: resourceID,
        kind: "verification",
        uri: evidenceURI.trim(),
        summary: "Team Control verification evidence",
      },
      () => {
        setLastEvidenceID(id);
        setEvidenceURI("");
      },
    );
  };

  return (
    <div className="collection-page">
      <PageHeader
        title="Bug 与风险"
        description="Defect 记录已经发生的问题；Risk 记录未来可能损失。两者拥有独立状态机、处置与证据门禁。"
        actions={
          <Button tone="accent" onClick={() => setCreateOpen((value) => !value)}>
            登记{lane === "defect" ? "缺陷" : "风险"}
          </Button>
        }
      />
      <div className="summary-strip">
        <span>
          <strong>
            {defects.filter((item) => item.status !== "closed").length}
          </strong>
          <small>活动 Defect</small>
        </span>
        <span>
          <strong>{risks.filter((item) => item.status !== "closed").length}</strong>
          <small>活动 Risk</small>
        </span>
        <span>
          <strong>
            {
              defects.filter((item) =>
                ["critical", "high"].includes(item.severity),
              ).length
            }
          </strong>
          <small>高严重度</small>
        </span>
        <span>
          <strong>
            {
              risks.filter(
                (item) => item.probability === "high" || item.impact === "high",
              ).length
            }
          </strong>
          <small>高风险</small>
        </span>
        <span>
          <strong>{Object.keys(projection.evidence).length}</strong>
          <small>Evidence</small>
        </span>
      </div>

      <div className="segmented-control control-tabs" role="tablist">
        <button
          className={lane === "defect" ? "is-active" : ""}
          onClick={() => {
            setLane("defect");
            setSelectedID("");
            setLastEvidenceID("");
          }}
        >
          Defect 生命周期
        </button>
        <button
          className={lane === "risk" ? "is-active" : ""}
          onClick={() => {
            setLane("risk");
            setSelectedID("");
            setLastEvidenceID("");
          }}
        >
          Risk 生命周期
        </button>
      </div>

      {actionError ? (
        <p className="inline-error" role="alert">
          {actionError}
        </p>
      ) : null}

      {createOpen ? (
        <form className="create-surface" onSubmit={create}>
          <div className="surface-heading">
            <strong>登记{lane === "defect" ? " Defect" : " Risk"}</strong>
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
              <span>说明</span>
              <textarea
                rows={3}
                value={description}
                onChange={(event) => setDescription(event.target.value)}
                required
              />
            </label>
            {lane === "defect" ? (
              <>
                <label>
                  <span>严重度</span>
                  <select
                    value={severity}
                    onChange={(event) =>
                      setSeverity(event.target.value as DeliveryDefect["severity"])
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
                      setPriority(event.target.value as DeliveryDefect["priority"])
                    }
                  >
                    {["p0", "p1", "p2", "p3", "p4"].map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>环境</span>
                  <input
                    value={environment}
                    onChange={(event) => setEnvironment(event.target.value)}
                  />
                </label>
                <label>
                  <span>模块</span>
                  <input
                    value={module}
                    onChange={(event) => setModule(event.target.value)}
                  />
                </label>
              </>
            ) : (
              <>
                <label>
                  <span>发生概率</span>
                  <select
                    value={probability}
                    onChange={(event) =>
                      setProbability(event.target.value as DeliveryRisk["probability"])
                    }
                  >
                    {["high", "medium", "low"].map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                </label>
                <label>
                  <span>影响</span>
                  <select
                    value={impact}
                    onChange={(event) =>
                      setImpact(event.target.value as DeliveryRisk["impact"])
                    }
                  >
                    {["high", "medium", "low"].map((value) => (
                      <option key={value}>{value}</option>
                    ))}
                  </select>
                </label>
                <label className="span-two">
                  <span>触发条件</span>
                  <input
                    value={trigger}
                    onChange={(event) => setTrigger(event.target.value)}
                    required
                  />
                </label>
              </>
            )}
          </div>
          <Button
            tone="accent"
            busy={busy === `create-${lane}`}
            disabled={lane === "risk" && !trigger.trim()}
          >
            保存
          </Button>
        </form>
      ) : null}

      <div className="master-detail">
        <div className="collection-list">
          {items.length === 0 ? (
            <Empty
              title={lane === "defect" ? "暂无 Defect" : "暂无 Risk"}
              detail="登记后，状态只能通过服务端允许的迁移推进。"
            />
          ) : (
            items.map((item) => (
              <button
                type="button"
                key={item.id}
                className={selected?.id === item.id ? "is-selected" : ""}
                onClick={() => {
                  setSelectedID(item.id);
                  setLastEvidenceID("");
                }}
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

        {selected && lane === "defect" ? (
          <DefectDetail
            item={selected as DeliveryDefect}
            busy={busy}
            reproduction={reproduction}
            expected={expected}
            actual={actual}
            rootCause={rootCause}
            resolution={resolution}
            workItemIDs={workItemIDs}
            evidenceURI={evidenceURI}
            lastEvidenceID={lastEvidenceID}
            setReproduction={setReproduction}
            setExpected={setExpected}
            setActual={setActual}
            setRootCause={setRootCause}
            setResolution={setResolution}
            setWorkItemIDs={setWorkItemIDs}
            setEvidenceURI={setEvidenceURI}
            transition={(status, extra = {}) =>
              void mutate("defect-transition", "defect.transition", {
                defect_id: selected.id,
                status,
                ...extra,
              })
            }
            recordEvidence={() => recordEvidence("defect", selected.id)}
          />
        ) : null}

        {selected && lane === "risk" ? (
          <RiskDetail
            item={selected as DeliveryRisk}
            busy={busy}
            response={response}
            responsePlan={responsePlan}
            acceptanceReason={acceptanceReason}
            reviewAt={reviewAt}
            workItemIDs={workItemIDs}
            evidenceURI={evidenceURI}
            lastEvidenceID={lastEvidenceID}
            setResponse={setResponse}
            setResponsePlan={setResponsePlan}
            setAcceptanceReason={setAcceptanceReason}
            setReviewAt={setReviewAt}
            setWorkItemIDs={setWorkItemIDs}
            setEvidenceURI={setEvidenceURI}
            assess={() =>
              void mutate("risk-assess", "risk.transition", {
                risk_id: selected.id,
                status: "assessed",
              })
            }
            decide={() =>
              void mutate("risk-response", "risk.response.decide", {
                risk_id: selected.id,
                response,
                response_plan: responsePlan.trim(),
                acceptance_reason: acceptanceReason.trim(),
                review_at: reviewAt
                  ? new Date(reviewAt).toISOString()
                  : undefined,
                work_item_ids: workItemIDs
                  .split(",")
                  .map((value) => value.trim())
                  .filter(Boolean),
              })
            }
            review={() =>
              void mutate("risk-review", "risk.transition", {
                risk_id: selected.id,
                status: "reviewed",
                evidence_ids: lastEvidenceID ? [lastEvidenceID] : [],
              })
            }
            close={() =>
              void mutate("risk-close", "risk.transition", {
                risk_id: selected.id,
                status: "closed",
              })
            }
            recordEvidence={() => recordEvidence("risk", selected.id)}
          />
        ) : null}
      </div>
    </div>
  );
}

interface DefectDetailProps {
  item: DeliveryDefect;
  busy: string;
  reproduction: string;
  expected: string;
  actual: string;
  rootCause: string;
  resolution: string;
  workItemIDs: string;
  evidenceURI: string;
  lastEvidenceID: string;
  setReproduction: (value: string) => void;
  setExpected: (value: string) => void;
  setActual: (value: string) => void;
  setRootCause: (value: string) => void;
  setResolution: (value: string) => void;
  setWorkItemIDs: (value: string) => void;
  setEvidenceURI: (value: string) => void;
  transition: (status: string, extra?: Record<string, unknown>) => void;
  recordEvidence: () => void;
}

function DefectDetail(props: DefectDetailProps) {
  const { item } = props;
  const workIDs = props.workItemIDs
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean);
  return (
    <aside className="detail-panel">
      <div className="detail-heading">
        <div>
          <span className="eyebrow">Defect</span>
          <h2>{item.title}</h2>
          <small>{item.id}</small>
        </div>
        <Status tone={toneForState(item.status)}>{item.status}</Status>
      </div>
      <p>{item.description}</p>
      <div className="review-checks">
        <Status tone={toneForState(item.severity)}>{item.severity}</Status>
        <Status>{item.priority}</Status>
        <Status>{item.owner_id || "未分配"}</Status>
      </div>
      {item.status === "reported" ? (
        <Button onClick={() => props.transition("confirmed")}>确认缺陷</Button>
      ) : null}
      {item.status === "confirmed" ? (
        <section>
          <h3>复现证据</h3>
          <textarea
            rows={2}
            placeholder="复现步骤"
            value={props.reproduction}
            onChange={(event) => props.setReproduction(event.target.value)}
          />
          <input
            placeholder="Expected"
            value={props.expected}
            onChange={(event) => props.setExpected(event.target.value)}
          />
          <input
            placeholder="Actual"
            value={props.actual}
            onChange={(event) => props.setActual(event.target.value)}
          />
          <Button
            disabled={
              !props.reproduction.trim() ||
              !props.expected.trim() ||
              !props.actual.trim()
            }
            onClick={() =>
              props.transition("reproduced", {
                reproduction: props.reproduction.trim(),
                expected: props.expected.trim(),
                actual: props.actual.trim(),
              })
            }
          >
            标记已复现
          </Button>
        </section>
      ) : null}
      {item.status === "reproduced" ? (
        <Button onClick={() => props.transition("classified")}>完成分级</Button>
      ) : null}
      {item.status === "classified" || item.status === "reopened" ? (
        <section>
          <h3>关联修复 WorkItem</h3>
          <input
            placeholder="work-1, work-2"
            value={props.workItemIDs}
            onChange={(event) => props.setWorkItemIDs(event.target.value)}
          />
          <Button
            disabled={!workIDs.length}
            onClick={() => props.transition("fixing", { work_item_ids: workIDs })}
          >
            开始修复
          </Button>
        </section>
      ) : null}
      {item.status === "fixing" ? (
        <section>
          <h3>根因与修复</h3>
          <textarea
            rows={2}
            placeholder="Root cause"
            value={props.rootCause}
            onChange={(event) => props.setRootCause(event.target.value)}
          />
          <textarea
            rows={2}
            placeholder="Resolution"
            value={props.resolution}
            onChange={(event) => props.setResolution(event.target.value)}
          />
          <Button
            disabled={!props.rootCause.trim() || !props.resolution.trim()}
            onClick={() =>
              props.transition("verifying", {
                root_cause: props.rootCause.trim(),
                resolution: props.resolution.trim(),
              })
            }
          >
            进入验证
          </Button>
        </section>
      ) : null}
      {item.status === "verifying" ? (
        <EvidenceInput
          value={props.evidenceURI}
          onChange={props.setEvidenceURI}
          onRecord={props.recordEvidence}
          evidenceID={props.lastEvidenceID}
          onAdvance={() =>
            props.transition("verified", {
              evidence_ids: props.lastEvidenceID
                ? [props.lastEvidenceID]
                : [],
            })
          }
          advanceLabel="验证通过"
        />
      ) : null}
      {item.status === "verified" ? (
        <Button onClick={() => props.transition("released")}>记录已发布</Button>
      ) : null}
      {item.status === "released" ? (
        <Button onClick={() => props.transition("closed")}>关闭 Defect</Button>
      ) : null}
    </aside>
  );
}

interface RiskDetailProps {
  item: DeliveryRisk;
  busy: string;
  response: NonNullable<DeliveryRisk["response"]>;
  responsePlan: string;
  acceptanceReason: string;
  reviewAt: string;
  workItemIDs: string;
  evidenceURI: string;
  lastEvidenceID: string;
  setResponse: (value: NonNullable<DeliveryRisk["response"]>) => void;
  setResponsePlan: (value: string) => void;
  setAcceptanceReason: (value: string) => void;
  setReviewAt: (value: string) => void;
  setWorkItemIDs: (value: string) => void;
  setEvidenceURI: (value: string) => void;
  assess: () => void;
  decide: () => void;
  review: () => void;
  close: () => void;
  recordEvidence: () => void;
}

function RiskDetail(props: RiskDetailProps) {
  const { item } = props;
  return (
    <aside className="detail-panel">
      <div className="detail-heading">
        <div>
          <span className="eyebrow">Risk</span>
          <h2>{item.title}</h2>
          <small>{item.id}</small>
        </div>
        <Status tone={toneForState(item.status)}>{item.status}</Status>
      </div>
      <p>{item.description}</p>
      <div className="review-checks">
        <Status>概率 · {item.probability}</Status>
        <Status>影响 · {item.impact}</Status>
        <Status>Owner · {item.owner_id}</Status>
      </div>
      <section>
        <h3>触发条件</h3>
        <p>{item.trigger}</p>
      </section>
      {item.status === "identified" ? (
        <Button onClick={props.assess}>完成风险评估</Button>
      ) : null}
      {item.status === "assessed" ? (
        <section>
          <h3>处置决策</h3>
          <select
            value={props.response}
            onChange={(event) =>
              props.setResponse(
                event.target.value as NonNullable<DeliveryRisk["response"]>,
              )
            }
          >
            {["avoid", "mitigate", "transfer", "accept", "monitor"].map(
              (value) => (
                <option key={value}>{value}</option>
              ),
            )}
          </select>
          <textarea
            rows={2}
            placeholder="处置计划"
            value={props.responsePlan}
            onChange={(event) => props.setResponsePlan(event.target.value)}
          />
          {props.response === "accept" ? (
            <textarea
              rows={2}
              placeholder="接受理由"
              value={props.acceptanceReason}
              onChange={(event) => props.setAcceptanceReason(event.target.value)}
            />
          ) : null}
          {["accept", "monitor"].includes(props.response) ? (
            <label>
              <span>复审时间</span>
              <input
                type="datetime-local"
                value={props.reviewAt}
                onChange={(event) => props.setReviewAt(event.target.value)}
              />
            </label>
          ) : null}
          {["avoid", "mitigate"].includes(props.response) ? (
            <input
              placeholder="关联 WorkItem IDs"
              value={props.workItemIDs}
              onChange={(event) => props.setWorkItemIDs(event.target.value)}
            />
          ) : null}
          <Button onClick={props.decide}>确认处置</Button>
        </section>
      ) : null}
      {item.status === "monitoring" || item.status === "mitigating" ? (
        <EvidenceInput
          value={props.evidenceURI}
          onChange={props.setEvidenceURI}
          onRecord={props.recordEvidence}
          evidenceID={props.lastEvidenceID}
          onAdvance={props.review}
          advanceLabel="完成复审"
        />
      ) : null}
      {item.status === "reviewed" ? (
        <Button onClick={props.close}>关闭 Risk</Button>
      ) : null}
      {item.response ? (
        <section>
          <h3>当前处置</h3>
          <p>
            {item.response} · {item.response_plan || item.acceptance_reason}
          </p>
          <small>{item.review_at ? formatRelative(item.review_at) : ""}</small>
        </section>
      ) : null}
    </aside>
  );
}

function EvidenceInput({
  value,
  onChange,
  onRecord,
  evidenceID,
  onAdvance,
  advanceLabel,
}: {
  value: string;
  onChange: (value: string) => void;
  onRecord: () => void;
  evidenceID: string;
  onAdvance: () => void;
  advanceLabel: string;
}) {
  return (
    <section>
      <h3>Evidence</h3>
      <input
        value={value}
        onChange={(event) => onChange(event.target.value)}
        placeholder="https://… 或绝对文件 URI"
      />
      <div className="button-row">
        <Button disabled={!value.trim()} onClick={onRecord}>
          登记证据
        </Button>
        <Button disabled={!evidenceID} tone="accent" onClick={onAdvance}>
          {advanceLabel}
        </Button>
      </div>
      {evidenceID ? <small>已登记：{evidenceID}</small> : null}
    </section>
  );
}
