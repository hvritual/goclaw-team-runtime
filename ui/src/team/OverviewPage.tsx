import { useCallback } from 'react';
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
import {
  DevTask,
  HarnessTrace,
  KnowledgeProposal,
  TeamIssue,
  TeamPolicyStatus,
  TeamRunner,
  TeamWorkItem,
} from './types';
import { useAsyncData } from './use-data';

interface OverviewData {
  work: TeamWorkItem[];
  issues: TeamIssue[];
  runners: TeamRunner[];
  policy: TeamPolicyStatus;
  tasks: DevTask[];
  traces: HarnessTrace[];
  proposals: KnowledgeProposal[];
}

export function OverviewPage() {
  const { client, projectID } = useTeam();
  const load = useCallback(async (): Promise<OverviewData> => {
    const [work, issues, runners, policy, tasks, traces, proposals] = await Promise.all([
      client.rpc<TeamWorkItem[]>('work.items', { project_id: projectID, limit: 40 }),
      client.rpc<TeamIssue[]>('issue.list', { project_id: projectID, limit: 40 }),
      client.rpc<TeamRunner[]>('runner.list', { project_id: projectID }),
      client.rpc<TeamPolicyStatus>('policy.status', { project_id: projectID }),
      client.rpc<DevTask[]>('dev.tasks', { project_id: projectID }),
      client.rpc<HarnessTrace[]>('harness.traces', { project_id: projectID, limit: 12 }),
      client.rpc<KnowledgeProposal[]>('knowledge.proposals', { status: 'pending' }),
    ]);
    return { work, issues, runners, policy, tasks, traces, proposals };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  if (state.loading && !state.data) return <Loading label="正在汇总项目控制面…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;

  const { work, issues, runners, policy, tasks, traces, proposals } = state.data;
  const done = work.filter((item) => item.status === 'done').length;
  const active = work.filter((item) => ['ready', 'in_progress', 'verifying'].includes(item.status)).length;
  const blocked = work.filter((item) => item.status === 'blocked').length;
  const openIssues = issues.filter((item) => !['resolved', 'closed'].includes(item.status));
  const onlineRunners = runners.filter((item) => ['online', 'busy'].includes(item.status)).length;
  const evidenceCount = tasks.filter((task) => Boolean(task.last_evidence || task.last_gate?.evidence_path)).length;
  const progress = work.length === 0 ? 0 : Math.round((done / work.length) * 100);

  return (
    <>
      <PageHeader
        title="项目执行总览"
        description="需求、任务、缺陷、工作站和证据的同一控制面；数据均来自当前项目运行时。"
      />
      <div className="overview-hero">
        <div>
          <span className="eyebrow">项目健康度</span>
          <strong>{progress}%</strong>
          <p>{done} / {work.length} 个工作项已完成</p>
        </div>
        <progress className="progress-track" max={100} value={progress} aria-label={`项目进度 ${progress}%`} />
        <Status tone={policy.compliant ? 'success' : 'danger'}>
          {policy.compliant ? '策略一致' : `${policy.drift_count} 项策略漂移`}
        </Status>
      </div>
      <div className="metric-grid">
        <Metric label="待审批" value={proposals.length + tasks.filter((task) => ['review_pending', 'awaiting_acceptance'].includes(task.status)).length} detail="知识与任务治理" tone="warning" />
        <Metric label="进行中" value={active} detail={`${tasks.filter((task) => task.status === 'running').length} 个 Codex 任务执行中`} tone="accent" />
        <Metric label="阻塞" value={blocked} detail="需要项目经理处理" tone={blocked ? 'danger' : 'success'} />
        <Metric label="证据覆盖" value={`${evidenceCount}/${tasks.length}`} detail="任务已有 EvidencePackage" tone={evidenceCount === tasks.length ? 'success' : 'warning'} />
      </div>
      <div className="dashboard-grid">
        <Section title="当前工作" className="span-two">
          {work.length === 0 ? <Empty title="暂无工作项" detail="创建并分配工作项后会在这里显示。" /> : (
            <div className="table-wrap">
              <table>
                <thead><tr><th>工作项</th><th>负责人</th><th>优先级</th><th>状态</th><th>更新</th></tr></thead>
                <tbody>
                  {work.slice(0, 8).map((item) => (
                    <tr key={item.id}>
                      <td><strong>{item.title}</strong><small>{item.id}</small></td>
                      <td>{item.assignee_id || '未分配'}</td>
                      <td>{item.priority || 'medium'}</td>
                      <td><Status tone={toneForState(item.status)}>{item.status}</Status></td>
                      <td>{formatRelative(item.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </Section>
        <Section title="Issue 风险">
          {openIssues.length === 0 ? <Empty title="没有未解决 Issue" /> : (
            <div className="stack-list">
              {openIssues.slice(0, 6).map((issue) => (
                <article key={issue.id} className="stack-item">
                  <div><strong>{issue.title}</strong><small>{issue.id} · {issue.owner_id || '未分配'}</small></div>
                  <Status tone={toneForState(issue.severity)}>{issue.severity}</Status>
                </article>
              ))}
            </div>
          )}
        </Section>
        <Section title={`工作站 ${onlineRunners}/${runners.length} 在线`}>
          {runners.length === 0 ? <Empty title="暂无已注册 Runner" /> : (
            <div className="stack-list">
              {runners.slice(0, 6).map((runner) => (
                <article key={runner.id} className="stack-item">
                  <div><strong>{runner.display_name || runner.id}</strong><small>{runner.current_work_id || (runner.capabilities ?? []).join(' · ') || '空闲'}</small></div>
                  <Status tone={toneForState(runner.status)}>{runner.status}</Status>
                </article>
              ))}
            </div>
          )}
        </Section>
        <Section title="最近证据" className="span-two">
          {traces.length === 0 ? <Empty title="尚无 Trace" detail="任务执行后会形成可追溯证据。" /> : (
            <div className="evidence-grid">
              {traces.slice(0, 6).map((trace) => (
                <article key={trace.id} className="evidence-card">
                  <div><Status tone={toneForState(trace.status)}>{trace.status}</Status><span>{formatRelative(trace.started_at)}</span></div>
                  <strong>{trace.task_id || trace.work_item_id || trace.topic_id}</strong>
                  <small>{trace.id} · Harness {trace.harness_version}</small>
                </article>
              ))}
            </div>
          )}
        </Section>
      </div>
    </>
  );
}
