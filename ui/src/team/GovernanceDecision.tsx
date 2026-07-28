import { FormEvent, useState } from 'react';
import { useTeam } from './context';
import { Button } from './primitives';

export interface GovernanceDecisionValue {
  rationale: string;
  counterargument: string;
  evidenceRefs: string[];
}

export function GovernanceDecision({
  approveLabel = '批准',
  rejectLabel = '拒绝',
  onApprove,
  onReject,
}: {
  approveLabel?: string;
  rejectLabel?: string;
  onApprove: (value: GovernanceDecisionValue) => Promise<void>;
  onReject: (value: GovernanceDecisionValue) => Promise<void>;
}) {
  const { reviewerToken } = useTeam();
  const [rationale, setRationale] = useState('');
  const [counterargument, setCounterargument] = useState('');
  const [evidence, setEvidence] = useState('');
  const [decision, setDecision] = useState<'approve' | 'reject' | null>(null);
  const [error, setError] = useState('');

  const submit = async (event: FormEvent, next: 'approve' | 'reject') => {
    event.preventDefault();
    setError('');
    setDecision(next);
    try {
      const value = {
        rationale: rationale.trim(),
        counterargument: counterargument.trim(),
        evidenceRefs: evidence.split(/[\n,]/).map((item) => item.trim()).filter(Boolean),
      };
      if (!value.rationale || !value.counterargument || value.evidenceRefs.length === 0) {
        throw new Error('理由、反方论点和至少一条证据引用均为必填项');
      }
      if (!reviewerToken) throw new Error('请先在右上角身份菜单输入 Reviewer Token');
      await (next === 'approve' ? onApprove(value) : onReject(value));
      setRationale('');
      setCounterargument('');
      setEvidence('');
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : String(reason));
    } finally {
      setDecision(null);
    }
  };

  return (
    <form className="governance-form">
      <div className="form-grid">
        <label><span>决策理由</span><textarea value={rationale} onChange={(event) => setRationale(event.target.value)} /></label>
        <label><span>反方论点</span><textarea value={counterargument} onChange={(event) => setCounterargument(event.target.value)} /></label>
        <label><span>证据引用</span><textarea value={evidence} onChange={(event) => setEvidence(event.target.value)} placeholder="Trace / 文档 / Issue ID，每行一条" /></label>
      </div>
      {error ? <p className="inline-error">{error}</p> : null}
      <div className="button-row">
        <Button tone="success" busy={decision === 'approve'} onClick={(event) => void submit(event, 'approve')}>{approveLabel}</Button>
        <Button tone="danger" busy={decision === 'reject'} onClick={(event) => void submit(event, 'reject')}>{rejectLabel}</Button>
      </div>
    </form>
  );
}
