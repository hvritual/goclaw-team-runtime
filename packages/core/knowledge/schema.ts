import { z } from "zod";
import type {
  KnowledgeCandidate,
  KnowledgeCandidateListResponse,
  KnowledgeEntry,
  KnowledgeListResponse,
  ReviewKnowledgeResponse,
  CommentKnowledgeProposalResponse,
} from "../types";

export const commentKnowledgeProposalResponseSchema = z.object({
  queued: z.boolean(),
  evidence_id: z.string().nullable(),
  source_revision: z.string(),
}).loose().transform((value) => ({
  queued: value.queued,
  evidenceId: value.evidence_id,
  sourceRevision: value.source_revision,
})) as z.ZodType<CommentKnowledgeProposalResponse>;

const knowledgeKindSchema = z
  .enum([
    "goal",
    "decision",
    "constraint",
    "requirement",
    "procedure",
    "lesson",
    "reference",
  ])
  .catch("reference");

const publishedKnowledgeStatusSchema = z
  .enum(["published", "superseded"])
  .catch("published");

const candidateKnowledgeStatusSchema = z
  .enum([
    "candidate",
    "in_review",
    "published",
    "rejected",
    "quarantined",
  ])
  .catch("candidate");

const sourceRefSchema = z.object({
  type: z.string(),
  id: z.string(),
  revision: z.string().default(""),
  uri: z.string().default(""),
  checksum: z.string().default(""),
  metadata: z.record(z.string(), z.string()).optional(),
}).loose().transform((value) => ({
  type: value.type,
  id: value.id,
  revision: value.revision,
  uri: value.uri,
  checksum: value.checksum,
  metadata: value.metadata,
}));

const revisionSchema = z.object({
  Number: z.number().optional(),
  SupersedesRevision: z.number().optional(),
  Title: z.string().optional(),
  Content: z.string().optional(),
  CreatedBy: z.string().optional(),
  CreatedAt: z.string().optional(),
  SourceRefs: z.array(sourceRefSchema).optional(),
  number: z.number().optional(),
  supersedes_revision: z.number().optional(),
  title: z.string().optional(),
  content: z.string().optional(),
  created_by: z.string().optional(),
  created_at: z.string().optional(),
  source_refs: z.array(sourceRefSchema).optional(),
}).loose().transform((value) => ({
  number: value.number ?? value.Number ?? 0,
  supersedesRevision:
    value.supersedes_revision ?? value.SupersedesRevision ?? 0,
  title: value.title ?? value.Title ?? "",
  content: value.content ?? value.Content ?? "",
  createdBy: value.created_by ?? value.CreatedBy ?? "",
  createdAt: value.created_at ?? value.CreatedAt ?? "",
  sourceRefs: value.source_refs ?? value.SourceRefs ?? [],
}));

export const knowledgeEntrySchema = z.object({
  id: z.string(),
  workspace_id: z.string(),
  project_id: z.string().nullable().optional().default(null),
  candidate_id: z.string().nullable().optional().default(null),
  kind: knowledgeKindSchema,
  status: publishedKnowledgeStatusSchema,
  current_revision: z.number(),
  revisions: z.array(revisionSchema).default([]),
  created_at: z.string(),
  updated_at: z.string(),
  citation: z.string().optional(),
  score: z.number().optional(),
}).loose().transform((value) => ({
  id: value.id,
  workspaceId: value.workspace_id,
  projectId: value.project_id,
  candidateId: value.candidate_id,
  kind: value.kind,
  status: value.status,
  currentRevision: value.current_revision,
  revisions: value.revisions,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
  citation: value.citation,
  score: value.score,
})) as z.ZodType<KnowledgeEntry>;

export const knowledgeCandidateSchema = z.object({
  id: z.string(),
  workspace_id: z.string(),
  project_id: z.string().nullable().optional().default(null),
  knowledge_id: z.string().nullable().optional().default(null),
  target_revision: z.number().optional().default(0),
  kind: knowledgeKindSchema,
  title: z.string(),
  content: z.string(),
  reason: z.string(),
  status: candidateKnowledgeStatusSchema,
  revision: z.number(),
  proposed_by: z.string(),
  source_refs: z.array(sourceRefSchema).default([]),
  created_at: z.string(),
  updated_at: z.string(),
}).loose().transform((value) => ({
  id: value.id,
  workspaceId: value.workspace_id,
  projectId: value.project_id,
  knowledgeId: value.knowledge_id,
  targetRevision: value.target_revision,
  kind: value.kind,
  title: value.title,
  content: value.content,
  reason: value.reason,
  status: value.status,
  revision: value.revision,
  proposedBy: value.proposed_by,
  sourceRefs: value.source_refs,
  createdAt: value.created_at,
  updatedAt: value.updated_at,
})) as z.ZodType<KnowledgeCandidate>;

export const knowledgeListSchema = z.object({
  entries: z.array(knowledgeEntrySchema).default([]),
  total: z.number().default(0),
  next_cursor: z.string().nullable().optional().default(null),
}).loose().transform((value) => ({
  entries: value.entries,
  total: value.total,
  nextCursor: value.next_cursor,
})) as z.ZodType<KnowledgeListResponse>;

export const knowledgeCandidateListSchema = z.object({
  candidates: z.array(knowledgeCandidateSchema).default([]),
  total: z.number().default(0),
  next_cursor: z.string().nullable().optional().default(null),
}).loose().transform((value) => ({
  candidates: value.candidates,
  total: value.total,
  nextCursor: value.next_cursor,
})) as z.ZodType<KnowledgeCandidateListResponse>;

export const reviewKnowledgeResponseSchema = z.object({
  candidate: knowledgeCandidateSchema,
  entry: knowledgeEntrySchema.nullable(),
}).loose() as z.ZodType<ReviewKnowledgeResponse>;

export const EMPTY_KNOWLEDGE_LIST: KnowledgeListResponse = {
  entries: [],
  total: 0,
  nextCursor: null,
};

export const EMPTY_KNOWLEDGE_CANDIDATE_LIST: KnowledgeCandidateListResponse = {
  candidates: [],
  total: 0,
  nextCursor: null,
};
