import { ReactNode } from 'react';
import { Icon, IconName } from './icons';

export type Tone = 'neutral' | 'success' | 'warning' | 'danger' | 'accent';

export function Button({
  children,
  tone = 'neutral',
  busy = false,
  icon,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  tone?: Tone;
  busy?: boolean;
  icon?: IconName;
}) {
  return (
    <button className={`button is-${tone}`} disabled={busy || props.disabled} {...props}>
      {busy ? <span className="spinner" /> : icon ? <Icon name={icon} /> : null}
      <span>{children}</span>
    </button>
  );
}

export function Status({ children, tone = 'neutral' }: { children: ReactNode; tone?: Tone }) {
  return <span className={`status is-${tone}`}><i />{children}</span>;
}

export function Section({
  title,
  action,
  children,
  className = '',
}: {
  title: string;
  action?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <section className={`panel ${className}`}>
      <div className="panel-heading">
        <h2>{title}</h2>
        {action}
      </div>
      {children}
    </section>
  );
}

export function PageHeader({
  title,
  description,
  actions,
}: {
  title: string;
  description: string;
  actions?: ReactNode;
}) {
  return (
    <div className="page-heading">
      <div>
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      {actions ? <div className="page-actions">{actions}</div> : null}
    </div>
  );
}

export function Loading({ label = '加载中' }: { label?: string }) {
  return <div className="feedback"><span className="spinner" /><span>{label}</span></div>;
}

export function Empty({ title, detail }: { title: string; detail?: string }) {
  return (
    <div className="empty-state">
      <Icon name="document" />
      <strong>{title}</strong>
      {detail ? <p>{detail}</p> : null}
    </div>
  );
}

export function ErrorState({ error, onRetry }: { error: unknown; onRetry?: () => void }) {
  const message = error instanceof Error ? error.message : String(error);
  return (
    <div className="error-state" role="alert">
      <Icon name="warning" />
      <div>
        <strong>模块暂不可用</strong>
        <p>{message}</p>
      </div>
      {onRetry ? <Button onClick={onRetry} icon="refresh">重试</Button> : null}
    </div>
  );
}

export function Metric({
  label,
  value,
  detail,
  tone = 'neutral',
}: {
  label: string;
  value: string | number;
  detail?: string;
  tone?: Tone;
}) {
  return (
    <div className={`metric is-${tone}`}>
      <span>{label}</span>
      <strong>{value}</strong>
      {detail ? <small>{detail}</small> : null}
    </div>
  );
}

export function formatRelative(value?: string): string {
  if (!value) return '未知';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  const delta = Date.now() - date.getTime();
  const minutes = Math.floor(delta / 60_000);
  if (minutes < 1) return '刚刚';
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days} 天前`;
  return date.toLocaleDateString('zh-CN');
}

export function formatDuration(durationMS: number): string {
  if (durationMS < 1000) return `${durationMS} ms`;
  if (durationMS < 60_000) return `${(durationMS / 1000).toFixed(1)} s`;
  return `${(durationMS / 60_000).toFixed(1)} min`;
}

export function toneForState(value?: string): Tone {
  if (!value) return 'neutral';
  if (['active', 'online', 'done', 'resolved', 'closed', 'approved', 'passed', 'success', 'compliant'].includes(value)) {
    return 'success';
  }
  if (['blocked', 'failed', 'critical', 'offline', 'rejected', 'cancelled', 'error'].includes(value)) {
    return 'danger';
  }
  if (['pending', 'triaged', 'assigned', 'verifying', 'review', 'in_review', 'away', 'draining', 'warning', 'awaiting_acceptance'].includes(value)) {
    return 'warning';
  }
  return 'neutral';
}
