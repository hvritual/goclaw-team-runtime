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

export interface ProjectRequirementLinkedIssue {
  id: string;
  identifier: string;
  title: string;
  status: string;
  createdBy: string;
  createdAt: string;
}

export interface ProjectRequirementCoverageItem {
  requirementKey: string;
  section: "goals" | "inScope" | "constraints" | "acceptanceCriteria";
  issues: ProjectRequirementLinkedIssue[];
}

export interface ProjectRequirementCoverageSnapshot {
  revision: number;
  total: number;
  linked: number;
  unlinked: number;
  linkedIssueDone: number;
  linkedIssueBlocked: number;
  items: ProjectRequirementCoverageItem[];
}

export interface ProjectRequirementCoverage {
  current: ProjectRequirementCoverageSnapshot | null;
  effective: ProjectRequirementCoverageSnapshot | null;
}

export interface ProjectRequirementLinkRequest { requirementKey: string; issueId: string; revision: number; }
export interface ProjectRequirementCreateIssueRequest { revision: number; }

export interface SaveProjectRequirementDraftRequest {
  expectedRevision: number;
  content: ProjectRequirementContent;
  changeSummary: string;
}

export interface ProjectRequirementTransitionRequest {
  expectedRevision: number;
}
