import { describe, expect, it } from "vitest";
import { parseWithFallback } from "../api/schema";
import {
  EMPTY_ACCEPTANCE_CONCLUSION_LIST,
  acceptanceConclusionListSchema,
  acceptanceConclusionSchema,
  projectRetrospectiveActionLinkSchema,
  projectRetrospectiveListSchema,
  projectRetrospectiveSchema,
} from "./schema";

export const VALID_RETROSPECTIVE_WIRE = {
  id: "retro-1",
  workspace_id: "workspace-1",
  project_id: "project-1",
  status: "published",
  current_revision: 3,
  published_revision: 3,
  created_by: "member-1",
  created_at: "2026-08-01T00:00:00Z",
  updated_at: "2026-08-03T00:00:00Z",
  current: {
    revision: 3,
    status: "published",
    action: "publish_revision",
    content: {
      summary: "Delivery completed.",
      successes: ["Small batches"],
      problems: [],
      lessons: ["Review sooner"],
      action_items: [{
        id: "action-1",
        title: "Schedule review",
        description: "Before freeze",
        assignee_id: "member-2",
        due_date: "2026-08-10",
      }],
    },
    participants: [
      { member_id: "member-1", role: "participant" },
      { member_id: "member-2", role: "facilitator" },
    ],
    actor_id: "member-2",
    created_at: "2026-08-03T00:00:00Z",
  },
  history: [
    {
      revision: 1,
      status: "draft",
      action: "create",
      content: {
        summary: "Delivery draft.",
        successes: [],
        problems: [],
        lessons: ["Review"],
        action_items: [],
      },
      participants: [{ member_id: "member-1", role: "participant" }],
      actor_id: "member-1",
      created_at: "2026-08-01T00:00:00Z",
    },
    {
      revision: 2,
      status: "superseded",
      action: "publish",
      content: {
        summary: "Delivery published.",
        successes: ["Small batches"],
        problems: [],
        lessons: ["Review"],
        action_items: [{ id: "action-1", title: "Schedule review" }],
      },
      participants: [{ member_id: "member-1", role: "participant" }],
      actor_id: "member-1",
      created_at: "2026-08-02T00:00:00Z",
    },
    {
      revision: 3,
      status: "published",
      action: "publish_revision",
      content: {
        summary: "Delivery completed.",
        successes: ["Small batches"],
        problems: [],
        lessons: ["Review sooner"],
        action_items: [{
          id: "action-1",
          title: "Schedule review",
          description: "Before freeze",
          assignee_id: "member-2",
          due_date: "2026-08-10",
        }],
      },
      participants: [
        { member_id: "member-1", role: "participant" },
        { member_id: "member-2", role: "facilitator" },
      ],
      actor_id: "member-2",
      created_at: "2026-08-03T00:00:00Z",
    },
  ],
  action_links: [{
    retrospective_id: "retro-1",
    action_item_id: "action-1",
    source_revision: 3,
    state: "linked",
    target_kind: "task",
    target_id: "task-1",
    created_by: "member-2",
    created_at: "2026-08-03T01:00:00Z",
  }],
  access: { can_edit: false, can_publish: true, can_archive: true },
} as const;

describe("implementation knowledge response schemas", () => {
  it("transforms acceptance conclusions at the API boundary", () => {
    const result = parseWithFallback(
      {
        acceptance_conclusions: [{
          id: "acceptance-1",
          workspace_id: "workspace-1",
          issue_id: "issue-1",
          result: "accepted",
          rationale: "Acceptance checks passed.",
          evidence_refs: ["artifact://report/v1"],
          actor_id: "member-1",
          created_at: "2026-08-01T00:00:00Z",
          updated_at: "2026-08-01T00:00:00Z",
        }],
        total: 1,
      },
      acceptanceConclusionListSchema,
      EMPTY_ACCEPTANCE_CONCLUSION_LIST,
      { endpoint: "GET /api/issues/:id/acceptance-conclusions" },
    );

    expect(result.acceptanceConclusions[0]).toMatchObject({
      workspaceId: "workspace-1",
      issueId: "issue-1",
      evidenceRefs: ["artifact://report/v1"],
      actorId: "member-1",
    });
  });

  it("falls back when acceptance conclusions are malformed", () => {
    const result = parseWithFallback(
      { acceptance_conclusions: [{ id: 42 }], total: "one" },
      acceptanceConclusionListSchema,
      EMPTY_ACCEPTANCE_CONCLUSION_LIST,
      { endpoint: "GET /api/issues/:id/acceptance-conclusions" },
    );
    expect(result).toEqual(EMPTY_ACCEPTANCE_CONCLUSION_LIST);
  });

  it("transforms the complete Retrospective graph without losing provenance", () => {
    const parsed = projectRetrospectiveSchema.parse(VALID_RETROSPECTIVE_WIRE);

    expect(parsed).toMatchObject({
      id: "retro-1",
      workspaceId: "workspace-1",
      projectId: "project-1",
      currentRevision: 3,
      publishedRevision: 3,
      current: {
        content: { actionItems: [{ id: "action-1", assigneeId: "member-2" }] },
        participants: [
          { memberId: "member-1", role: "participant" },
          { memberId: "member-2", role: "facilitator" },
        ],
      },
      actionLinks: [{
        retrospectiveId: "retro-1",
        actionItemId: "action-1",
        sourceRevision: 3,
        targetKind: "task",
        targetId: "task-1",
      }],
      access: { canEdit: false, canPublish: true, canArchive: true },
    });
    expect(parsed.history.map((revision) => revision.status)).toEqual([
      "draft",
      "superseded",
      "published",
    ]);
  });

  it("accepts only an exact empty list envelope and preserves its cursor", () => {
    expect(projectRetrospectiveListSchema.parse({ retrospectives: [] })).toEqual({
      retrospectives: [],
    });
    expect(projectRetrospectiveListSchema.parse({
      retrospectives: [VALID_RETROSPECTIVE_WIRE],
      next_cursor: "opaque.cursor",
    })).toMatchObject({ nextCursor: "opaque.cursor" });

    expect(projectRetrospectiveListSchema.safeParse({}).success).toBe(false);
    expect(projectRetrospectiveListSchema.safeParse({ retrospectives: [], total: 0 }).success).toBe(false);
    expect(projectRetrospectiveListSchema.safeParse({ retrospectives: [{ id: null }] }).success).toBe(false);
  });

  it("fails closed on unknown fields and internally inconsistent authority", () => {
    expect(projectRetrospectiveSchema.safeParse({
      ...VALID_RETROSPECTIVE_WIRE,
      private_candidate: "must not be ignored",
    }).success).toBe(false);
    expect(projectRetrospectiveSchema.safeParse({
      ...VALID_RETROSPECTIVE_WIRE,
      current_revision: 4,
    }).success).toBe(false);
    expect(projectRetrospectiveSchema.safeParse({
      ...VALID_RETROSPECTIVE_WIRE,
      action_links: [{
        ...VALID_RETROSPECTIVE_WIRE.action_links[0],
        retrospective_id: "foreign-retro",
      }],
    }).success).toBe(false);
    expect(projectRetrospectiveActionLinkSchema.safeParse({
      ...VALID_RETROSPECTIVE_WIRE.action_links[0],
      target_id: undefined,
    }).success).toBe(false);
  });

  it("rejects malformed singular mutation responses", () => {
    expect(acceptanceConclusionSchema.safeParse({ id: 42, result: "accepted" }).success).toBe(false);
    expect(projectRetrospectiveSchema.safeParse({ id: "retro-1", lessons: "none" }).success).toBe(false);
  });
});
