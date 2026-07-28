import { FormEvent, useCallback, useEffect, useRef, useState } from 'react';
import { useTeam } from './context';
import {
  Button,
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
  TeamAssignment,
  TeamComponentsSummary,
  TeamDocsSummary,
  TeamIssue,
  TeamMember,
  TeamPolicyStatus,
  TeamRunner,
  TeamWorkItem,
} from './types';
import { useAsyncData } from './use-data';
import { nextIssueStatuses, nextWorkItemStatuses } from './workflow-state';

interface TeamData {
  members: TeamMember[];
  work: TeamWorkItem[];
  issues: TeamIssue[];
  assignments: TeamAssignment[];
  runners: TeamRunner[];
  policy: TeamPolicyStatus;
  docs: TeamDocsSummary;
  components: TeamComponentsSummary;
}

const selectStyle = {
  minWidth: 130,
  border: '1px solid #dedbd8',
  borderRadius: 4,
  background: '#fff',
  padding: '7px 9px',
  fontSize: 11,
};

export function TeamPage() {
  const { client, projectID } = useTeam();
  const [busy, setBusy] = useState('');
  const [actionError, setActionError] = useState('');
  const [issueTitle, setIssueTitle] = useState('');
  const [issueDescription, setIssueDescription] = useState('');
  const [issueSeverity, setIssueSeverity] = useState<TeamIssue['severity']>('medium');
  const [workTitle, setWorkTitle] = useState('');
  const [workInstructions, setWorkInstructions] = useState('');
  const [linkedIssue, setLinkedIssue] = useState('');
  const [issueNext, setIssueNext] = useState<Record<string, TeamIssue['status'] | ''>>({});
  const [workNext, setWorkNext] = useState<Record<string, TeamWorkItem['status'] | ''>>({});
  const [resolutions, setResolutions] = useState<Record<string, string>>({});
  const [assignees, setAssignees] = useState<Record<string, string>>({});
  const projectRef = useRef(projectID);
  projectRef.current = projectID;
  const load = useCallback(async (): Promise<TeamData> => {
    const [members, work, issues, assignments, runners, policy, docs, components] =
      await Promise.all([
        client.rpc<TeamMember[]>('team.members', { project_id: projectID }),
        client.rpc<TeamWorkItem[]>('work.items', { project_id: projectID, limit: 60 }),
        client.rpc<TeamIssue[]>('issue.list', { project_id: projectID, limit: 60 }),
        client.rpc<TeamAssignment[]>('assignment.list', { project_id: projectID }),
        client.rpc<TeamRunner[]>('runner.list', { project_id: projectID }),
        client.rpc<TeamPolicyStatus>('policy.status', { project_id: projectID }),
        client.rpc<TeamDocsSummary>('docs.summary', { project_id: projectID, limit: 10 }),
        client.rpc<TeamComponentsSummary>('components.summary', { project_id: projectID, limit: 10 }),
      ]);
    return { members, work, issues, assignments, runners, policy, docs, components };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  useEffect(() => {
    setBusy('');
    setActionError('');
    setIssueNext({});
    setWorkNext({});
    setResolutions({});
    setAssignees({});
    setLinkedIssue('');
  }, [projectID]);

  const mutate = async (
    key: string,
    action: () => Promise<unknown>,
    onSuccess?: () => void,
  ) => {
    const mutationProject = projectID;
    setBusy(key);
    setActionError('');
    try {
      await action();
      if (projectRef.current !== mutationProject) return;
      onSuccess?.();
      state.reload();
    } catch (reason) {
      if (projectRef.current === mutationProject) {
        setActionError(reason instanceof Error ? reason.message : String(reason));
      }
    } finally {
      if (projectRef.current === mutationProject) setBusy('');
    }
  };

  const createIssue = (event: FormEvent) => {
    event.preventDefault();
    if (!issueTitle.trim()) return;
    void mutate('create-issue', () => client.rpc('issue.create', {
      project_id: projectID,
      type: 'bug',
      title: issueTitle.trim(),
      description: issueDescription.trim(),
      severity: issueSeverity,
    }), () => {
      setIssueTitle('');
      setIssueDescription('');
      setIssueSeverity('medium');
    });
  };

  const createWorkItem = (event: FormEvent) => {
    event.preventDefault();
    if (!workTitle.trim() || !workInstructions.trim()) return;
    void mutate('create-work', () => client.rpc('work.create', {
      project_id: projectID,
      issue_id: linkedIssue,
      title: workTitle.trim(),
      instructions: workInstructions.trim(),
      priority: 'p2',
    }), () => {
      setWorkTitle('');
      setWorkInstructions('');
      setLinkedIssue('');
    });
  };

  if (state.loading && !state.data) return <Loading label="加载成员、任务、Runner 与工程资产…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;
  const { members, work, issues, assignments, runners, policy, docs, components } = state.data;
  const names = new Map(members.map((member) => [member.id, member.display_name]));
  const openIssues = issues.filter((item) =>
    !['resolved', 'closed', 'cancelled'].includes(item.status));
  const activeMembers = members.filter((member) => member.status !== 'disabled');
  const activeAssignment = (targetType: TeamAssignment['target_type'], targetID: string) =>
    assignments.find((assignment) =>
      assignment.target_type === targetType &&
      assignment.target_id === targetID &&
      assignment.role === 'owner' &&
      assignment.status === 'active');

  const assign = (targetType: TeamAssignment['target_type'], targetID: string) => {
    const key = `${targetType}:${targetID}`;
    const userID = assignees[key];
    if (!userID) return;
    void mutate(`assign-${key}`, () => client.rpc('assignment.create', {
      project_id: projectID,
      target_type: targetType,
      target_id: targetID,
      user_id: userID,
      role: 'owner',
    }), () => setAssignees((current) => ({ ...current, [key]: '' })));
  };

  return (
    <>
      <PageHeader title="团队" description="成员、任务、Bug、Runner、策略、文档和共享组件均由 Gateway 按项目授权返回。" />
      <div className="metric-grid compact">
        <Metric label="项目成员" value={members.length} />
        <Metric label="活动任务" value={work.filter((item) => !['done', 'cancelled'].includes(item.status)).length} />
        <Metric label="未关闭 Bug" value={openIssues.length} tone={openIssues.some((item) => ['critical', 'high'].includes(item.severity)) ? 'danger' : 'neutral'} />
        <Metric label="在线 Runner" value={runners.filter((item) => ['online', 'busy'].includes(item.status)).length} tone="success" />
      </div>
      {actionError ? <p className="inline-error" role="alert">{actionError}</p> : null}
      <div className="dashboard-grid">
        <Section title="新建 Bug" className="span-two">
          <form className="governance-form" style={{ margin: 0, padding: 12 }} onSubmit={createIssue}>
            <div className="form-grid">
              <label><span>标题</span><input value={issueTitle} onChange={(event) => setIssueTitle(event.target.value)} maxLength={300} required /></label>
              <label><span>严重度</span>
                <select style={selectStyle} value={issueSeverity} onChange={(event) => setIssueSeverity(event.target.value as TeamIssue['severity'])}>
                  {(['critical', 'high', 'medium', 'low'] as const).map((value) => <option key={value} value={value}>{value}</option>)}
                </select>
              </label>
              <label><span>描述</span><textarea value={issueDescription} onChange={(event) => setIssueDescription(event.target.value)} rows={2} /></label>
            </div>
            <div className="button-row"><Button tone="accent" busy={busy === 'create-issue'} disabled={!issueTitle.trim()}>登记 Bug</Button></div>
          </form>
        </Section>
        <Section title="新建任务">
          <form className="governance-form" style={{ margin: 0, padding: 12 }} onSubmit={createWorkItem}>
            <label><span>标题</span><input value={workTitle} onChange={(event) => setWorkTitle(event.target.value)} maxLength={300} required /></label>
            <label><span>执行说明</span><textarea value={workInstructions} onChange={(event) => setWorkInstructions(event.target.value)} rows={2} required /></label>
            <label><span>关联 Bug（可选）</span>
              <select style={selectStyle} value={linkedIssue} onChange={(event) => setLinkedIssue(event.target.value)}>
                <option value="">不关联</option>
                {openIssues.map((issue) => <option key={issue.id} value={issue.id}>{issue.id} · {issue.title}</option>)}
              </select>
            </label>
            <div className="button-row"><Button tone="accent" busy={busy === 'create-work'} disabled={!workTitle.trim() || !workInstructions.trim()}>创建任务</Button></div>
          </form>
        </Section>

        <Section title="成员负载" className="span-two">
          {members.length === 0 ? <Empty title="尚未登记项目成员" /> : (
            <div className="member-grid">
              {members.map((member) => {
                const capacity = member.capacity;
                const utilization = Math.max(0, Math.min(100, capacity?.utilization_percent ?? 0));
                return (
                  <article className="member-card" key={member.id}>
                    <div className="avatar">{(member.display_name || member.id).slice(0, 1).toUpperCase()}</div>
                    <div>
                      <div className="card-heading"><div><strong>{member.display_name || member.id}</strong><small>{member.role} · {(member.business_domains ?? []).join(' / ') || '未设置业务域'}</small></div><Status tone={toneForState(member.status)}>{member.status}</Status></div>
                      <progress className="progress-track" max={100} value={utilization} aria-label={`${member.display_name} 利用率 ${utilization}%`} />
                      <small>{capacity?.active_work ?? 0} 进行中 · {capacity?.queued_work ?? 0} 排队 · {capacity?.blocked_work ?? 0} 阻塞 · {utilization}%</small>
                    </div>
                  </article>
                );
              })}
            </div>
          )}
        </Section>
        <Section title="Runner 与租约">
          {runners.length === 0 ? <Empty title="没有已登记 Runner" /> : (
            <div className="stack-list">
              {runners.map((runner) => (
                <article className="stack-item" key={runner.id}>
                  <div><strong>{runner.display_name || runner.id}</strong><small>{names.get(runner.member_id ?? '') || runner.member_id || '未绑定'} · {runner.current_work_id || '空闲'} · {formatRelative(runner.last_seen_at)}</small></div>
                  <Status tone={toneForState(runner.status)}>{runner.status}</Status>
                </article>
              ))}
            </div>
          )}
        </Section>
        <Section title="工程策略">
          <div className="fact-strip vertical">
            <span><small>有效版本</small><strong>{policy.effective_version?.slice(0, 16) || '未锁定'}</strong></span>
            <span><small>状态</small><strong>{policy.compliant ? '一致' : '存在漂移'}</strong></span>
            <span><small>层数</small><strong>{policy.layers?.length ?? 0}</strong></span>
          </div>
          {(policy.layers ?? []).map((layer) => <div className="stack-item compact" key={layer.id}><span>{layer.scope} · {layer.id}@{layer.version}</span><Status tone={layer.compliant === false ? 'danger' : 'success'}>{layer.compliant === false ? '漂移' : '一致'}</Status></div>)}
        </Section>

        <Section title="活动任务" className="span-two">
          {work.length === 0 ? <Empty title="暂无任务" /> : (
            <div className="table-wrap"><table><thead><tr><th>任务</th><th>负责人</th><th>状态流转</th><th>负责人操作</th></tr></thead><tbody>
              {work.filter((item) => !['done', 'cancelled'].includes(item.status)).slice(0, 20).map((item) => {
                const key = `work_item:${item.id}`;
                const owner = activeAssignment('work_item', item.id);
                const next = workNext[item.id] ?? '';
                return <tr key={item.id}>
                  <td><strong>{item.title}</strong><small>{item.id} · {item.business_domain || '未设置业务域'}</small></td>
                  <td>{names.get(owner?.user_id ?? '') || owner?.user_id || item.assignee_id || '未分配'}</td>
                  <td><div className="button-row">
                    <select style={selectStyle} aria-label={`${item.id} 下一状态`} value={next} onChange={(event) => setWorkNext((current) => ({ ...current, [item.id]: event.target.value as TeamWorkItem['status'] }))}>
                      <option value="">选择合法状态</option>
                      {nextWorkItemStatuses(item.status).map((status) => <option key={status} value={status}>{status}</option>)}
                    </select>
                    <Button busy={busy === `transition-work-${item.id}`} disabled={!next} onClick={() => void mutate(`transition-work-${item.id}`, () => client.rpc('work.transition', { project_id: projectID, work_item_id: item.id, status: next }), () => setWorkNext((current) => ({ ...current, [item.id]: '' })))}>流转</Button>
                  </div></td>
                  <td>{owner ? (
                    <Button tone="danger" busy={busy === `release-${key}`} onClick={() => void mutate(`release-${key}`, () => client.rpc('assignment.release', { project_id: projectID, assignment_id: owner.id }))}>解除</Button>
                  ) : (
                    <div className="button-row">
                      <select style={selectStyle} aria-label={`${item.id} 负责人`} value={assignees[key] ?? ''} onChange={(event) => setAssignees((current) => ({ ...current, [key]: event.target.value }))}>
                        <option value="">选择成员</option>
                        {activeMembers.map((member) => <option key={member.id} value={member.id}>{member.display_name}</option>)}
                      </select>
                      <Button busy={busy === `assign-${key}`} disabled={!assignees[key]} onClick={() => assign('work_item', item.id)}>分配</Button>
                    </div>
                  )}</td>
                </tr>;
              })}
            </tbody></table></div>
          )}
        </Section>

        <Section title="Bug 状态" className="span-two">
          {openIssues.length === 0 ? <Empty title="没有未关闭 Bug" /> : (
            <div className="stack-list">{openIssues.slice(0, 20).map((item) => {
              const key = `issue:${item.id}`;
              const owner = activeAssignment('issue', item.id);
              const next = issueNext[item.id] ?? '';
              const needsResolution = next === 'resolved' || next === 'closed';
              return <article className="stack-item vertical" key={item.id}>
                <div>
                  <strong>{item.title}</strong>
                  <small>{item.id} · {names.get(owner?.user_id ?? '') || owner?.user_id || item.owner_id || '未分配'}</small>
                  <div className="button-row" style={{ justifyContent: 'flex-start', marginTop: 8 }}>
                    <select style={selectStyle} aria-label={`${item.id} 下一状态`} value={next} onChange={(event) => setIssueNext((current) => ({ ...current, [item.id]: event.target.value as TeamIssue['status'] }))}>
                      <option value="">选择合法状态</option>
                      {nextIssueStatuses(item.status).map((status) => <option key={status} value={status}>{status}</option>)}
                    </select>
                    {needsResolution ? <input aria-label={`${item.id} 解决说明`} placeholder="解决说明或关联修复证据" value={resolutions[item.id] ?? ''} onChange={(event) => setResolutions((current) => ({ ...current, [item.id]: event.target.value }))} style={{ minWidth: 210 }} /> : null}
                    <Button busy={busy === `transition-issue-${item.id}`} disabled={!next || (needsResolution && !(resolutions[item.id] ?? '').trim())} onClick={() => void mutate(`transition-issue-${item.id}`, () => client.rpc('issue.transition', { project_id: projectID, issue_id: item.id, status: next, resolution: resolutions[item.id]?.trim() ?? '' }), () => {
                      setIssueNext((current) => ({ ...current, [item.id]: '' }));
                      setResolutions((current) => ({ ...current, [item.id]: '' }));
                    })}>流转</Button>
                  </div>
                  <div className="button-row" style={{ justifyContent: 'flex-start', marginTop: 8 }}>
                    {owner ? <Button tone="danger" busy={busy === `release-${key}`} onClick={() => void mutate(`release-${key}`, () => client.rpc('assignment.release', { project_id: projectID, assignment_id: owner.id }))}>解除负责人</Button> : <>
                      <select style={selectStyle} aria-label={`${item.id} 负责人`} value={assignees[key] ?? ''} onChange={(event) => setAssignees((current) => ({ ...current, [key]: event.target.value }))}>
                        <option value="">选择成员</option>
                        {activeMembers.map((member) => <option key={member.id} value={member.id}>{member.display_name}</option>)}
                      </select>
                      <Button busy={busy === `assign-${key}`} disabled={!assignees[key]} onClick={() => assign('issue', item.id)}>分配负责人</Button>
                    </>}
                  </div>
                </div>
                <Status tone={toneForState(item.severity)}>{item.severity} · {item.status}</Status>
              </article>;
            })}</div>
          )}
        </Section>

        <Section title={`方案文档 ${docs.total}`}>
          <div className="fact-strip vertical"><span><small>已批准</small><strong>{docs.approved}</strong></span><span><small>陈旧</small><strong>{docs.stale}</strong></span><span><small>待复核</small><strong>{docs.review_due}</strong></span></div>
          {(docs.items ?? []).map((item) => <div className="stack-item compact" key={item.id}><div><strong>{item.title}</strong><small>{item.path}</small></div><Status tone={toneForState(item.status)}>{item.status}</Status></div>)}
        </Section>
        <Section title={`共享组件 ${components.total}`}>
          <div className="fact-strip vertical"><span><small>可复用</small><strong>{components.reusable}</strong></span><span><small>待评审</small><strong>{components.pending_review}</strong></span><span><small>弃用</small><strong>{components.deprecated}</strong></span></div>
          {(components.items ?? []).map((item) => <div className="stack-item compact" key={item.id}><div><strong>{item.name}</strong><small>{item.kind} · {item.owner_id || '未分配'}</small></div><Status tone={toneForState(item.status)}>{item.status}</Status></div>)}
        </Section>
      </div>
    </>
  );
}
