import { useCallback } from 'react';
import { useTeam } from './context';
import {
  Empty,
  ErrorState,
  formatDuration,
  formatRelative,
  Loading,
  Metric,
  PageHeader,
  Section,
  Status,
  toneForState,
} from './primitives';
import { DevTask, HarnessTrace } from './types';
import { useAsyncData } from './use-data';

interface ProgressData {
  tasks: DevTask[];
  traces: HarnessTrace[];
}

export function ProgressPage() {
  const { client, projectID } = useTeam();
  const load = useCallback(async (): Promise<ProgressData> => {
    const [tasks, traces] = await Promise.all([
      client.rpc<DevTask[]>('dev.tasks', { project_id: projectID }),
      client.rpc<HarnessTrace[]>('harness.traces', { project_id: projectID, limit: 60 }),
    ]);
    return { tasks, traces };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);
  if (state.loading && !state.data) return <Loading label="加载任务状态与运行轨迹…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;
  const { tasks, traces } = state.data;
  const passed = tasks.filter((item) => item.last_gate?.passed).length;
  const failed = traces.filter((item) => !['completed', 'success', 'done'].includes(item.status)).length;

  return (
    <>
      <PageHeader title="进度" description="进度不使用虚假的单一百分比：任务状态、证据覆盖和运行健康度分别呈现。" />
      <div className="metric-grid compact">
        <Metric label="任务总数" value={tasks.length} />
        <Metric label="DoneGate 通过" value={`${passed}/${tasks.length}`} tone={passed === tasks.length ? 'success' : 'warning'} />
        <Metric label="运行轨迹" value={traces.length} />
        <Metric label="异常轨迹" value={failed} tone={failed ? 'danger' : 'success'} />
      </div>
      <div className="dashboard-grid">
        <Section title="任务状态" className="span-two">
          {tasks.length === 0 ? <Empty title="暂无任务进度" /> : (
            <div className="table-wrap"><table><thead><tr><th>任务</th><th>Revision</th><th>状态</th><th>DoneGate</th><th>更新</th></tr></thead><tbody>
              {tasks.map((task) => <tr key={task.id}><td><strong>{task.title}</strong><small>{task.id}</small></td><td>{task.compile.revision}</td><td><Status tone={toneForState(task.status)}>{task.status}</Status></td><td><Status tone={task.last_gate?.passed ? 'success' : task.last_gate ? 'danger' : 'neutral'}>{task.last_gate?.passed ? '通过' : task.last_gate ? '失败' : '无证据'}</Status></td><td>{formatRelative(task.updated_at)}</td></tr>)}
            </tbody></table></div>
          )}
        </Section>
        <Section title="最近运行轨迹">
          {traces.length === 0 ? <Empty title="还没有运行轨迹" /> : (
            <div className="timeline">
              {traces.map((trace) => (
                <article key={trace.id}>
                  <i className={`timeline-dot is-${toneForState(trace.status)}`} />
                  <div><div><strong>{trace.task_id || trace.work_item_id || trace.topic_id}</strong><Status tone={toneForState(trace.status)}>{trace.status}</Status></div><p>{trace.output || trace.error || trace.input || '无摘要'}</p><small>{trace.id} · {formatDuration(trace.duration_ms)} · {formatRelative(trace.started_at)}</small></div>
                </article>
              ))}
            </div>
          )}
        </Section>
      </div>
    </>
  );
}
