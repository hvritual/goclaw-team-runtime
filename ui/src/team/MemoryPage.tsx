import { FormEvent, useCallback, useState } from 'react';
import { useTeam } from './context';
import {
  Empty,
  ErrorState,
  Loading,
  Metric,
  PageHeader,
  Section,
  Status,
  toneForState,
} from './primitives';
import {
  CatalogMemoryRecord,
  CatalogMemorySearchResult,
  CatalogMemoryStats,
} from './types';
import { useAsyncData } from './use-data';

interface MemoryData {
  stats: CatalogMemoryStats;
  active: CatalogMemoryRecord[];
  pending: CatalogMemoryRecord[];
}

export function MemoryPage() {
  const { client, projectID } = useTeam();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<CatalogMemorySearchResult[] | null>(null);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const load = useCallback(async (): Promise<MemoryData> => {
    const [stats, active, pending] = await Promise.all([
      client.rpc<CatalogMemoryStats>('memory.catalog.status', { project_id: projectID }),
      client.rpc<CatalogMemoryRecord[]>('memory.catalog.list', { project_id: projectID, status: 'active', limit: 50 }),
      client.rpc<CatalogMemoryRecord[]>('memory.catalog.list', { project_id: projectID, status: 'pending', limit: 30 }),
    ]);
    return { stats, active, pending };
  }, [client, projectID]);
  const state = useAsyncData(load, [load]);

  const search = async (event: FormEvent) => {
    event.preventDefault();
    setSearching(true);
    setSearchError('');
    try {
      const value = await client.rpc<CatalogMemorySearchResult[]>('memory.catalog.search', {
        project_id: projectID,
        query: query.trim(),
        include_shared: true,
        limit: 30,
      });
      setResults(value);
    } catch (reason) {
      setSearchError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setSearching(false);
    }
  };

  if (state.loading && !state.data) return <Loading label="加载可信记忆目录…" />;
  if (state.error || !state.data) return <ErrorState error={state.error} onRetry={state.reload} />;

  const { stats, active, pending } = state.data;
  const display = results ?? active.map((record) => ({
    record,
    score: 0.5,
    citation: `catalog:${record.id}@v${record.version}`,
    review_due: Boolean(record.review_at && Date.parse(record.review_at) <= Date.now()),
    expired: Boolean(record.expires_at && Date.parse(record.expires_at) <= Date.now()),
  }));

  return (
    <>
      <PageHeader title="记忆" description="Markdown 是人类知识资产，Catalog 负责身份、版本、来源、有效期与检索准入。" />
      <div className="metric-grid compact">
        <Metric label="在藏" value={stats.by_status.active ?? 0} />
        <Metric label="待编目" value={stats.by_status.pending ?? 0} tone="warning" />
        <Metric label="待复核" value={stats.review_due} tone={stats.review_due ? 'warning' : 'success'} />
        <Metric label="未解冲突" value={stats.unresolved_contradictions} tone={stats.unresolved_contradictions ? 'danger' : 'success'} />
      </div>
      <form className="memory-search" onSubmit={(event) => void search(event)}>
        <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="检索项目决策、约束、事实或偏好…" aria-label="检索记忆" />
        <button className="button is-accent" disabled={searching}>{searching ? '检索中…' : '检索'}</button>
      </form>
      {searchError ? <p className="inline-error">{searchError}</p> : null}
      <div className="dashboard-grid">
        <Section title={`检索结果 ${display.length}`} className="span-two">
          {display.length === 0 ? <Empty title="没有匹配的已批准记忆" detail="候选与过期记录不会自动进入模型上下文。" /> : (
            <div className="card-list">
              {display.map((result) => (
                <article className="memory-record" key={result.record.id}>
                  <div className="card-heading">
                    <div><strong>{result.record.title}</strong><small>{result.record.kind} · v{result.record.version} · {result.citation}</small></div>
                    <Status tone={result.review_due ? 'warning' : toneForState(result.record.status)}>{result.review_due ? '待复核' : result.record.status}</Status>
                  </div>
                  <p>{result.record.abstract || result.record.content}</p>
                  <div className="source-line"><span>{result.record.provenance.source_uri}</span><span>置信度 {Math.round(result.record.confidence * 100)}%</span></div>
                </article>
              ))}
            </div>
          )}
        </Section>
        <Section title={`待编目 ${pending.length}`}>
          {pending.length === 0 ? <Empty title="没有待编目记录" /> : (
            <div className="stack-list">
              {pending.slice(0, 10).map((record) => (
                <article className="stack-item vertical" key={record.id}>
                  <div><strong>{record.title}</strong><small>{record.kind} · {record.provenance.source_uri}</small></div>
                  <Status tone="warning">pending</Status>
                </article>
              ))}
              <small>审批操作集中在“审批”工作区，避免检索与治理职责混合。</small>
            </div>
          )}
        </Section>
      </div>
    </>
  );
}
