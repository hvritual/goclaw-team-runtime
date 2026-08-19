export type AcceptanceResult = "accepted" | "conditional" | "rejected";

export interface AcceptanceConclusionInput {
  result: AcceptanceResult;
  rationale: string;
  evidenceRefs: string[];
}

export interface AcceptanceConclusion {
  id: string;
  workspaceId: string;
  issueId: string;
  result: AcceptanceResult;
  rationale: string;
  evidenceRefs: string[];
  actorId: string;
  createdAt: string;
  updatedAt: string;
}

export interface AcceptanceConclusionListResponse {
  acceptanceConclusions: AcceptanceConclusion[];
  total: number;
}

export type ProjectRetrospectiveStatus = "draft" | "published" | "archived";
export type ProjectRetrospectiveRevisionStatus =
  | ProjectRetrospectiveStatus
  | "superseded";
export type ProjectRetrospectiveAction =
  | "create"
  | "save_draft"
  | "publish"
  | "publish_revision"
  | "archive";
export type ProjectRetrospectiveMutationAction = Exclude<
  ProjectRetrospectiveAction,
  "create" | "archive"
>;
export type ProjectRetrospectiveParticipantRole =
  | "participant"
  | "facilitator";
export type ProjectRetrospectiveTargetKind = "task" | "issue";
export type ProjectRetrospectiveTargetState = "pending" | "linked";

export interface ProjectRetrospectiveActionItemInput {
  /** Omitted for draft creation; required when retaining an existing item. */
  id?: string;
  title: string;
  description?: string;
  assigneeId?: string;
  dueDate?: string;
}

export interface ProjectRetrospectiveActionItem
  extends ProjectRetrospectiveActionItemInput {
  id: string;
}

export interface ProjectRetrospectiveContentInput {
  summary: string;
  successes: string[];
  problems: string[];
  lessons: string[];
  actionItems: ProjectRetrospectiveActionItemInput[];
}

export interface ProjectRetrospectiveContent
  extends Omit<ProjectRetrospectiveContentInput, "actionItems"> {
  actionItems: ProjectRetrospectiveActionItem[];
}

export interface ProjectRetrospectiveParticipantInput {
  memberId: string;
  role: ProjectRetrospectiveParticipantRole;
}

export type ProjectRetrospectiveParticipant =
  ProjectRetrospectiveParticipantInput;

export interface ProjectRetrospectiveRevision {
  revision: number;
  status: ProjectRetrospectiveRevisionStatus;
  action: ProjectRetrospectiveAction;
  content: ProjectRetrospectiveContent;
  participants: ProjectRetrospectiveParticipant[];
  actorId: string;
  createdAt: string;
}

export interface ProjectRetrospectiveActionLink {
  retrospectiveId: string;
  actionItemId: string;
  sourceRevision: number;
  state: ProjectRetrospectiveTargetState;
  targetKind: ProjectRetrospectiveTargetKind;
  targetId?: string;
  createdBy: string;
  createdAt: string;
}

export interface ProjectRetrospectiveAccess {
  canEdit: boolean;
  canPublish: boolean;
  canArchive: boolean;
}

export interface ProjectRetrospectiveInput {
  content: ProjectRetrospectiveContentInput;
  participants: ProjectRetrospectiveParticipantInput[];
  idempotencyKey?: string;
}

export type ProjectRetrospectiveUpdateInput =
  | {
      expectedRevision: number;
      action: "publish";
      content?: never;
      participants?: never;
    }
  | {
      expectedRevision: number;
      action: "save_draft" | "publish_revision";
      content: ProjectRetrospectiveContentInput;
      participants: ProjectRetrospectiveParticipantInput[];
    };

export interface ProjectRetrospectiveTargetInput {
  targetKind?: ProjectRetrospectiveTargetKind;
  idempotencyKey?: string;
}

export interface ProjectRetrospective {
  id: string;
  workspaceId: string;
  projectId: string;
  status: ProjectRetrospectiveStatus;
  currentRevision: number;
  publishedRevision?: number;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  current: ProjectRetrospectiveRevision;
  history: ProjectRetrospectiveRevision[];
  actionLinks: ProjectRetrospectiveActionLink[];
  access: ProjectRetrospectiveAccess;
}

export interface ProjectRetrospectiveListParams {
  limit?: number;
  cursor?: string;
  includeArchived?: boolean;
}

export interface ProjectRetrospectiveListResponse {
  retrospectives: ProjectRetrospective[];
  nextCursor?: string;
}
