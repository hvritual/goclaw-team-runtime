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
  ProjectRequirementCoverageIssue,
  ProjectRequirementCoverageItem,
  ProjectRequirementCoverageSnapshot,
  ProjectRequirementGrant,
  ProjectRequirementIssueLink,
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

export const EMPTY_PROJECT_REQUIREMENT_BASELINE: ProjectRequirementBaselineResponse =
  {
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

const coverageIssueSchema = z
  .object({
    id: nonEmptyString,
    identifier: nonEmptyString,
    title: nonEmptyString,
    status: nonEmptyString,
    acceptance_result: z
      .enum(["accepted", "conditional", "rejected"])
      .nullable(),
  })
  .strict()
  .transform(
    (value): ProjectRequirementCoverageIssue => ({
      id: value.id,
      identifier: value.identifier,
      title: value.title,
      status: value.status,
      acceptanceResult: value.acceptance_result,
    })
  );

const coverageItemSchema = z
  .object({
    requirement_key: nonEmptyString,
    section: z.enum([
      "goals",
      "in_scope",
      "constraints",
      "acceptance_criteria",
    ]),
    text: nonEmptyString,
    stage: z.enum(["unlinked", "linked", "implemented", "accepted"]),
    issues: z.array(coverageIssueSchema),
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
      text: value.text,
      stage: value.stage,
      issues: value.issues,
    })
  );

const coverageSnapshotSchema = z
  .object({
    revision: z.number().int().positive(),
    state: baselineStatusSchema,
    total: revisionNumber,
    linked: revisionNumber,
    implemented: revisionNumber,
    accepted: revisionNumber,
    unlinked: revisionNumber,
    items: z.array(coverageItemSchema),
  })
  .strict()
  .superRefine((value, context) => {
    const keys = new Set<string>();
    let linked = 0;
    let implemented = 0;
    let accepted = 0;
    let unlinked = 0;
    for (const [index, item] of value.items.entries()) {
      if (keys.has(item.requirementKey)) {
        context.addIssue({
          code: "custom",
          path: ["items", index, "requirement_key"],
          message: "duplicate Requirement key",
        });
      }
      keys.add(item.requirementKey);
      const allDone =
        item.issues.length > 0 &&
        item.issues.every((issue) => issue.status === "done");
      const allAccepted =
        item.issues.length > 0 &&
        item.issues.every((issue) => issue.acceptanceResult === "accepted");
      const validStage =
        (item.stage === "unlinked" && item.issues.length === 0) ||
        (item.stage === "linked" && item.issues.length > 0 && !allDone) ||
        (item.stage === "implemented" && allDone && !allAccepted) ||
        (item.stage === "accepted" && allDone && allAccepted);
      if (!validStage) {
        context.addIssue({
          code: "custom",
          path: ["items", index, "stage"],
          message: "coverage stage does not match Issue evidence",
        });
      }
      switch (item.stage) {
        case "accepted":
          accepted += 1;
          implemented += 1;
          linked += 1;
          break;
        case "implemented":
          implemented += 1;
          linked += 1;
          break;
        case "linked":
          linked += 1;
          break;
        case "unlinked":
          unlinked += 1;
          break;
      }
    }
    if (
      value.total !== value.items.length ||
      value.linked !== linked ||
      value.implemented !== implemented ||
      value.accepted !== accepted ||
      value.unlinked !== unlinked ||
      value.unlinked !== value.total - value.linked
    ) {
      context.addIssue({
        code: "custom",
        path: ["total"],
        message: "coverage counters do not match item stages",
      });
    }
  })
  .transform(
    (value): ProjectRequirementCoverageSnapshot => ({
      revision: value.revision,
      state: value.state,
      total: value.total,
      linked: value.linked,
      implemented: value.implemented,
      accepted: value.accepted,
      unlinked: value.unlinked,
      items: value.items,
    })
  );

export const projectRequirementCoverageSchema = z
  .object({
    baseline_status: baselineStatusSchema.nullable(),
    current: coverageSnapshotSchema.nullable(),
    effective: coverageSnapshotSchema.nullable(),
  })
  .strict()
  .superRefine((value, context) => {
    if (value.baseline_status === null) {
      if (value.current !== null || value.effective !== null) {
        context.addIssue({
          code: "custom",
          path: ["baseline_status"],
          message: "missing baseline cannot expose coverage snapshots",
        });
      }
      return;
    }
    if (
      value.current === null ||
      value.current.state !== value.baseline_status
    ) {
      context.addIssue({
        code: "custom",
        path: ["current"],
        message: "current coverage must match baseline status",
      });
    }
    if (
      value.current &&
      value.effective &&
      value.effective.revision > value.current.revision
    ) {
      context.addIssue({
        code: "custom",
        path: ["effective", "revision"],
        message: "effective revision cannot exceed current revision",
      });
    }
  })
  .transform(
    (value): ProjectRequirementCoverage => ({
      baselineStatus: value.baseline_status,
      current: value.current,
      effective: value.effective,
    })
  );
