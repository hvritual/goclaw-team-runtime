import type { Issue, IssueMetadata, IssueReaction } from "./issue";
import type { IssueProperty, IssuePropertyValues } from "./property";
import type { Comment, Reaction } from "./comment";
import type { TimelineEntry } from "./activity";
import type { Workspace, MemberWithUser, Invitation } from "./workspace";
import type { Project } from "./project";
import type { Label } from "./label";
import type { Task } from "./task";
import type { Skill } from "./skill";
import { z } from "zod";

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

const RealtimeIssueSchema = z.object({
  id: z.string().min(1),
  workspace_id: z.string().min(1),
  number: z.number(),
  identifier: z.string().min(1),
  title: z.string(),
  description: z.string().nullable(),
  status: z.enum(["backlog", "todo", "in_progress", "in_review", "done", "blocked", "cancelled"]),
  priority: z.enum(["urgent", "high", "medium", "low", "none"]),
  assignee_type: z.enum(["member", "agent"]).nullable(),
  assignee_id: z.string().nullable(),
  creator_type: z.enum(["member", "agent"]),
  creator_id: z.string(),
  parent_issue_id: z.string().nullable(),
  project_id: z.string().nullable(),
  position: z.number(),
  stage: z.number().nullable(),
  start_date: z.iso.date().nullable(),
  due_date: z.iso.date().nullable(),
  metadata: z.record(z.string(), z.union([z.string(), z.number(), z.boolean()])),
  properties: z.record(z.string(), z.union([z.string(), z.number(), z.boolean(), z.array(z.string())])),
  created_at: z.iso.datetime({ offset: true }),
  updated_at: z.iso.datetime({ offset: true }),
}).passthrough();

const IssueCreatedEventPayloadSchema = z.object({ issue: RealtimeIssueSchema }).passthrough();
const IssueUpdatedEventPayloadSchema = z.object({
  issue: RealtimeIssueSchema,
  assignee_changed: z.boolean().optional(),
  status_changed: z.boolean().optional(),
  project_changed: z.boolean().optional(),
}).passthrough();
const IssueDeletedEventPayloadSchema = z.object({ issue_id: z.string().min(1) }).passthrough();
const IssueMetadataChangedEventPayloadSchema = z.object({
  issue_id: z.string().min(1),
  metadata: z.record(z.string(), z.union([z.string(), z.number(), z.boolean()])),
}).passthrough();

const RealtimeReactionSchema = z.object({
  id: z.string().min(1),
  comment_id: z.string().min(1),
  actor_type: z.enum(["member", "agent"]),
  actor_id: z.string().min(1),
  emoji: z.string().min(1),
  created_at: z.iso.datetime({ offset: true }),
}).passthrough();
const RealtimeIssueReactionSchema = RealtimeReactionSchema.omit({ comment_id: true }).extend({
  issue_id: z.string().min(1),
}).passthrough();
const RealtimeCommentSchema = z.object({
  id: z.string().min(1),
  issue_id: z.string().min(1),
  author_type: z.enum(["member", "agent", "system"]),
  author_id: z.string().min(1),
  content: z.string(),
  type: z.enum(["comment", "status_change", "progress_update", "system"]),
  parent_id: z.string().nullable(),
  reactions: z.array(RealtimeReactionSchema),
  attachments: z.array(z.object({ id: z.string().min(1) }).passthrough()),
  created_at: z.iso.datetime({ offset: true }),
  updated_at: z.iso.datetime({ offset: true }),
  resolved_at: z.iso.datetime({ offset: true }).nullable(),
  resolved_by_type: z.enum(["member", "agent"]).nullable(),
  resolved_by_id: z.string().nullable(),
}).passthrough();
const RealtimeActivitySchema = z.object({
  type: z.literal("activity"),
  id: z.string().min(1),
  actor_type: z.enum(["member", "agent", "system"]),
  actor_id: z.string().min(1),
  action: z.string().min(1),
  details: z.record(z.string(), z.unknown()),
  created_at: z.iso.datetime({ offset: true }),
}).passthrough();
const CommentEventPayloadSchema = z.object({ comment: RealtimeCommentSchema }).passthrough();
const CommentDeletedEventPayloadSchema = z.object({
  comment_id: z.string().min(1),
  issue_id: z.string().min(1),
}).passthrough();
const ReactionAddedEventPayloadSchema = z.object({
  reaction: RealtimeReactionSchema,
  issue_id: z.string().min(1),
}).passthrough();
const ReactionRemovedEventPayloadSchema = z.object({
  comment_id: z.string().min(1),
  issue_id: z.string().min(1),
  emoji: z.string().min(1),
  actor_type: z.enum(["member", "agent"]),
  actor_id: z.string().min(1),
}).passthrough();
const IssueReactionAddedEventPayloadSchema = z.object({
  reaction: RealtimeIssueReactionSchema,
  issue_id: z.string().min(1),
}).passthrough();
const IssueReactionRemovedEventPayloadSchema = z.object({
  issue_id: z.string().min(1),
  emoji: z.string().min(1),
  actor_type: z.enum(["member", "agent"]),
  actor_id: z.string().min(1),
}).passthrough();
const SubscriberAddedEventPayloadSchema = z.object({
  issue_id: z.string().min(1),
  user_type: z.literal("member"),
  user_id: z.string().min(1),
  reason: z.enum(["creator", "assignee", "commenter", "mentioned", "manual"]),
}).passthrough();
const SubscriberRemovedEventPayloadSchema = SubscriberAddedEventPayloadSchema.omit({ reason: true }).passthrough();
const ActivityCreatedEventPayloadSchema = z.object({
  issue_id: z.string().min(1),
  entry: RealtimeActivitySchema,
}).passthrough();

/** Validates Canonical Issue and collaboration events before cache consumers run. */
export function isValidCanonicalIssueEventPayload(type: string, payload: unknown): boolean {
  switch (type) {
    case "issue:created": return IssueCreatedEventPayloadSchema.safeParse(payload).success;
    case "issue:updated": return IssueUpdatedEventPayloadSchema.safeParse(payload).success;
    case "issue:deleted": return IssueDeletedEventPayloadSchema.safeParse(payload).success;
    case "issue_metadata:changed": return IssueMetadataChangedEventPayloadSchema.safeParse(payload).success;
    case "comment:created":
    case "comment:updated":
    case "comment:resolved":
    case "comment:unresolved": return CommentEventPayloadSchema.safeParse(payload).success;
    case "comment:deleted": return CommentDeletedEventPayloadSchema.safeParse(payload).success;
    case "reaction:added": return ReactionAddedEventPayloadSchema.safeParse(payload).success;
    case "reaction:removed": return ReactionRemovedEventPayloadSchema.safeParse(payload).success;
    case "issue_reaction:added": return IssueReactionAddedEventPayloadSchema.safeParse(payload).success;
    case "issue_reaction:removed": return IssueReactionRemovedEventPayloadSchema.safeParse(payload).success;
    case "subscriber:added": return SubscriberAddedEventPayloadSchema.safeParse(payload).success;
    case "subscriber:removed": return SubscriberRemovedEventPayloadSchema.safeParse(payload).success;
    case "activity:created": return ActivityCreatedEventPayloadSchema.safeParse(payload).success;
    default: return true;
  }
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
        : E extends "issue_metadata:changed" ? IssueMetadataChangedPayload
        : unknown;
