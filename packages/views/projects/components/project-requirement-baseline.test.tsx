import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiError } from "@multica/core/api";
import { renderWithI18n } from "../../test/i18n";
import {
  appendRequirementItem,
  formatRequirementAuditTime,
  getProjectRequirementError,
  ProjectRequirementBaseline,
  removeRequirementItem,
  updateRequirementItem,
} from "./project-requirement-baseline";

const mocks = vi.hoisted(() => ({
  save: vi.fn(),
  submit: vi.fn(),
  approve: vi.fn(),
  withdraw: vi.fn(),
  freeze: vi.fn(),
  retire: vi.fn(),
  linkIssue: vi.fn(),
  unlinkIssue: vi.fn(),
  linkOutline: vi.fn(),
  unlinkOutline: vi.fn(),
  createOutline: vi.fn(),
  status: "changed",
  canEdit: true,
  canApprove: true,
  canManageOutline: true,
}));

const EMPTY_CONTENT = {
  problemStatement: "",
  goals: [],
  inScope: [],
  outOfScope: [],
  constraints: [],
  acceptanceCriteria: [],
  dependencies: [],
};

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey: string[] }) => ({
    isLoading: false,
    dataUpdatedAt: 1,
    data:
      options.queryKey[0] === "issues"
        ? {
            issues: [
              {
                id: "issue-2",
                identifier: "MUL-2",
                title: "Existing issue",
                status: "todo",
              },
            ],
          }
        : options.queryKey[0] === "outline"
          ? {
              revision: 2,
              nodes: [
                {
                  id: "node-1",
                  workspaceId: "workspace-1",
                  projectId: "project-1",
                  title: "Delivery",
                  createdBy: "member-1",
                  createdAt: "2026-08-01T00:00:00Z",
                },
                {
                  id: "node-2",
                  workspaceId: "workspace-1",
                  projectId: "project-1",
                  title: "Operations",
                  createdBy: "member-1",
                  createdAt: "2026-08-01T00:01:00Z",
                },
              ],
            }
          : {
              baseline: {
                id: "baseline-1",
                workspaceId: "workspace-1",
                projectId: "project-1",
                status: mocks.status,
                currentRevision: 8,
                approvedRevision: 5,
                effectiveRevision: 6,
                submittedBy: null,
                submittedAt: null,
                approvedBy: "owner-1",
                approvedAt: "2026-08-01T00:00:00Z",
                frozenBy: "owner-1",
                frozenAt: "2026-08-01T01:00:00Z",
                retiredBy: null,
                retiredAt: null,
                createdAt: "2026-08-01T00:00:00Z",
                updatedAt: "2026-08-01T02:00:00Z",
              },
              currentContent: {
                ...EMPTY_CONTENT,
                problemStatement: "New scope",
                goals: [{ key: "goal-1", text: "Track this" }],
              },
              effectiveContent: {
                ...EMPTY_CONTENT,
                problemStatement: "Effective scope",
                goals: [{ key: "goal-1", text: "Track this" }],
              },
              history: [
                {
                  baselineId: "baseline-1",
                  revision: 8,
                  content: { ...EMPTY_CONTENT, problemStatement: "New scope" },
                  state: "changed",
                  action: "material_change",
                  changeSummary: "Expand scope",
                  actorId: "member-1",
                  submittedBy: null,
                  submittedAt: null,
                  approvedBy: "owner-1",
                  approvedAt: "2026-08-01T00:00:00Z",
                  frozenBy: "owner-1",
                  frozenAt: "2026-08-01T01:00:00Z",
                  createdAt: "2026-08-01T02:00:00Z",
                },
              ],
              issueLinks: [
                {
                  requirementKey: "goal-1",
                  issueId: "issue-1",
                  identifier: "MUL-1",
                  title: "Track it",
                  status: "blocked",
                  linkedRevision: 5,
                  reviewRequired: true,
                  linkedBy: "member-1",
                  linkedAt: "2026-08-01T00:30:00Z",
                  unlinkedAt: null,
                },
              ],
              outlineLinks: [
                {
                  requirementKey: "goal-1",
                  nodeId: "node-1",
                  nodeTitle: "Delivery",
                  linkedRevision: 7,
                  linkedBy: "member-1",
                  linkedAt: "2026-08-01T01:30:00Z",
                  unlinkedAt: null,
                },
              ],
              access: {
                canEdit: mocks.canEdit,
                canApprove: mocks.canApprove,
                canManageAccess: false,
                canManageOutline: mocks.canManageOutline,
              },
            },
  }),
}));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/workspace/hooks", () => ({
  useActorName: () => ({
    getActorName: (_type: string, id: string) =>
      id === "owner-1" ? "Owner One" : "Member One",
  }),
}));
vi.mock("@multica/core/project-requirements", () => ({
  projectRequirementBaselineOptions: () => ({ queryKey: ["requirements"] }),
  projectRequirementIssuesOptions: () => ({ queryKey: ["issues"] }),
  projectOutlineOptions: () => ({ queryKey: ["outline"] }),
  useSaveProjectRequirementDraft: () => ({ mutate: mocks.save, isPending: false }),
  useSubmitProjectRequirementReview: () => ({ mutate: mocks.submit, isPending: false }),
  useApproveProjectRequirement: () => ({ mutate: mocks.approve, isPending: false }),
  useWithdrawProjectRequirementReview: () => ({ mutate: mocks.withdraw, isPending: false }),
  useFreezeProjectRequirement: () => ({ mutate: mocks.freeze, isPending: false }),
  useRetireProjectRequirement: () => ({ mutate: mocks.retire, isPending: false }),
  useLinkProjectRequirementIssue: () => ({ mutate: mocks.linkIssue, isPending: false }),
  useUnlinkProjectRequirementIssue: () => ({ mutate: mocks.unlinkIssue, isPending: false }),
  useLinkProjectRequirementOutline: () => ({ mutate: mocks.linkOutline, isPending: false }),
  useUnlinkProjectRequirementOutline: () => ({ mutate: mocks.unlinkOutline, isPending: false }),
  useCreateProjectOutlineNode: () => ({ mutate: mocks.createOutline, isPending: false }),
}));

describe("ProjectRequirementBaseline", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.status = "changed";
    mocks.canEdit = true;
    mocks.canApprove = true;
    mocks.canManageOutline = true;
  });

  it("keeps stable item keys across edit, deletion, and addition", () => {
    const items = [
      { key: "first", text: "First" },
      { key: "second", text: "Second" },
    ];
    expect(updateRequirementItem(items, "second", "Edited")[1]).toEqual({
      key: "second",
      text: "Edited",
    });
    expect(removeRequirementItem(items, "first")).toEqual([
      { key: "second", text: "Second" },
    ]);
    const appended = appendRequirementItem(items);
    expect(appended[2]?.key).not.toBe("first");
    expect(appended[2]?.key).not.toBe("second");
  });

  it("shows current and effective baselines, material impact, links, and immutable history", () => {
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    expect(screen.getByText("Current v8")).toBeInTheDocument();
    expect(screen.getByText("Effective v6")).toBeInTheDocument();
    expect(screen.getByText("Effective scope")).toBeInTheDocument();
    expect(screen.getByText("MUL-1 · Track it · blocked")).toBeInTheDocument();
    expect(screen.getByText("Review required after material change")).toBeInTheDocument();
    expect(screen.getByText("Delivery")).toBeInTheDocument();
    expect(screen.getByText("Revision 8 · Changed · Material change")).toBeInTheDocument();
    expect(
      screen.getByText(
        `Frozen by Owner One at ${formatRequirementAuditTime("2026-08-01T01:00:00Z")}`
      )
    ).toBeInTheDocument();
  });

  it("uses server-projected access for independent approval", () => {
    mocks.status = "in_review";
    mocks.canApprove = false;
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Withdraw" })).toBeInTheDocument();
    expect(screen.getByDisplayValue("New scope")).toBeDisabled();
  });

  it("exposes approve, freeze, and retire only in valid projected states", () => {
    mocks.status = "in_review";
    const { unmount } = renderWithI18n(
      <ProjectRequirementBaseline projectId="project-1" />
    );
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retire" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Freeze" })).not.toBeInTheDocument();
    unmount();

    mocks.status = "approved";
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);
    expect(screen.getByRole("button", { name: "Freeze" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Save draft" })).not.toBeInTheDocument();
  });

  it("requires an explicit material-change intent before saving a frozen baseline", async () => {
    mocks.status = "frozen";
    const user = userEvent.setup();
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    const save = screen.getByRole("button", { name: "Save material change" });
    expect(save).toBeDisabled();
    await user.click(screen.getByRole("checkbox", { name: "Material change" }));
    await user.type(screen.getByLabelText("Change summary"), " impacts delivery");
    expect(save).toBeEnabled();
    await user.click(save);
    expect(mocks.save).toHaveBeenCalledWith(
      expect.objectContaining({
        expectedRevision: 8,
        materialChange: true,
        changeSummary: expect.stringContaining("impacts delivery"),
      }),
      expect.anything()
    );
  });

  it("links and unlinks existing Issues and roots without exposing Issue creation", async () => {
    const user = userEvent.setup();
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    await user.selectOptions(screen.getByLabelText("Link existing issue"), "issue-2");
    expect(mocks.linkIssue).toHaveBeenCalledWith(
      { requirementKey: "goal-1", issueId: "issue-2", expectedRevision: 8 },
      expect.anything()
    );
    await user.click(screen.getByRole("button", { name: "Unlink MUL-1" }));
    expect(mocks.unlinkIssue).toHaveBeenCalledWith(
      { requirementKey: "goal-1", issueId: "issue-1", expectedRevision: 8 },
      expect.anything()
    );
    await user.selectOptions(screen.getByLabelText("Link outline root"), "node-2");
    expect(mocks.linkOutline).toHaveBeenCalledWith(
      { requirementKey: "goal-1", nodeId: "node-2", expectedRevision: 8 },
      expect.anything()
    );
    await user.click(screen.getByRole("button", { name: "Unlink outline Delivery" }));
    expect(mocks.unlinkOutline).toHaveBeenCalledWith(
      { requirementKey: "goal-1", nodeId: "node-1", expectedRevision: 8 },
      expect.anything()
    );
    expect(screen.queryByRole("button", { name: "Create issue" })).not.toBeInTheDocument();
  });

  it("creates only a persistent root node when outline authority is projected", async () => {
    const user = userEvent.setup();
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    await user.type(screen.getByLabelText("New outline root"), "Launch");
    await user.click(screen.getByRole("button", { name: "Create root" }));
    expect(mocks.createOutline).toHaveBeenCalledWith(
      { expectedRevision: 2, title: "Launch" },
      expect.anything()
    );
    expect(screen.queryByText(/parent|reorder|number/i)).not.toBeInTheDocument();
  });

  it("renders retired baselines read-only with no mutation controls", () => {
    mocks.status = "retired";
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" />);

    expect(screen.getByDisplayValue("New scope")).toBeDisabled();
    expect(screen.queryByRole("button", { name: "Save draft" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Submit for review" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Retire" })).not.toBeInTheDocument();
  });

  it("extracts safe stale and independent-approval problem metadata", () => {
    expect(
      getProjectRequirementError(
        new ApiError("conflict", 409, "Conflict", {
          code: "revision_conflict",
          current_revision: 11,
          payload: "never display",
        })
      )
    ).toEqual({ code: "revision_conflict", currentRevision: 11 });
    expect(
      getProjectRequirementError(
        new ApiError("conflict", 409, "Conflict", {
          code: "independent_approval_required",
        })
      )
    ).toEqual({ code: "independent_approval_required" });
  });
});
