export type KnowledgeKind =
  | "goal"
  | "decision"
  | "constraint"
  | "requirement"
  | "procedure"
  | "lesson"
  | "reference";

export type KnowledgeStatus =
  | "candidate"
  | "in_review"
  | "published"
  | "superseded"
  | "rejected"
  | "quarantined";

export interface KnowledgeSourceRef {
  type: string;
  id: string;
  revision: string;
  uri: string;
  checksum: string;
  metadata?: Record<string, string>;
}

export interface KnowledgeRevision {
  number: number;
  supersedesRevision: number;
  title: string;
  content: string;
  createdBy: string;
  createdAt: string;
  sourceRefs: KnowledgeSourceRef[];
}

export interface KnowledgeEntry {
  id: string;
  workspaceId: string;
  projectId: string | null;
  candidateId: string | null;
  kind: KnowledgeKind;
  status: KnowledgeStatus;
  currentRevision: number;
  revisions: KnowledgeRevision[];
  createdAt: string;
  updatedAt: string;
  citation?: string;
  score?: number;
}

export interface KnowledgeCandidate {
  id: string;
  workspaceId: string;
  projectId: string | null;
  knowledgeId: string | null;
  targetRevision: number;
  kind: KnowledgeKind;
  title: string;
  content: string;
  reason: string;
  status: KnowledgeStatus;
  revision: number;
  proposedBy: string;
  sourceRefs: KnowledgeSourceRef[];
  createdAt: string;
  updatedAt: string;
}

export interface KnowledgeListResponse {
  entries: KnowledgeEntry[];
  total: number;
  nextCursor: string | null;
}

export interface KnowledgeCandidateListResponse {
  candidates: KnowledgeCandidate[];
  total: number;
  nextCursor: string | null;
}

export interface CommentKnowledgeProposalResponse {
  queued: boolean;
  evidenceId: string | null;
  sourceRevision: string;
}

export interface ProposeKnowledgeRequest {
  projectId?: string;
  knowledgeId?: string;
  kind: KnowledgeKind;
  title: string;
  content: string;
  reason: string;
  sourceRefs?: KnowledgeSourceRef[];
}

export type KnowledgeReviewAction = "approve" | "reject" | "quarantine";

export interface ReviewKnowledgeRequest {
  action: KnowledgeReviewAction;
  expectedRevision: number;
  rationale: string;
}

export interface ReviewKnowledgeResponse {
  candidate: KnowledgeCandidate;
  entry: KnowledgeEntry | null;
}
