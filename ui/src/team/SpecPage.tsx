import { FormEvent, useCallback, useState } from 'react';
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
import { OuroborosSession } from './types';
import { useAsyncData } from './use-data';

export function SpecPage() {
  const { client, projectID, topicID, session } = useTeam();
  const [request, setRequest] = useState('');
  const [brownfield, setBrownfield] = useState(true);
  const [busy, setBusy] = useState('');
  const [answers, setAnswers] = useState<Record<string, string>>({});
  const [actionError, setActionError] = useState('');
  const load = useCallback(
    () => client.rpc<OuroborosSession[]>('ouroboros.sessions', { project_id: projectID }),
    [client, projectID],
  );
  const state = useAsyncData(load, [load]);

  const run = async (key: string, action: () => Promise<unknown>) => {
    setBusy(key);
    setActionError('');
    try {
      await action();
      state.reload();
    } catch (reason) {
      setActionError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setBusy('');
    }
  };

  const start = (event: FormEvent) => {
    event.preventDefault();
    if (!request.trim()) return;
    void run('start', async () => {
      await client.rpc('ouroboros.session.start', {
        project_id: projectID,
        topic_id: topicID,
        raw_request: request.trim(),
        brownfield,
        base_ref: 'HEAD',
        created_by: session?.principal_id,
      }, 240_000);
      setRequest('');
    });
  };

  if (state.loading && !state.data) return <Loading label="加载 Ouroboros 规格闭环…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;

  const sessions = state.data;
  return (
    <>
      <PageHeader title="规格" description="interview → 不可变 Seed → 人工审批 → 编译 → 证据评估 → 演化" />
      <div className="metric-grid compact">
        <Metric label="规格会话" value={sessions.length} />
        <Metric label="待澄清" value={sessions.filter((item) => ['interviewing', 'clarification_required'].includes(item.status)).length} tone="warning" />
        <Metric label="待审批" value={sessions.filter((item) => ['awaiting_seed_approval', 'evolution_pending'].includes(item.status)).length} tone="accent" />
      </div>
      <form className="spec-start panel" onSubmit={start}>
        <div><strong>先结晶规格，再进入受控开发</strong><p>聊天记忆只提供上下文；Seed、验证证据和事件链才是执行依据。</p></div>
        <textarea value={request} onChange={(event) => setRequest(event.target.value)} rows={4} placeholder="描述目标、约束、非目标与验收方式…" />
        <div className="button-row">
          <label className="check-label"><input type="checkbox" checked={brownfield} onChange={(event) => setBrownfield(event.target.checked)} />现有代码库</label>
          <Button tone="accent" busy={busy === 'start'} disabled={!request.trim()}>开始规格访谈</Button>
        </div>
      </form>
      {actionError ? <p className="inline-error">{actionError}</p> : null}
      <div className="card-list">
        {sessions.map((item) => {
          const latest = item.rounds[item.rounds.length - 1];
          const answered = new Set((latest?.answers ?? []).map((answer) => answer.question_id));
          const questions = (latest?.questions ?? []).filter((question) => !answered.has(question.id));
          const actor = session?.principal_id;
          const compiledTasks = item.compiled_tasks ?? [];
          const taskID = compiledTasks[compiledTasks.length - 1]?.task_id;
          return (
            <article className="spec-card panel" key={item.id}>
              <div className="card-heading">
                <div><strong>{item.title || item.raw_request}</strong><small>{item.id} · {formatRelative(item.updated_at)}</small></div>
                <Status tone={toneForState(item.status)}>{item.status}</Status>
              </div>
              <p>{item.raw_request}</p>
              {latest ? (
                <div className="ambiguity">
                  <div><span>歧义度</span><strong>{Math.round(latest.assessment.overall * 100)}% / 阈值 {Math.round(latest.assessment.threshold * 100)}%</strong></div>
                  <progress
                    className="progress-track"
                    max={100}
                    value={Math.min(100, Math.max(0, latest.assessment.overall * 100))}
                    aria-label="规格明确度"
                  />
                  <small>{latest.assessment.summary} · ready {latest.assessment.ready_streak}/{latest.assessment.required_ready_streak}</small>
                </div>
              ) : null}
              {questions.length > 0 ? (
                <div className="question-list">
                  {questions.map((question) => (
                    <label key={question.id}>
                      <span>{question.blocking ? '必答 · ' : ''}{question.text}</span>
                      <textarea rows={2} value={answers[question.id] ?? ''} onChange={(event) => setAnswers((value) => ({ ...value, [question.id]: event.target.value }))} />
                    </label>
                  ))}
                  <Button
                    busy={busy === `answer-${item.id}`}
                    onClick={() => void run(`answer-${item.id}`, () => client.rpc('ouroboros.session.answer', {
                      id: item.id,
                      answers: questions.map((question) => ({ question_id: question.id, text: answers[question.id]?.trim() })).filter((answer) => answer.text),
                      actor,
                    }, 240_000))}
                  >提交答案并重评</Button>
                </div>
              ) : null}
              {item.last_error ? <p className="inline-error">{item.last_error}</p> : null}
              <div className="button-row">
                {['interviewing', 'clarification_required'].includes(item.status) ? <Button busy={busy === `reassess-${item.id}`} onClick={() => void run(`reassess-${item.id}`, () => client.rpc('ouroboros.session.reassess', { id: item.id, actor }, 240_000))}>重新评估</Button> : null}
                {item.status === 'seed_ready' ? <Button tone="accent" busy={busy === `seed-${item.id}`} onClick={() => void run(`seed-${item.id}`, () => client.rpc('ouroboros.session.crystallize', { id: item.id, actor }, 240_000))}>生成不可变 Seed</Button> : null}
                {item.status === 'approved' ? <Button tone="accent" busy={busy === `compile-${item.id}`} onClick={() => void run(`compile-${item.id}`, () => client.rpc('ouroboros.session.compile', { id: item.id, actor }))}>编译为开发任务</Button> : null}
                {item.status === 'compiled' && taskID ? <Button tone="accent" busy={busy === `evaluate-${item.id}`} onClick={() => void run(`evaluate-${item.id}`, () => client.rpc('ouroboros.session.evaluate', { id: item.id, task_id: taskID, actor }, 900_000))}>依据证据评估</Button> : null}
                {item.status === 'evaluated' ? <Button tone="accent" busy={busy === `evolve-${item.id}`} onClick={() => void run(`evolve-${item.id}`, () => client.rpc('ouroboros.session.evolve', { id: item.id, actor }, 240_000))}>生成演化候选</Button> : null}
              </div>
            </article>
          );
        })}
        {sessions.length === 0 ? <Empty title="还没有规格会话" detail="输入开发目标，Ouroboros 会优先追问高信息增益问题。" /> : null}
      </div>
    </>
  );
}
