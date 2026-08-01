import { z } from "zod";
import type { ProjectRequirementBaseline, ProjectRequirementBaselineResponse, ProjectRequirementContent, ProjectRequirementRevision } from "../types";

const itemSchema = z.object({ key: z.string(), text: z.string() }).loose();
const baselineStatusSchema = z.enum(["draft", "in_review", "approved"]).catch("draft");
const revisionStateSchema = z.enum(["draft", "in_review", "approved", "superseded"]).catch("superseded");

export const projectRequirementContentSchema = z.object({
  problem_statement: z.string().default(""),
  goals: z.array(itemSchema).default([]),
  in_scope: z.array(itemSchema).default([]),
  out_of_scope: z.array(itemSchema).default([]),
  constraints: z.array(itemSchema).default([]),
  acceptance_criteria: z.array(itemSchema).default([]),
  dependencies: z.array(itemSchema).default([]),
}).loose().transform((value) => ({
  problemStatement: value.problem_statement,
  goals: value.goals,
  inScope: value.in_scope,
  outOfScope: value.out_of_scope,
  constraints: value.constraints,
  acceptanceCriteria: value.acceptance_criteria,
  dependencies: value.dependencies,
})) as z.ZodType<ProjectRequirementContent>;

const baselineSchema = z.object({
  id: z.string(), workspace_id: z.string(), project_id: z.string(), status: baselineStatusSchema,
  current_revision: z.number(), approved_revision: z.number().nullable(),
  submitted_by: z.string().nullable().default(null), submitted_at: z.string().nullable().default(null),
  approved_by: z.string().nullable().default(null), approved_at: z.string().nullable().default(null),
  created_at: z.string(), updated_at: z.string(),
}).loose().transform((value) => ({
  id: value.id, workspaceId: value.workspace_id, projectId: value.project_id, status: value.status,
  currentRevision: value.current_revision, approvedRevision: value.approved_revision,
  submittedBy: value.submitted_by, submittedAt: value.submitted_at, approvedBy: value.approved_by, approvedAt: value.approved_at,
  createdAt: value.created_at, updatedAt: value.updated_at,
})) as z.ZodType<ProjectRequirementBaseline>;

const revisionSchema = z.object({
  baseline_id: z.string(), revision: z.number(), content: projectRequirementContentSchema,
  change_summary: z.string(), actor_id: z.string(), created_at: z.string(), state: revisionStateSchema,
  submitted_by: z.string().nullable().default(null), submitted_at: z.string().nullable().default(null),
  approved_by: z.string().nullable().default(null), approved_at: z.string().nullable().default(null),
}).loose().transform((value) => ({
  baselineId: value.baseline_id, revision: value.revision, content: value.content,
  changeSummary: value.change_summary, actorId: value.actor_id, createdAt: value.created_at, state: value.state,
  submittedBy: value.submitted_by, submittedAt: value.submitted_at, approvedBy: value.approved_by, approvedAt: value.approved_at,
})) as z.ZodType<ProjectRequirementRevision>;

export const projectRequirementBaselineResponseSchema = z.object({
  baseline: baselineSchema.nullable(),
  current_content: projectRequirementContentSchema.nullable().default(null),
  effective_content: projectRequirementContentSchema.nullable().default(null),
  history: z.array(revisionSchema).default([]),
}).loose().transform((value) => ({
  baseline: value.baseline,
  currentContent: value.current_content,
  effectiveContent: value.effective_content,
  history: value.history,
})) as z.ZodType<ProjectRequirementBaselineResponse>;

export const EMPTY_PROJECT_REQUIREMENT_BASELINE: ProjectRequirementBaselineResponse = {
  baseline: null, currentContent: null, effectiveContent: null, history: [],
};
