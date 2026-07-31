import { z } from "zod";
import type {
  AcceptanceConclusion,
  AcceptanceConclusionListResponse,
  ProjectRetrospective,
  ProjectRetrospectiveListResponse,
} from "../types";

export const acceptanceConclusionSchema = z.object({
  id: z.string(),
  workspace_id: z.string(),
  issue_id: z.string(),
  result: z.enum(["accepted", "conditional", "rejected"]),
  rationale: z.string(),
  evidence_refs: z.array(z.string()).default([]),
  actor_id: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
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
  acceptance_conclusions: z.array(acceptanceConclusionSchema).default([]),
  total: z.number().default(0),
}).loose().transform((value) => ({
  acceptanceConclusions: value.acceptance_conclusions,
  total: value.total,
})) as z.ZodType<AcceptanceConclusionListResponse>;

export const projectRetrospectiveSchema = z.object({
  id: z.string(),
  workspace_id: z.string(),
  project_id: z.string(),
  summary: z.string(),
  successes: z.array(z.string()).default([]),
  problems: z.array(z.string()).default([]),
  lessons: z.array(z.string()).default([]),
  follow_up_refs: z.array(z.string()).default([]),
  actor_id: z.string(),
  created_at: z.string(),
  updated_at: z.string(),
}).loose().transform((value) => ({
  id: value.id,
  workspaceId: value.workspace_id,
  projectId: value.project_id,
  summary: value.summary,
  successes: value.successes,
  problems: value.problems,
  lessons: value.lessons,
  followUpRefs: value.follow_up_refs,
  actorId: value.actor_id,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
})) as z.ZodType<ProjectRetrospective>;

export const projectRetrospectiveListSchema = z.object({
  retrospectives: z.array(projectRetrospectiveSchema).default([]),
  total: z.number().default(0),
}).loose().transform((value) => ({
  retrospectives: value.retrospectives,
  total: value.total,
})) as z.ZodType<ProjectRetrospectiveListResponse>;

export const EMPTY_ACCEPTANCE_CONCLUSION_LIST: AcceptanceConclusionListResponse = {
  acceptanceConclusions: [],
  total: 0,
};

export const EMPTY_RETROSPECTIVE_LIST: ProjectRetrospectiveListResponse = {
  retrospectives: [],
  total: 0,
};
