import { z } from "zod";
import type {
  AcceptanceConclusion,
  AcceptanceConclusionListResponse,
  ProjectRetrospective,
  ProjectRetrospectiveActionItem,
  ProjectRetrospectiveActionLink,
  ProjectRetrospectiveAccess,
  ProjectRetrospectiveContent,
  ProjectRetrospectiveListResponse,
  ProjectRetrospectiveParticipant,
  ProjectRetrospectiveRevision,
} from "../types/implementation-knowledge";

const nonEmpty = z.string().min(1).refine((value) => value === value.trim(), {
  message: "Expected canonical non-whitespace-bounded text",
});
const timestamp = z.iso.datetime({ offset: true });
const dueDate = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

export const acceptanceConclusionSchema = z.object({
  id: z.string().min(1),
  workspace_id: z.string().min(1),
  issue_id: z.string().min(1),
  result: z.enum(["accepted", "conditional", "rejected"]),
  rationale: z.string().min(1),
  evidence_refs: z.array(z.string()),
  actor_id: z.string().min(1),
  created_at: timestamp,
  updated_at: timestamp,
}).loose().transform((value) => ({
  id: value.id,
  workspaceId: value.workspace_id,
  issueId: value.issue_id,
  result: value.result,
  rationale: value.rationale,
  evidenceRefs: value.evidence_refs,
  actorId: value.actor_id,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
})) as z.ZodType<AcceptanceConclusion>;

export const acceptanceConclusionListSchema = z.object({
  acceptance_conclusions: z.array(acceptanceConclusionSchema),
  total: z.number(),
}).loose().transform((value) => ({
  acceptanceConclusions: value.acceptance_conclusions,
  total: value.total,
})) as z.ZodType<AcceptanceConclusionListResponse>;

export const projectRetrospectiveActionItemSchema = z.object({
  id: nonEmpty.max(64).regex(/^[A-Za-z0-9_-]+$/),
  title: nonEmpty.max(500),
  description: nonEmpty.max(5000).optional(),
  assignee_id: nonEmpty.optional(),
  due_date: dueDate.optional(),
}).strict().transform((value) => ({
  id: value.id,
  title: value.title,
  ...(value.description === undefined ? {} : { description: value.description }),
  ...(value.assignee_id === undefined ? {} : { assigneeId: value.assignee_id }),
  ...(value.due_date === undefined ? {} : { dueDate: value.due_date }),
})) as z.ZodType<ProjectRetrospectiveActionItem>;

export const projectRetrospectiveContentSchema = z.object({
  summary: nonEmpty.max(5000),
  successes: z.array(nonEmpty.max(2000)).max(100),
  problems: z.array(nonEmpty.max(2000)).max(100),
  lessons: z.array(nonEmpty.max(2000)).min(1).max(100),
  action_items: z.array(projectRetrospectiveActionItemSchema).max(100),
}).strict().superRefine((value, context) => {
  for (const key of ["successes", "problems", "lessons"] as const) {
    if (new Set(value[key]).size !== value[key].length) {
      context.addIssue({ code: "custom", path: [key], message: "Duplicate content item" });
    }
  }
  const actionIds = new Set<string>();
  for (const [index, item] of value.action_items.entries()) {
    if (actionIds.has(item.id)) {
      context.addIssue({
        code: "custom",
        path: ["action_items", index, "id"],
        message: "Duplicate action item ID",
      });
    }
    actionIds.add(item.id);
  }
}).transform((value) => ({
  summary: value.summary,
  successes: value.successes,
  problems: value.problems,
  lessons: value.lessons,
  actionItems: value.action_items,
})) as z.ZodType<ProjectRetrospectiveContent>;

export const projectRetrospectiveParticipantSchema = z.object({
  member_id: nonEmpty,
  role: z.enum(["participant", "facilitator"]),
}).strict().transform((value) => ({
  memberId: value.member_id,
  role: value.role,
})) as z.ZodType<ProjectRetrospectiveParticipant>;

export const projectRetrospectiveRevisionSchema = z.object({
  revision: z.number().int().positive(),
  status: z.enum(["draft", "published", "superseded", "archived"]),
  action: z.enum(["create", "save_draft", "publish", "publish_revision", "archive"]),
  content: projectRetrospectiveContentSchema,
  participants: z.array(projectRetrospectiveParticipantSchema).max(100),
  actor_id: nonEmpty,
  created_at: timestamp,
}).strict().superRefine((value, context) => {
  const members = new Set<string>();
  for (const [index, participant] of value.participants.entries()) {
    if (members.has(participant.memberId)) {
      context.addIssue({
        code: "custom",
        path: ["participants", index, "member_id"],
        message: "Duplicate participant",
      });
    }
    members.add(participant.memberId);
  }
}).transform((value) => ({
  revision: value.revision,
  status: value.status,
  action: value.action,
  content: value.content,
  participants: value.participants,
  actorId: value.actor_id,
  createdAt: value.created_at,
})) as z.ZodType<ProjectRetrospectiveRevision>;

export const projectRetrospectiveActionLinkSchema = z.object({
  retrospective_id: nonEmpty,
  action_item_id: nonEmpty.max(64),
  source_revision: z.number().int().positive(),
  state: z.enum(["pending", "linked"]),
  target_kind: z.enum(["task", "issue"]),
  target_id: nonEmpty.optional(),
  created_by: nonEmpty,
  created_at: timestamp,
}).strict().superRefine((value, context) => {
  if (value.state === "linked" && value.target_id === undefined) {
    context.addIssue({ code: "custom", path: ["target_id"], message: "Linked target ID is required" });
  }
  if (value.state === "pending" && value.target_id !== undefined) {
    context.addIssue({ code: "custom", path: ["target_id"], message: "Pending target ID is not allowed" });
  }
}).transform((value) => ({
  retrospectiveId: value.retrospective_id,
  actionItemId: value.action_item_id,
  sourceRevision: value.source_revision,
  state: value.state,
  targetKind: value.target_kind,
  ...(value.target_id === undefined ? {} : { targetId: value.target_id }),
  createdBy: value.created_by,
  createdAt: value.created_at,
})) as z.ZodType<ProjectRetrospectiveActionLink>;

export const projectRetrospectiveAccessSchema = z.object({
  can_edit: z.boolean(),
  can_publish: z.boolean(),
  can_archive: z.boolean(),
}).strict().transform((value) => ({
  canEdit: value.can_edit,
  canPublish: value.can_publish,
  canArchive: value.can_archive,
})) as z.ZodType<ProjectRetrospectiveAccess>;

export const projectRetrospectiveSchema = z.object({
  id: nonEmpty,
  workspace_id: nonEmpty,
  project_id: nonEmpty,
  status: z.enum(["draft", "published", "archived"]),
  current_revision: z.number().int().positive(),
  published_revision: z.number().int().positive().optional(),
  created_by: nonEmpty,
  created_at: timestamp,
  updated_at: timestamp,
  current: projectRetrospectiveRevisionSchema,
  history: z.array(projectRetrospectiveRevisionSchema).min(1),
  action_links: z.array(projectRetrospectiveActionLinkSchema),
  access: projectRetrospectiveAccessSchema,
}).strict().superRefine((value, context) => {
  if (value.current.revision !== value.current_revision || value.current.status !== value.status) {
    context.addIssue({ code: "custom", path: ["current"], message: "Current revision does not match the head" });
  }
  if (value.published_revision !== undefined && value.published_revision > value.current_revision) {
    context.addIssue({ code: "custom", path: ["published_revision"], message: "Published revision exceeds current revision" });
  }
  let previousRevision = 0;
  for (const [index, revision] of value.history.entries()) {
    if (revision.revision <= previousRevision) {
      context.addIssue({ code: "custom", path: ["history", index, "revision"], message: "History is not strictly ordered" });
    }
    previousRevision = revision.revision;
  }
  const last = value.history[value.history.length - 1];
  if (!last || last.revision !== value.current_revision || last.status !== value.status) {
    context.addIssue({ code: "custom", path: ["history"], message: "History does not end at current authority" });
  }
  const linkItems = new Set<string>();
  for (const [index, link] of value.action_links.entries()) {
    if (link.retrospectiveId !== value.id || link.sourceRevision > value.current_revision || linkItems.has(link.actionItemId)) {
      context.addIssue({ code: "custom", path: ["action_links", index], message: "Invalid action link provenance" });
    }
    linkItems.add(link.actionItemId);
  }
}).transform((value) => ({
  id: value.id,
  workspaceId: value.workspace_id,
  projectId: value.project_id,
  status: value.status,
  currentRevision: value.current_revision,
  ...(value.published_revision === undefined ? {} : { publishedRevision: value.published_revision }),
  createdBy: value.created_by,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
  current: value.current,
  history: value.history,
  actionLinks: value.action_links,
  access: value.access,
})) as z.ZodType<ProjectRetrospective>;

export const projectRetrospectiveListSchema = z.object({
  retrospectives: z.array(projectRetrospectiveSchema),
  next_cursor: nonEmpty.optional(),
}).strict().transform((value) => ({
  retrospectives: value.retrospectives,
  ...(value.next_cursor === undefined ? {} : { nextCursor: value.next_cursor }),
})) as z.ZodType<ProjectRetrospectiveListResponse>;

export const EMPTY_ACCEPTANCE_CONCLUSION_LIST: AcceptanceConclusionListResponse = {
  acceptanceConclusions: [],
  total: 0,
};
