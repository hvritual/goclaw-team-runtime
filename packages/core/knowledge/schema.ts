import { z } from "zod";
import type {
  KnowledgeCandidate,
  KnowledgeCandidateListResponse,
  KnowledgeEntry,
  KnowledgeListResponse,
  ReviewKnowledgeResponse,
  CommentKnowledgeProposalResponse,
} from "../types";

export const commentKnowledgeProposalResponseSchema = z
  .object({
    queued: z.boolean(),
    evidence_id: z.string().nullable(),
    source_revision: z.string(),
  })
  .loose()
  .transform((value) => ({
    queued: value.queued,
    evidenceId: value.evidence_id,
    sourceRevision: value.source_revision,
  })) as z.ZodType<CommentKnowledgeProposalResponse>;

const knowledgeKindSchema = z.enum([
  "goal",
  "decision",
  "constraint",
  "requirement",
  "procedure",
  "lesson",
  "reference",
]);

const candidateKnowledgeStatusSchema = z.enum([
  "candidate",
  "in_review",
  "published",
  "rejected",
  "quarantined",
]);

const governedSourceRefSchema = z
  .object({
    type: z.string().min(1),
    id: z.string().min(1),
    revision: z.string().min(1),
    citation: z.string().min(1),
    asset_id: z.string().min(1).nullable(),
    asset_version_id: z.string().min(1).nullable(),
  })
  .strict()
  .transform((value) => ({
    type: value.type,
    id: value.id,
    revision: value.revision,
    citation: value.citation,
    assetId: value.asset_id,
    assetVersionId: value.asset_version_id,
  }));

const revisionSchema = z
  .object({
    number: z.number().int().positive(),
    supersedes_revision: z.number().int().nonnegative(),
    title: z.string().min(1),
    content: z.string(),
    created_by: z.string().min(1),
    created_at: z.string().min(1),
    source_refs: z.array(governedSourceRefSchema),
  })
  .strict()
  .transform((value) => ({
    number: value.number,
    supersedesRevision: value.supersedes_revision,
    title: value.title,
    content: value.content,
    createdBy: value.created_by,
    createdAt: value.created_at,
    sourceRefs: value.source_refs,
  }));

export const knowledgeEntrySchema = z
  .object({
    id: z.string().min(1),
    workspace_id: z.string().min(1),
    project_id: z.string().min(1).nullable(),
    candidate_id: z.string().min(1).nullable(),
    kind: z.enum([
      "goal",
      "decision",
      "constraint",
      "requirement",
      "procedure",
      "lesson",
      "reference",
    ]),
    status: z.enum(["published", "superseded", "quarantined"]),
    current_revision: z.number().int().positive(),
    revision: revisionSchema,
    revisions: z.array(revisionSchema).optional(),
    created_at: z.string().min(1),
    updated_at: z.string().min(1),
    citation: z.string(),
    matched_by: z.enum([
      "recent",
      "title_exact",
      "title_prefix",
      "title",
      "content",
      "source",
      "detail",
    ]),
  })
  .strict()
  .transform((value) => ({
    id: value.id,
    workspaceId: value.workspace_id,
    projectId: value.project_id,
    candidateId: value.candidate_id,
    kind: value.kind,
    status: value.status,
    currentRevision: value.current_revision,
    revisions: value.revisions ?? [value.revision],
    createdAt: value.created_at,
    updatedAt: value.updated_at,
    citation: value.citation,
    matchedBy: value.matched_by,
  })) as z.ZodType<KnowledgeEntry>;

export const knowledgeCandidateSchema = z
  .object({
    id: z.string().min(1),
    workspace_id: z.string().min(1),
    project_id: z.string().min(1).nullable(),
    knowledge_id: z.string().min(1).nullable(),
    target_revision: z.number().int().nonnegative(),
    kind: knowledgeKindSchema,
    title: z.string(),
    content: z.string(),
    reason: z.string(),
    status: candidateKnowledgeStatusSchema,
    revision: z.number().int().positive(),
    proposed_by: z.string().min(1),
    source_refs: z.array(governedSourceRefSchema),
    created_at: z.string().min(1),
    updated_at: z.string().min(1),
  })
  .strict()
  .transform((value) => ({
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

export const knowledgeListSchema = z
  .object({
    entries: z.array(knowledgeEntrySchema),
    total: z.number().int().nonnegative(),
    next_cursor: z.string().min(1).nullable(),
  })
  .strict()
  .transform((value) => ({
    entries: value.entries,
    total: value.total,
    nextCursor: value.next_cursor,
  })) as z.ZodType<KnowledgeListResponse>;

export const knowledgeCandidateListSchema = z
  .object({
    candidates: z.array(knowledgeCandidateSchema),
    total: z.number().int().nonnegative(),
    next_cursor: z.string().min(1).nullable(),
  })
  .strict()
  .transform((value) => ({
    candidates: value.candidates,
    total: value.total,
    nextCursor: value.next_cursor,
  })) as z.ZodType<KnowledgeCandidateListResponse>;

export const reviewKnowledgeResponseSchema = z
  .object({
    candidate: knowledgeCandidateSchema,
    entry: knowledgeEntrySchema.nullable(),
  })
  .strict() as z.ZodType<ReviewKnowledgeResponse>;

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
