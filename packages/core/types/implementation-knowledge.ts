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

export interface ProjectRetrospectiveInput {
  summary: string;
  successes: string[];
  problems: string[];
  lessons: string[];
  followUpRefs: string[];
}

export interface ProjectRetrospective extends ProjectRetrospectiveInput {
  id: string;
  workspaceId: string;
  projectId: string;
  actorId: string;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectRetrospectiveListResponse {
  retrospectives: ProjectRetrospective[];
  total: number;
}
