import { z } from "zod";

const IdentifierSchema = z.string().min(1).max(64);
const TimestampSchema = z.string().min(1);

export const TeamControlWorkspaceSchema = z.object({
  id: IdentifierSchema,
  name: z.string().min(1),
  state: z.string().min(1),
  version: z.number().int().positive(),
  created_at: TimestampSchema,
  updated_at: TimestampSchema,
}).loose();

export const TeamControlWorkspaceResponseSchema = z.object({
  schema_version: z.literal(1),
  workspace: TeamControlWorkspaceSchema,
}).loose();

export const TeamControlMemberSchema = z.object({
  workspace_id: IdentifierSchema,
  id: IdentifierSchema,
  kind: z.string().min(1),
  role: z.string().min(1),
  state: z.string().min(1),
  version: z.number().int().positive(),
  created_at: TimestampSchema,
  updated_at: TimestampSchema,
}).loose();

export const TeamControlMembersResponseSchema = z.object({
  schema_version: z.literal(1),
  members: z.array(TeamControlMemberSchema),
}).loose();

export const TeamControlWorkNodeSchema = z.object({
  id: IdentifierSchema,
  kind: z.string().min(1),
  revision: z.number().int().positive(),
  state: z.string().min(1),
  creator_id: IdentifierSchema,
  assignee_ids: z.array(IdentifierSchema),
  executor_ids: z.array(IdentifierSchema),
  data: z.unknown().optional(),
}).loose();

export const TeamControlWorkEdgeSchema = z.object({
  id: IdentifierSchema,
  from: IdentifierSchema,
  to: IdentifierSchema,
  kind: z.string().min(1),
}).loose();

export const TeamControlEvidenceSchema = z.object({
  id: IdentifierSchema,
  subject_id: IdentifierSchema,
  kind: z.string().min(1),
  uri: z.string().min(1),
  sha256: z.string().regex(/^[0-9a-f]{64}$/),
  size: z.number().int().nonnegative(),
  media_type: z.string().min(1),
  produced_by: IdentifierSchema,
  run_id: IdentifierSchema.optional(),
  sanitized: z.boolean(),
  captured_at: TimestampSchema,
}).loose();

export const TeamControlCheckSchema = z.object({
  id: IdentifierSchema,
  policy_id: z.string().min(1),
  subject_id: IdentifierSchema,
  revision: z.number().int().positive(),
  outcome: z.string().min(1),
  evidence_ids: z.array(IdentifierSchema),
  checker_id: IdentifierSchema,
  deterministic: z.boolean(),
  recorded_at: TimestampSchema,
}).loose();

export const TeamControlAcceptanceSchema = z.object({
  subject_id: IdentifierSchema,
  revision: z.number().int().positive(),
  acceptor_id: IdentifierSchema,
  policy_ids: z.array(z.string().min(1)),
  accepted_at: TimestampSchema,
}).loose();

export const TeamControlProjectionSchema = z.object({
  schema_version: z.literal(1),
  workspace_id: IdentifierSchema,
  project_id: IdentifierSchema,
  head: z.number().int().nonnegative(),
  head_hash: z.string(),
  nodes: z.record(z.string(), TeamControlWorkNodeSchema),
  edges: z.record(z.string(), TeamControlWorkEdgeSchema),
  evidence: z.record(z.string(), TeamControlEvidenceSchema),
  checks: z.record(z.string(), z.array(TeamControlCheckSchema)),
  acceptances: z.record(z.string(), TeamControlAcceptanceSchema),
}).loose();

export const TeamControlSessionEventSchema = z.object({
  schema_version: z.literal(1),
  workspace_id: IdentifierSchema,
  project_id: IdentifierSchema,
  sequence: z.number().int().positive(),
  event_id: z.string().min(1),
  command_id: IdentifierSchema,
  type: z.string().min(1),
  actor_id: IdentifierSchema,
  actor_kind: z.string().min(1),
  payload: z.unknown(),
  previous_hash: z.string(),
  hash: z.string().regex(/^[0-9a-f]{64}$/),
  occurred_at: TimestampSchema,
}).loose();

export const TeamControlAppendResultSchema = z.object({
  events: z.array(TeamControlSessionEventSchema),
  head: z.number().int().positive(),
  head_hash: z.string().regex(/^[0-9a-f]{64}$/),
  replayed: z.boolean(),
}).loose();

export const TeamControlProblemSchema = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number().int(),
  code: z.string(),
  detail: z.string(),
  field: z.string().optional(),
}).loose();
