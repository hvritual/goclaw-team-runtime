import { useCallback } from 'react';
import { GovernanceDecision, GovernanceDecisionValue } from './GovernanceDecision';
import { useTeam } from './context';
import {
  Empty,
  ErrorState,
  formatRelative,
  Loading,
  Metric,
  PageHeader,
  Section,
  Status,
  toneForState,
} from './primitives';
import { HarnessExperiment, HarnessStatus, HarnessTrace } from './types';
import { useAsyncData } from './use-data';

interface HarnessData {
  status: HarnessStatus;
  experiments: HarnessExperiment[];
  traces: HarnessTrace[];
}

export function HarnessPage() {
  const { client, governance, projectID } = useTeam();
  const load = useCallback(async (): Promise<HarnessData> => {
    const [status, experiments, traces] = await Promise.all([
      client.rpc<HarnessStatus>('harness.status'),
      client.rpc<HarnessExperiment[]>('harness.experiments'),
      client.rpc<HarnessTrace[]>('harness.traces', { project_id: projectID, limit: 20 }),
    ]);
    return { status, experiments, traces };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);
  const payload = (value: GovernanceDecisionValue, refs: string[]) => governance({
    rationale: value.rationale,
    counterargument: value.counterargument,
    evidenceRefs: [...refs, ...value.evidenceRefs],
  });
  const action = async (method: string, params: Record<string, unknown>) => {
    await client.rpc(method, params);
    state.reload();
  };

  if (state.loading && !state.data) return <Loading label="加载 Harness 版本、实验与 Trace…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;
  const { status, experiments, traces } = state.data;
  const activeExperiment = experiments.find((item) => ['validated', 'human_approved'].includes(item.status));

  return (
    <>
      <PageHeader title="Better Harness" description="候选版本在隔离评测、人工批准和显式提升后才会替代当前基线；随时保留回滚路径。" />
      <div className="harness-hero panel">
        <span className="eyebrow">当前生效版本</span>
        <strong>{status.active.version}</strong>
        <p>{status.manifest.description || status.manifest.name}</p>
        <div className="fact-strip">
          <span><small>模型配置</small><strong>{status.manifest.model_profile || '默认'}</strong></span>
          <span><small>组件</small><strong>{Object.keys(status.manifest.components).length}</strong></span>
          <span><small>激活人</small><strong>{status.active.activated_by}</strong></span>
          <span><small>激活时间</small><strong>{formatRelative(status.active.activated_at)}</strong></span>
        </div>
      </div>
      <div className="metric-grid compact">
        <Metric label="实验" value={experiments.length} />
        <Metric label="待治理" value={experiments.filter((item) => ['validated', 'human_approved'].includes(item.status)).length} tone="warning" />
        <Metric label="当前 Trace" value={traces.length} />
        <Metric label="前一版本" value={status.active.previous_version || '无'} />
      </div>
      <div className="dashboard-grid">
        <Section title="实验版本" className="span-two">
          {experiments.length === 0 ? <Empty title="暂无 Harness 实验" /> : (
            <div className="table-wrap"><table><thead><tr><th>候选</th><th>基线</th><th>变更</th><th>状态</th><th>创建</th></tr></thead><tbody>
              {experiments.map((item) => <tr key={item.id}><td><strong>{item.candidate_version}</strong><small>{item.id}</small></td><td>{item.base_version}</td><td>{item.change_manifest.change_summary}</td><td><Status tone={toneForState(item.status)}>{item.status}</Status></td><td>{formatRelative(item.created_at)}</td></tr>)}
            </tbody></table></div>
          )}
        </Section>
        <Section title="版本治理">
          {activeExperiment ? (
            <>
              <p><strong>{activeExperiment.candidate_version}</strong></p>
              <p>{activeExperiment.change_manifest.root_cause}</p>
              <GovernanceDecision
                approveLabel={activeExperiment.status === 'validated' ? '批准实验' : '提升版本'}
                onApprove={(value) => action(activeExperiment.status === 'validated' ? 'harness.experiment.approve' : 'harness.experiment.promote', { id: activeExperiment.id, ...payload(value, [`experiment:${activeExperiment.id}`]) })}
                onReject={(value) => action('harness.experiment.reject', { id: activeExperiment.id, ...payload(value, [`experiment:${activeExperiment.id}`]) })}
              />
            </>
          ) : status.active.previous_version ? (
            <>
              <p>当前没有待审批实验。需要时可治理回滚到 {status.active.previous_version}。</p>
              <GovernanceDecision
                approveLabel={`回滚到 ${status.active.previous_version}`}
                rejectLabel="保留当前版本"
                onApprove={(value) => action('harness.rollback', payload(value, [`harness:${status.active.version}`, `rollback:${status.active.previous_version}`]))}
                onReject={async () => undefined}
              />
            </>
          ) : <Empty title="当前无待治理版本" />}
        </Section>
      </div>
    </>
  );
}
