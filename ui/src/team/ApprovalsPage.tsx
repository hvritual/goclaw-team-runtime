import { useCallback, useState } from 'react';
import { GovernanceDecision, GovernanceDecisionValue } from './GovernanceDecision';
import { useTeam } from './context';
import {
  Button,
  Empty,
  ErrorState,
  formatRelative,
  Loading,
  Metric,
  PageHeader,
  Status,
  toneForState,
} from './primitives';
import {
  CatalogMemoryRecord,
  DevReviewKind,
  DevTask,
  HarnessExperiment,
  KnowledgeProposal,
  OuroborosSession,
} from './types';
import { useAsyncData } from './use-data';

interface ApprovalData {
  proposals: KnowledgeProposal[];
  memories: CatalogMemoryRecord[];
  experiments: HarnessExperiment[];
  tasks: DevTask[];
  specs: OuroborosSession[];
}

const reviewLabels: Record<DevReviewKind, string> = {
  scenario: '场景评审',
  capacity: '容量评审',
  risk: '风险评审',
  cost: '成本评审',
};

export function ApprovalsPage() {
  const { client, governance, projectID, session } = useTeam();
  const [actionError, setActionError] = useState('');
  const [busy, setBusy] = useState('');
  const load = useCallback(async (): Promise<ApprovalData> => {
    const [proposals, memories, experiments, tasks, specs] = await Promise.all([
      client.rpc<KnowledgeProposal[]>('knowledge.proposals', { status: 'pending' }),
      client.rpc<CatalogMemoryRecord[]>('memory.catalog.list', { project_id: projectID, status: 'pending', limit: 100 }),
      client.rpc<HarnessExperiment[]>('harness.experiments'),
      client.rpc<DevTask[]>('dev.tasks', { project_id: projectID }),
      client.rpc<OuroborosSession[]>('ouroboros.sessions', { project_id: projectID }),
    ]);
    return { proposals, memories, experiments, tasks, specs };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  const governed = (value: GovernanceDecisionValue, refs: string[]) => governance({
    rationale: value.rationale,
    counterargument: value.counterargument,
    evidenceRefs: [...refs, ...value.evidenceRefs],
  });
  const act = async (key: string, action: () => Promise<unknown>) => {
    setBusy(key);
    setActionError('');
    try {
      await action();
      state.reload();
    } catch (reason) {
      setActionError(reason instanceof Error ? reason.message : String(reason));
      throw reason;
    } finally {
      setBusy('');
    }
  };

  if (state.loading && !state.data) return <Loading label="加载治理审批队列…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;

  const { proposals, memories, experiments, tasks, specs } = state.data;
  const harnessQueue = experiments.filter((item) => ['validated', 'human_approved'].includes(item.status));
  const devQueue = tasks.filter((item) => ['review_pending', 'blocked', 'ready_to_freeze', 'awaiting_acceptance'].includes(item.status));
  const seedQueue = specs.filter((item) => item.status === 'awaiting_seed_approval');
  const total = proposals.length + memories.length + harnessQueue.length + devQueue.length + seedQueue.length;

  return (
    <>
      <PageHeader title="审批" description="审批人、执行人和最终验收人职责分离；每次决策都必须记录理由、反方论点和证据。" />
      <div className="metric-grid compact">
        <Metric label="总待办" value={total} tone={total ? 'warning' : 'success'} />
        <Metric label="知识与记忆" value={proposals.length + memories.length} />
        <Metric label="规格与开发" value={seedQueue.length + devQueue.length} />
        <Metric label="Harness" value={harnessQueue.length} />
      </div>
      {actionError ? <p className="inline-error">{actionError}</p> : null}
      <div className="approval-list">
        {proposals.map((item) => (
          <article className="approval-card panel" key={item.id}>
            <ApprovalHeading type="知识写入提案" title={item.target_path} state={item.status} detail={`${item.created_by} · ${formatRelative(item.created_at)}`} />
            <p>{item.reason}</p>
            <div className="safety-note">基于创建时内容 SHA {item.base_sha256?.slice(0, 12) || 'new'}；批准时会重新检查冲突。</div>
            <GovernanceDecision
              onApprove={(value) => act(item.id, () => client.rpc('knowledge.proposal.approve', { id: item.id, ...governed(value, [`knowledge-proposal:${item.id}`]) }))}
              onReject={(value) => act(item.id, () => client.rpc('knowledge.proposal.reject', { id: item.id, ...governed(value, [`knowledge-proposal:${item.id}`]) }))}
            />
          </article>
        ))}
        {memories.map((item) => (
          <article className="approval-card panel" key={item.id}>
            <ApprovalHeading type="记忆候选" title={item.title} state={item.status} detail={`${item.kind} · v${item.version} · ${item.provenance.source_uri}`} />
            <p>{item.abstract || item.content}</p>
            <div className="safety-note">候选在批准前不会进入自动检索上下文。</div>
            <GovernanceDecision
              approveLabel="批准入藏"
              onApprove={(value) => act(item.id, () => client.rpc('memory.catalog.candidate.approve', { id: item.id, ...governed(value, [`catalog:${item.id}@v${item.version}`]) }))}
              onReject={(value) => act(item.id, () => client.rpc('memory.catalog.candidate.reject', { id: item.id, ...governed(value, [`catalog:${item.id}@v${item.version}`]) }))}
            />
          </article>
        ))}
        {seedQueue.map((item) => (
          <article className="approval-card panel" key={item.id}>
            <ApprovalHeading type="不可变 Seed" title={item.title} state={item.status} detail={`${item.id} · ${item.pending_seed_hash?.slice(0, 12) || 'hash pending'}`} />
            <p>{item.raw_request}</p>
            <GovernanceDecision
              onApprove={(value) => act(item.id, () => client.rpc('ouroboros.seed.approve', { id: item.id, ...governed(value, [`session:${item.id}`, `seed:${item.pending_seed_hash ?? ''}`]) }))}
              onReject={(value) => act(item.id, () => client.rpc('ouroboros.seed.reject', { id: item.id, ...governed(value, [`session:${item.id}`]) }))}
            />
          </article>
        ))}
        {devQueue.map((task) => {
          const missing = (Object.keys(reviewLabels) as DevReviewKind[]).find((kind) => task.reviews[kind]?.decision !== 'approved');
          return (
            <article className="approval-card panel" key={task.id}>
              <ApprovalHeading type="开发任务" title={task.title} state={task.status} detail={`${task.id} · revision ${task.compile.revision}`} />
              <p>{task.goal.objective}</p>
              <div className="review-chip-grid">
                {(Object.keys(reviewLabels) as DevReviewKind[]).map((kind) => (
                  <Status key={kind} tone={toneForState(task.reviews[kind]?.decision)}>{reviewLabels[kind]} · {task.reviews[kind]?.decision ?? 'pending'}</Status>
                ))}
              </div>
              <div className="safety-note">变更上限：{task.scope.max_changed_files} 个文件 / {task.scope.max_changed_lines} 行</div>
              {missing && ['review_pending', 'blocked'].includes(task.status) ? (
                <GovernanceDecision
                  approveLabel={`批准${reviewLabels[missing]}`}
                  rejectLabel={`拒绝${reviewLabels[missing]}`}
                  onApprove={(value) => act(task.id, () => client.rpc('dev.task.review', { id: task.id, kind: missing, decision: 'approved', ...governed(value, [`task:${task.id}`, `review-kind:${missing}`]) }))}
                  onReject={(value) => act(task.id, () => client.rpc('dev.task.review', { id: task.id, kind: missing, decision: 'rejected', ...governed(value, [`task:${task.id}`, `review-kind:${missing}`]) }))}
                />
              ) : null}
              {task.status === 'ready_to_freeze' ? (
                <div className="button-row">
                  <Button tone="accent" busy={busy === task.id} onClick={() => void act(task.id, () => client.rpc('dev.task.freeze', { id: task.id, actor: session?.principal_id }))}>冻结执行包</Button>
                </div>
              ) : null}
              {task.status === 'awaiting_acceptance' ? (
                <GovernanceDecision
                  approveLabel="最终验收"
                  rejectLabel="拒绝验收"
                  onApprove={(value) => act(task.id, () => client.rpc('dev.task.accept', { id: task.id, ...governed(value, [`task:${task.id}`, task.last_evidence ?? 'evidence:missing']) }))}
                  onReject={(value) => act(task.id, () => client.rpc('dev.task.revise', { id: task.id, expected_revision: task.compile.revision, reason: `验收拒绝：${value.rationale}` }))}
                />
              ) : null}
            </article>
          );
        })}
        {harnessQueue.map((item) => (
          <article className="approval-card panel" key={item.id}>
            <ApprovalHeading type="Harness 实验" title={item.candidate_version} state={item.status} detail={`${item.base_version} → ${item.candidate_version}`} />
            <p>{item.change_manifest.change_summary}</p>
            <div className="safety-note">根因：{item.change_manifest.root_cause}</div>
            <GovernanceDecision
              approveLabel={item.status === 'validated' ? '批准实验' : '提升为生效版本'}
              onApprove={(value) => act(item.id, () => client.rpc(item.status === 'validated' ? 'harness.experiment.approve' : 'harness.experiment.promote', { id: item.id, ...governed(value, [`experiment:${item.id}`]) }))}
              onReject={(value) => act(item.id, () => client.rpc('harness.experiment.reject', { id: item.id, ...governed(value, [`experiment:${item.id}`]) }))}
            />
          </article>
        ))}
        {total === 0 ? <Empty title="审批队列已清空" detail="没有任何候选被自动批准；新提案会在这里等待人工治理。" /> : null}
      </div>
    </>
  );
}

function ApprovalHeading({
  type,
  title,
  state,
  detail,
}: {
  type: string;
  title: string;
  state: string;
  detail: string;
}) {
  return (
    <div className="card-heading">
      <div><span className="eyebrow">{type}</span><strong>{title}</strong><small>{detail}</small></div>
      <Status tone={toneForState(state)}>{state}</Status>
    </div>
  );
}
