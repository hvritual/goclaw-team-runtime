import {
  IssueSeverity,
  IssueStatus,
  RunnerStatus,
  TeamMemberStatus,
  WorkItemStatus
} from "./types";

export type TeamStateTone = "accent" | "success" | "warning" | "danger" | "muted";

export interface TeamStatePresentation {
  label: string;
  tone: TeamStateTone;
}

const memberStates: Record<TeamMemberStatus, TeamStatePresentation> = {
  active: { label: "在线", tone: "success" },
  away: { label: "暂离", tone: "warning" },
  offline: { label: "离线", tone: "muted" },
  disabled: { label: "停用", tone: "danger" }
};

const workStates: Record<WorkItemStatus, TeamStatePresentation> = {
  backlog: { label: "待规划", tone: "muted" },
  ready: { label: "可执行", tone: "accent" },
  in_progress: { label: "进行中", tone: "success" },
  blocked: { label: "受阻", tone: "danger" },
  in_review: { label: "评审中", tone: "warning" },
  done: { label: "已完成", tone: "muted" },
  cancelled: { label: "已取消", tone: "muted" }
};

const issueStates: Record<IssueStatus, TeamStatePresentation> = {
  open: { label: "待处理", tone: "danger" },
  triaged: { label: "已分诊", tone: "warning" },
  in_progress: { label: "修复中", tone: "accent" },
  in_review: { label: "验证中", tone: "warning" },
  resolved: { label: "已解决", tone: "success" },
  closed: { label: "已关闭", tone: "muted" },
  reopened: { label: "重新打开", tone: "danger" }
};

const runnerStates: Record<RunnerStatus, TeamStatePresentation> = {
  online: { label: "在线", tone: "success" },
  busy: { label: "执行中", tone: "accent" },
  draining: { label: "排空中", tone: "warning" },
  offline: { label: "离线", tone: "danger" }
};

const severityStates: Record<IssueSeverity, TeamStatePresentation> = {
  critical: { label: "致命", tone: "danger" },
  high: { label: "高", tone: "danger" },
  medium: { label: "中", tone: "warning" },
  low: { label: "低", tone: "muted" }
};

export function memberState(value: TeamMemberStatus): TeamStatePresentation {
  return memberStates[value] ?? { label: value, tone: "muted" };
}

export function workState(value: WorkItemStatus): TeamStatePresentation {
  return workStates[value] ?? { label: value, tone: "muted" };
}

export function issueState(value: IssueStatus): TeamStatePresentation {
  return issueStates[value] ?? { label: value, tone: "muted" };
}

export function runnerState(value: RunnerStatus): TeamStatePresentation {
  return runnerStates[value] ?? { label: value, tone: "muted" };
}

export function severityState(value: IssueSeverity): TeamStatePresentation {
  return severityStates[value] ?? { label: value, tone: "muted" };
}

export function leaseState(
  expiresAt: string | undefined,
  now = Date.now()
): TeamStatePresentation {
  if (!expiresAt) return { label: "无租约", tone: "muted" };
  const expires = new Date(expiresAt).getTime();
  if (!Number.isFinite(expires)) return { label: "租约时间无效", tone: "danger" };
  const remaining = expires - now;
  if (remaining <= 0) return { label: "租约已过期", tone: "danger" };
  if (remaining <= 5 * 60_000) return { label: "租约即将到期", tone: "warning" };
  return { label: "租约有效", tone: "success" };
}

export function teamStateClass(presentation: TeamStatePresentation): string {
  return `goclaw-team-state is-${presentation.tone}`;
}
