import { TeamIssue, TeamWorkItem } from './types';

type IssueStatus = TeamIssue['status'];
type WorkItemStatus = TeamWorkItem['status'];

const issueTransitions: Record<IssueStatus, IssueStatus[]> = {
  new: ['triaged', 'cancelled'],
  triaged: ['assigned', 'in_progress', 'blocked', 'cancelled'],
  assigned: ['in_progress', 'blocked', 'cancelled'],
  in_progress: ['verifying', 'blocked', 'cancelled'],
  blocked: ['triaged', 'assigned', 'in_progress', 'cancelled'],
  verifying: ['resolved', 'in_progress', 'blocked'],
  resolved: ['closed', 'reopened'],
  closed: ['reopened'],
  reopened: ['triaged', 'assigned', 'in_progress', 'cancelled'],
  cancelled: [],
};

const workItemTransitions: Record<WorkItemStatus, WorkItemStatus[]> = {
  pending: ['ready', 'cancelled'],
  ready: ['in_progress', 'blocked', 'cancelled'],
  in_progress: ['verifying', 'blocked', 'cancelled'],
  blocked: ['ready', 'in_progress', 'cancelled'],
  verifying: ['done', 'in_progress', 'blocked'],
  done: [],
  cancelled: [],
};

export function nextIssueStatuses(status: IssueStatus): IssueStatus[] {
  return issueTransitions[status] ?? [];
}

export function nextWorkItemStatuses(status: WorkItemStatus): WorkItemStatus[] {
  return workItemTransitions[status] ?? [];
}
