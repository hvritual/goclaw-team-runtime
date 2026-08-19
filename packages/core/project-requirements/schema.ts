import { z } from "zod";
import type {
  ProjectOutline,
  ProjectOutlineNode,
  ProjectRequirementAccessProjection,
  ProjectRequirementAccessSet,
  ProjectRequirementBaseline,
  ProjectRequirementBaselineResponse,
  ProjectRequirementContent,
  ProjectRequirementCoverage,
  ProjectRequirementCoverageItem,
  ProjectRequirementCoverageSnapshot,
  ProjectRequirementGrant,
  ProjectRequirementIssueLink,
  ProjectRequirementLinkedIssue,
  ProjectRequirementOutlineLink,
  ProjectRequirementRevision,
} from "../types";

const nonEmptyString = z.string().min(1);
const timestamp = z.iso.datetime({ offset: true });
const revisionNumber = z.number().int().nonnegative();

const itemSchema = z
  .object({ key: nonEmptyString, text: nonEmptyString })
  .strict();

const baselineStatusSchema = z.enum([
  "draft",
  "in_review",
  "approved",
  "frozen",
  "changed",
  "retired",
]);

const revisionActionSchema = z.enum([
  "create",
  "save_draft",
  "submit_review",
  "withdraw_review",
  "approve",
  "freeze",
  "material_change",
  "retire",
  "link_issue",
  "unlink_issue",
  "link_outline",
  "unlink_outline",
  "issue_deleted",
  "legacy_import",
]);

export const projectRequirementContentSchema = z
  .object({
    problem_statement: z.string(),
    goals: z.array(itemSchema),
    in_scope: z.array(itemSchema),
    out_of_scope: z.array(itemSchema),
    constraints: z.array(itemSchema),
    acceptance_criteria: z.array(itemSchema),
    dependencies: z.array(itemSchema),
  })
  .strict()
  .transform(
    (value): ProjectRequirementContent => ({
      problemStatement: value.problem_statement,
      goals: value.goals,
      inScope: value.in_scope,
      outOfScope: value.out_of_scope,
      constraints: value.constraints,
      acceptanceCriteria: value.acceptance_criteria,
      dependencies: value.dependencies,
    })
  );

const baselineSchema = z
  .object({
    id: nonEmptyString,
    workspace_id: nonEmptyString,
    project_id: nonEmptyString,
    status: baselineStatusSchema,
    current_revision: revisionNumber,
    approved_revision: revisionNumber.nullable(),
    effective_revision: revisionNumber.nullable(),
    submitted_by: nonEmptyString.nullable(),
    submitted_at: timestamp.nullable(),
    approved_by: nonEmptyString.nullable(),
    approved_at: timestamp.nullable(),
    frozen_by: nonEmptyString.nullable(),
    frozen_at: timestamp.nullable(),
    retired_by: nonEmptyString.nullable(),
    retired_at: timestamp.nullable(),
    created_at: timestamp,
    updated_at: timestamp,
  })
  .strict()
  .transform(
    (value): ProjectRequirementBaseline => ({
      id: value.id,
      workspaceId: value.workspace_id,
      projectId: value.project_id,
      status: value.status,
      currentRevision: value.current_revision,
      approvedRevision: value.approved_revision,
      effectiveRevision: value.effective_revision,
      submittedBy: value.submitted_by,
      submittedAt: value.submitted_at,
      approvedBy: value.approved_by,
      approvedAt: value.approved_at,
      frozenBy: value.frozen_by,
      frozenAt: value.frozen_at,
      retiredBy: value.retired_by,
      retiredAt: value.retired_at,
      createdAt: value.created_at,
      updatedAt: value.updated_at,
    })
  );

const revisionSchema = z
  .object({
    baseline_id: nonEmptyString,
    revision: revisionNumber,
    content: projectRequirementContentSchema,
    state: baselineStatusSchema,
    action: revisionActionSchema,
    change_summary: z.string(),
    actor_id: nonEmptyString,
    submitted_by: nonEmptyString.nullable(),
    submitted_at: timestamp.nullable(),
    approved_by: nonEmptyString.nullable(),
    approved_at: timestamp.nullable(),
    frozen_by: nonEmptyString.nullable(),
    frozen_at: timestamp.nullable(),
    created_at: timestamp,
  })
  .strict()
  .transform(
    (value): ProjectRequirementRevision => ({
      baselineId: value.baseline_id,
      revision: value.revision,
      content: value.content,
      state: value.state,
      action: value.action,
      changeSummary: value.change_summary,
      actorId: value.actor_id,
      submittedBy: value.submitted_by,
      submittedAt: value.submitted_at,
      approvedBy: value.approved_by,
      approvedAt: value.approved_at,
      frozenBy: value.frozen_by,
      frozenAt: value.frozen_at,
      createdAt: value.created_at,
    })
  );

const issueLinkSchema = z
  .object({
    requirement_key: nonEmptyString,
    issue_id: nonEmptyString,
    identifier: nonEmptyString,
    title: nonEmptyString,
    status: nonEmptyString,
    linked_revision: revisionNumber,
    review_required: z.boolean(),
    linked_by: nonEmptyString,
    linked_at: timestamp,
    unlinked_at: timestamp.nullable(),
  })
  .strict()
  .transform(
    (value): ProjectRequirementIssueLink => ({
      requirementKey: value.requirement_key,
      issueId: value.issue_id,
      identifier: value.identifier,
      title: value.title,
      status: value.status,
      linkedRevision: value.linked_revision,
      reviewRequired: value.review_required,
      linkedBy: value.linked_by,
      linkedAt: value.linked_at,
      unlinkedAt: value.unlinked_at,
    })
  );

const outlineLinkSchema = z
  .object({
    requirement_key: nonEmptyString,
    node_id: nonEmptyString,
    node_title: nonEmptyString,
    linked_revision: revisionNumber,
    linked_by: nonEmptyString,
    linked_at: timestamp,
    unlinked_at: timestamp.nullable(),
  })
  .strict()
  .transform(
    (value): ProjectRequirementOutlineLink => ({
      requirementKey: value.requirement_key,
      nodeId: value.node_id,
      nodeTitle: value.node_title,
      linkedRevision: value.linked_revision,
      linkedBy: value.linked_by,
      linkedAt: value.linked_at,
      unlinkedAt: value.unlinked_at,
    })
  );

const accessProjectionSchema = z
  .object({
    can_edit: z.boolean(),
    can_approve: z.boolean(),
    can_manage_access: z.boolean(),
    can_manage_outline: z.boolean(),
  })
  .strict()
  .transform(
    (value): ProjectRequirementAccessProjection => ({
      canEdit: value.can_edit,
      canApprove: value.can_approve,
      canManageAccess: value.can_manage_access,
      canManageOutline: value.can_manage_outline,
    })
  );

export const projectRequirementBaselineResponseSchema = z
  .object({
    baseline: baselineSchema.nullable(),
    current_content: projectRequirementContentSchema.nullable(),
    effective_content: projectRequirementContentSchema.nullable(),
    history: z.array(revisionSchema),
    issue_links: z.array(issueLinkSchema),
    outline_links: z.array(outlineLinkSchema),
    access: accessProjectionSchema,
  })
  .strict()
  .transform(
    (value): ProjectRequirementBaselineResponse => ({
      baseline: value.baseline,
      currentContent: value.current_content,
      effectiveContent: value.effective_content,
      history: value.history,
      issueLinks: value.issue_links,
      outlineLinks: value.outline_links,
      access: value.access,
    })
  );

export const EMPTY_PROJECT_REQUIREMENT_BASELINE: ProjectRequirementBaselineResponse = {
  baseline: null,
  currentContent: null,
  effectiveContent: null,
  history: [],
  issueLinks: [],
  outlineLinks: [],
  access: {
    canEdit: false,
    canApprove: false,
    canManageAccess: false,
    canManageOutline: false,
  },
};

const grantSchema = z
  .object({
    member_id: nonEmptyString,
    user_id: nonEmptyString,
    role: z.enum(["owner", "admin", "member"]),
    grant_kind: z.enum(["project_editor", "requirement_approver"]),
    granted_by: nonEmptyString,
    granted_at: timestamp,
  })
  .strict()
  .transform(
    (value): ProjectRequirementGrant => ({
      memberId: value.member_id,
      userId: value.user_id,
      role: value.role,
      grantKind: value.grant_kind,
      grantedBy: value.granted_by,
      grantedAt: value.granted_at,
    })
  );

export const projectRequirementAccessSetSchema = z
  .object({ revision: revisionNumber, grants: z.array(grantSchema) })
  .strict()
  .transform(
    (value): ProjectRequirementAccessSet => ({
      revision: value.revision,
      grants: value.grants,
    })
  );

const outlineNodeSchema = z
  .object({
    id: nonEmptyString,
    workspace_id: nonEmptyString,
    project_id: nonEmptyString,
    title: nonEmptyString,
    created_by: nonEmptyString,
    created_at: timestamp,
  })
  .strict()
  .transform(
    (value): ProjectOutlineNode => ({
      id: value.id,
      workspaceId: value.workspace_id,
      projectId: value.project_id,
      title: value.title,
      createdBy: value.created_by,
      createdAt: value.created_at,
    })
  );

export const projectOutlineSchema = z
  .object({ revision: revisionNumber, nodes: z.array(outlineNodeSchema) })
  .strict()
  .transform(
    (value): ProjectOutline => ({
      revision: value.revision,
      nodes: value.nodes,
    })
  );

const linkedIssueSchema = z
  .object({
    id: nonEmptyString,
    identifier: nonEmptyString,
    title: nonEmptyString,
    status: nonEmptyString,
    created_by: nonEmptyString,
    created_at: timestamp,
  })
  .strict()
  .transform(
    (value): ProjectRequirementLinkedIssue => ({
      id: value.id,
      identifier: value.identifier,
      title: value.title,
      status: value.status,
      createdBy: value.created_by,
      createdAt: value.created_at,
    })
  );

const coverageItemSchema = z
  .object({
    requirement_key: nonEmptyString,
    section: z.enum(["goals", "in_scope", "constraints", "acceptance_criteria"]),
    issues: z.array(linkedIssueSchema),
  })
  .strict()
  .transform(
    (value): ProjectRequirementCoverageItem => ({
      requirementKey: value.requirement_key,
      section:
        value.section === "in_scope"
          ? "inScope"
          : value.section === "acceptance_criteria"
            ? "acceptanceCriteria"
            : value.section,
      issues: value.issues,
    })
  );

const coverageSnapshotSchema = z
  .object({
    revision: revisionNumber,
    total: revisionNumber,
    linked: revisionNumber,
    unlinked: revisionNumber,
    linked_issue_done: revisionNumber,
    linked_issue_blocked: revisionNumber,
    items: z.array(coverageItemSchema),
  })
  .strict()
  .transform(
    (value): ProjectRequirementCoverageSnapshot => ({
      revision: value.revision,
      total: value.total,
      linked: value.linked,
      unlinked: value.unlinked,
      linkedIssueDone: value.linked_issue_done,
      linkedIssueBlocked: value.linked_issue_blocked,
      items: value.items,
    })
  );

export const projectRequirementCoverageSchema = z
  .object({
    current: coverageSnapshotSchema.nullable(),
    effective: coverageSnapshotSchema.nullable(),
  })
  .strict()
  .transform(
    (value): ProjectRequirementCoverage => ({
      current: value.current,
      effective: value.effective,
    })
  );

export const EMPTY_PROJECT_REQUIREMENT_COVERAGE: ProjectRequirementCoverage = {
  current: null,
  effective: null,
};
