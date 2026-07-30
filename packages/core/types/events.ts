import type { Issue, IssueMetadata, IssueReaction } from "./issue";
import type { IssueProperty, IssuePropertyValues } from "./property";
import type { Comment, Reaction } from "./comment";
import type { TimelineEntry } from "./activity";
import type { Workspace, MemberWithUser, Invitation } from "./workspace";
import type { Project } from "./project";
import type { Label } from "./label";
import type { Task } from "./task";
import type { Skill } from "./skill";

export type WSEventType =
  | "issue:created"
  | "issue:updated"
  | "issue:deleted"
  | "issue_labels:changed"
  | "issue_metadata:changed"
  | "issue_properties:changed"
  | "issue_reaction:added"
  | "issue_reaction:removed"
  | "comment:created"
  | "comment:updated"
  | "comment:deleted"
  | "comment:resolved"
  | "comment:unresolved"
  | "subscriber:added"
  | "subscriber:removed"
  | "activity:created"
  | "reaction:added"
  | "reaction:removed"
  | "workspace:updated"
  | "workspace:deleted"
  | "member:added"
  | "member:updated"
  | "member:removed"
  | "invitation:created"
  | "invitation:accepted"
  | "invitation:declined"
  | "invitation:revoked"
  | "project:created"
  | "project:updated"
  | "project:deleted"
  | "task:created"
  | "task:updated"
  | "task:deleted"
  | "skill:created"
  | "skill:updated"
  | "skill:deleted"
  | "label:created"
  | "label:updated"
  | "label:deleted"
  | "property:created"
  | "property:updated"
  | "pin:created"
  | "pin:deleted"
  | "pin:reordered";

export interface WSMessage<T = unknown> {
  type: WSEventType;
  payload: T;
  actor_id?: string;
  actor_type?: string;
}

export interface IssueCreatedPayload {
  issue: Issue;
}
export interface IssueUpdatedPayload {
  issue: Issue;
  assignee_changed?: boolean;
  status_changed?: boolean;
  project_changed?: boolean;
}
export interface IssueDeletedPayload {
  issue_id: string;
}
export interface IssueLabelsChangedPayload {
  issue_id: string;
  labels: Label[];
}
export interface IssueMetadataChangedPayload {
  issue_id: string;
  metadata: IssueMetadata;
}
export interface IssuePropertiesChangedPayload {
  issue_id: string;
  properties: IssuePropertyValues;
}
export interface PropertyChangedPayload {
  property: IssueProperty;
}

export interface CommentCreatedPayload {
  comment: Comment;
}
export interface CommentUpdatedPayload {
  comment: Comment;
}
export interface CommentDeletedPayload {
  comment_id: string;
  issue_id: string;
}
export interface CommentResolvedPayload {
  comment: Comment;
}
export interface CommentUnresolvedPayload {
  comment: Comment;
}
export interface SubscriberAddedPayload {
  issue_id: string;
  user_type: string;
  user_id: string;
  reason: string;
}
export interface SubscriberRemovedPayload {
  issue_id: string;
  user_type: string;
  user_id: string;
}
export interface ActivityCreatedPayload {
  issue_id: string;
  entry: TimelineEntry;
}
export interface ReactionAddedPayload {
  reaction: Reaction;
  issue_id: string;
}
export interface ReactionRemovedPayload {
  comment_id: string;
  issue_id: string;
  emoji: string;
  actor_type: string;
  actor_id: string;
}
export interface IssueReactionAddedPayload {
  reaction: IssueReaction;
  issue_id: string;
}
export interface IssueReactionRemovedPayload {
  issue_id: string;
  emoji: string;
  actor_type: string;
  actor_id: string;
}

export interface WorkspaceUpdatedPayload {
  workspace: Workspace;
}
export interface WorkspaceDeletedPayload {
  workspace_id: string;
}
export interface MemberUpdatedPayload {
  member: MemberWithUser;
}
export interface MemberAddedPayload {
  member: MemberWithUser;
  workspace_id: string;
  workspace_name?: string;
}
export interface MemberRemovedPayload {
  member_id: string;
  user_id: string;
  workspace_id: string;
}
export interface InvitationCreatedPayload {
  invitation: Invitation;
  workspace_name?: string;
}
export interface InvitationAcceptedPayload {
  invitation_id: string;
  member: MemberWithUser;
}
export interface InvitationDeclinedPayload {
  invitation_id: string;
  invitee_email: string;
}
export interface InvitationRevokedPayload {
  invitation_id: string;
  invitee_email: string;
}

export interface ProjectCreatedPayload {
  project: Project;
}
export interface ProjectUpdatedPayload {
  project: Project;
}
export interface ProjectDeletedPayload {
  project_id: string;
}
export interface TaskChangedPayload {
  task: Task;
}
export interface SkillChangedPayload {
  skill: Skill;
}

export type WSEventPayload<E extends WSEventType> =
  E extends "issue:created" ? IssueCreatedPayload
    : E extends "issue:updated" ? IssueUpdatedPayload
      : E extends "issue:deleted" ? IssueDeletedPayload
        : unknown;
