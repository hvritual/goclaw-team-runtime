export type ProjectRequirementStatus = "draft" | "in_review" | "approved";
export type ProjectRequirementRevisionState = ProjectRequirementStatus | "superseded";

export interface ProjectRequirementItem {
  key: string;
  text: string;
}

export interface ProjectRequirementContent {
  problemStatement: string;
  goals: ProjectRequirementItem[];
  inScope: ProjectRequirementItem[];
  outOfScope: ProjectRequirementItem[];
  constraints: ProjectRequirementItem[];
  acceptanceCriteria: ProjectRequirementItem[];
  dependencies: ProjectRequirementItem[];
}

export interface ProjectRequirementBaseline {
  id: string;
  workspaceId: string;
  projectId: string;
  status: ProjectRequirementStatus;
  currentRevision: number;
  approvedRevision: number | null;
  submittedBy: string | null;
  submittedAt: string | null;
  approvedBy: string | null;
  approvedAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectRequirementRevision {
  baselineId: string;
  revision: number;
  content: ProjectRequirementContent;
  changeSummary: string;
  actorId: string;
  createdAt: string;
  state: ProjectRequirementRevisionState;
  submittedBy: string | null;
  submittedAt: string | null;
  approvedBy: string | null;
  approvedAt: string | null;
}

export interface ProjectRequirementBaselineResponse {
  baseline: ProjectRequirementBaseline | null;
  currentContent: ProjectRequirementContent | null;
  effectiveContent: ProjectRequirementContent | null;
  history: ProjectRequirementRevision[];
}

export interface SaveProjectRequirementDraftRequest {
  expectedRevision: number;
  content: ProjectRequirementContent;
  changeSummary: string;
}

export interface ProjectRequirementTransitionRequest {
  expectedRevision: number;
}
