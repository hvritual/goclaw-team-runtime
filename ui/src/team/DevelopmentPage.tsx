import { useCallback, useState } from 'react';
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
import { DevTask } from './types';
import { useAsyncData } from './use-data';

export function DevelopmentPage() {
  const { client, projectID } = useTeam();
  const [busy, setBusy] = useState('');
  const [actionError, setActionError] = useState('');
  const load = useCallback(
    () => client.rpc<DevTask[]>('dev.tasks', { project_id: projectID }),
    [client, projectID],
  );
  const state = useAsyncData(load, [load]);

  const act = async (task: DevTask, method: string, params: Record<string, unknown>) => {
    setBusy(task.id);
    setActionError('');
    try {
      await client.rpc(method, params, method === 'dev.task.enqueue' ? 120_000 : 30_000);
      state.reload();
    } catch (reason) {
      setActionError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy('');
    }
  };

  if (state.loading && !state.data) return <Loading label="加载冻结任务与工作站状态…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;

  const tasks = state.data;
  return (
    <>
      <PageHeader title="开发" description="每个 revision 使用独立 worktree；中央只发不可变执行包，本地 Runner 用成员自己的 Codex OAuth 执行。" />
      <div className="metric-grid compact">
        <Metric label="任务" value={tasks.length} />
        <Metric label="执行中" value={tasks.filter((item) => ['running', 'checking'].includes(item.status)).length} tone="accent" />
        <Metric label="待验收" value={tasks.filter((item) => item.status === 'awaiting_acceptance').length} tone="warning" />
        <Metric label="失败/修复" value={tasks.filter((item) => ['failed', 'repair_pending'].includes(item.status)).length} tone={tasks.some((item) => ['failed', 'repair_pending'].includes(item.status)) ? 'danger' : 'success'} />
      </div>
      {actionError ? <p className="inline-error">{actionError}</p> : null}
      <div className="card-list">
        {tasks.map((task) => (
          <article className="dev-card panel" key={task.id}>
            <div className="card-heading">
              <div><strong>{task.title}</strong><small>{task.id} · revision {task.compile.revision}</small></div>
              <Status tone={toneForState(task.status)}>{task.status}</Status>
            </div>
            <p>{task.goal.objective}</p>
            <div className="fact-strip">
              <span><small>分支 / 基线</small><strong>{task.branch || task.compile.base_ref}</strong></span>
              <span><small>文件上限</small><strong>{task.scope.max_changed_files}</strong></span>
              <span><small>行数上限</small><strong>{task.scope.max_changed_lines}</strong></span>
              <span><small>修复次数</small><strong>{task.repair_count}</strong></span>
              <span><small>更新</small><strong>{formatRelative(task.updated_at)}</strong></span>
            </div>
            {task.last_gate ? (
              <div className={`gate-result ${task.last_gate.passed ? 'is-passed' : 'is-failed'}`}>
                <strong>{task.last_gate.passed ? 'DoneGate 已通过' : 'DoneGate 未通过'}</strong>
                <span>{task.last_gate.verdict}</span>
                {(task.last_gate.reasons ?? []).length > 0 ? <small>{task.last_gate.reasons?.join('；')}</small> : null}
                <code>{task.last_gate.evidence_path}</code>
              </div>
            ) : <div className="safety-note">尚未生成 EvidencePackage；没有证据时不能完成任务。</div>}
            <div className="button-row">
              {task.status === 'frozen' ? <Button tone="accent" busy={busy === task.id} onClick={() => void act(task, 'dev.task.enqueue', { task_id: task.id, capabilities: ['codex'] })}>进入工作站队列</Button> : null}
              {['repair_pending', 'failed'].includes(task.status) ? <Button tone="accent" busy={busy === task.id} onClick={() => void act(task, 'dev.task.revise', { id: task.id, expected_revision: task.compile.revision, reason: 'DoneGate 未通过，创建隔离修复 revision 并重新履行评审。' })}>创建修复版本</Button> : null}
              <Button onClick={() => navigator.clipboard.writeText(task.id)}>复制任务 ID</Button>
            </div>
          </article>
        ))}
        {tasks.length === 0 ? <Empty title="还没有开发任务" detail="先在“规格”中完成 Seed，再编译为任务并履行四类评审。" /> : null}
      </div>
    </>
  );
}
