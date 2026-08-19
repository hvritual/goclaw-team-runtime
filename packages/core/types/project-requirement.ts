export type ProjectRequirementStatus =
  | "draft"
  | "in_review"
  | "approved"
  | "frozen"
  | "changed"
  | "retired";

export type ProjectRequirementRevisionState = ProjectRequirementStatus;

export type ProjectRequirementAction =
  | "create"
  | "save_draft"
  | "submit_review"
  | "withdraw_review"
  | "approve"
  | "freeze"
  | "material_change"
  | "retire"
  | "link_issue"
  | "unlink_issue"
  | "link_outline"
  | "unlink_outline"
  | "issue_deleted"
  | "legacy_import";

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
  effectiveRevision: number | null;
  submittedBy: string | null;
  submittedAt: string | null;
  approvedBy: string | null;
  approvedAt: string | null;
  frozenBy: string | null;
  frozenAt: string | null;
  retiredBy: string | null;
  retiredAt: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface ProjectRequirementRevision {
  baselineId: string;
  revision: number;
  content: ProjectRequirementContent;
  state: ProjectRequirementRevisionState;
  action: ProjectRequirementAction;
  changeSummary: string;
  actorId: string;
  submittedBy: string | null;
  submittedAt: string | null;
  approvedBy: string | null;
  approvedAt: string | null;
  frozenBy: string | null;
  frozenAt: string | null;
  createdAt: string;
}

export interface ProjectRequirementIssueLink {
  requirementKey: string;
  issueId: string;
  identifier: string;
  title: string;
  status: string;
  linkedRevision: number;
  reviewRequired: boolean;
  linkedBy: string;
  linkedAt: string;
  unlinkedAt: string | null;
}

export interface ProjectRequirementOutlineLink {
  requirementKey: string;
  nodeId: string;
  nodeTitle: string;
  linkedRevision: number;
  linkedBy: string;
  linkedAt: string;
  unlinkedAt: string | null;
}

export interface ProjectRequirementAccessProjection {
  canEdit: boolean;
  canApprove: boolean;
  canManageAccess: boolean;
  canManageOutline: boolean;
}

export interface ProjectRequirementBaselineResponse {
  baseline: ProjectRequirementBaseline | null;
  currentContent: ProjectRequirementContent | null;
  effectiveContent: ProjectRequirementContent | null;
  history: ProjectRequirementRevision[];
  issueLinks: ProjectRequirementIssueLink[];
  outlineLinks: ProjectRequirementOutlineLink[];
  access: ProjectRequirementAccessProjection;
}

export type ProjectRequirementGrantKind =
  | "project_editor"
  | "requirement_approver";

export interface ProjectRequirementGrant {
  memberId: string;
  userId: string;
  role: "owner" | "admin" | "member";
  grantKind: ProjectRequirementGrantKind;
  grantedBy: string;
  grantedAt: string;
}

export interface ProjectRequirementAccessSet {
  revision: number;
  grants: ProjectRequirementGrant[];
}

export interface ProjectRequirementGrantInput {
  memberId: string;
  grantKind: ProjectRequirementGrantKind;
}

export interface ReplaceProjectRequirementAccessRequest {
  expectedRevision: number;
  grants: ProjectRequirementGrantInput[];
}

export interface ProjectOutlineNode {
  id: string;
  workspaceId: string;
  projectId: string;
  title: string;
  createdBy: string;
  createdAt: string;
}

export interface ProjectOutline {
  revision: number;
  nodes: ProjectOutlineNode[];
}

export interface CreateProjectOutlineNodeRequest {
  expectedRevision: number;
  title: string;
  idempotencyKey?: string;
}

export interface ProjectRequirementIssueLinkRequest {
  requirementKey: string;
  issueId: string;
  expectedRevision: number;
}

export interface ProjectRequirementOutlineLinkRequest {
  requirementKey: string;
  nodeId: string;
  expectedRevision: number;
}

export type ProjectRequirementLinkRequest = ProjectRequirementIssueLinkRequest;

export interface SaveProjectRequirementDraftRequest {
  expectedRevision: number;
  content: ProjectRequirementContent;
  changeSummary: string;
  materialChange: boolean;
  idempotencyKey?: string;
}

export interface ProjectRequirementTransitionRequest {
  expectedRevision: number;
}

// S07C owns coverage semantics. These types remain available for its inactive
// compatibility endpoint, but the S07B view does not treat links as coverage.
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
