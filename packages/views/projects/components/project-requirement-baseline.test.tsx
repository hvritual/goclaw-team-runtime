import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { renderWithI18n } from "../../test/i18n";
import { appendRequirementItem, formatRequirementAuditTime, ProjectRequirementBaseline, removeRequirementItem, updateRequirementItem } from "./project-requirement-baseline";

const mocks = vi.hoisted(() => ({
  save: vi.fn(), submit: vi.fn(), approve: vi.fn(), withdraw: vi.fn(),
  link: vi.fn(), unlink: vi.fn(), createIssue: vi.fn(),
  inReview: false,
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey: string[] }) => ({
    isLoading: false,
    dataUpdatedAt: 1,
    data: options.queryKey[0] === "coverage" ? {
      current: { revision: 2, total: 1, linked: 1, unlinked: 0, linkedIssueDone: 0, linkedIssueBlocked: 1, items: [{ requirementKey: "goal-1", section: "goals", issues: [{ id: "issue-1", identifier: "MUL-1", title: "Track it", status: "blocked", createdBy: "member-1", createdAt: "2026-08-01T00:00:00Z" }, { id: "issue-unknown", identifier: "MUL-3", title: "Unknown", status: "future_status", createdBy: "member-1", createdAt: "2026-08-01T00:00:00Z" }] }] },
      effective: { revision: 1, total: 1, linked: 0, unlinked: 1, linkedIssueDone: 0, linkedIssueBlocked: 0, items: [] },
    } : options.queryKey[0] === "issues" ? { issues: [{ id: "issue-2", identifier: "MUL-2", title: "Existing issue", status: "unknown_status" }] } : {
      baseline: { id: "baseline-1", workspaceId: "workspace-1", projectId: "project-1", status: mocks.inReview ? "in_review" : "draft", currentRevision: 2, approvedRevision: 1, submittedBy: null, submittedAt: null, approvedBy: null, approvedAt: null, createdAt: "2026-08-01T00:00:00Z", updatedAt: "2026-08-01T00:00:00Z" },
      currentContent: { problemStatement: "New draft", goals: [{ key: "goal-1", text: "Track this" }], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] },
      effectiveContent: { problemStatement: "Approved scope", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] },
      history: [
        { baselineId: "baseline-1", revision: 2, content: { problemStatement: "New draft", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] }, changeSummary: "Next draft", actorId: "member-1", createdAt: "2026-08-01T00:00:00Z", state: "draft", submittedBy: null, submittedAt: null, approvedBy: null, approvedAt: null },
        { baselineId: "baseline-1", revision: 1, content: { problemStatement: "Approved scope", goals: [], inScope: [], outOfScope: [], constraints: [], acceptanceCriteria: [], dependencies: [] }, changeSummary: "Approved", actorId: "member-1", createdAt: "2026-08-01T00:00:00Z", state: "approved", submittedBy: "member-1", submittedAt: "2026-08-01T00:00:00Z", approvedBy: "lead-1", approvedAt: "2026-08-01T00:00:00Z" },
      ],
    },
  }),
}));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/workspace/hooks", () => ({ useActorName: () => ({ getActorName: (_type: string, id: string) => id === "lead-1" ? "Lead One" : "Member One" }) }));
vi.mock("@multica/core/project-requirements", () => ({
  projectRequirementBaselineOptions: () => ({ queryKey: ["requirements"] }),
  projectRequirementCoverageOptions: () => ({ queryKey: ["coverage"] }),
  projectRequirementIssuesOptions: () => ({ queryKey: ["issues"] }),
  useSaveProjectRequirementDraft: () => ({ mutate: mocks.save, isPending: false }),
  useSubmitProjectRequirementReview: () => ({ mutate: mocks.submit, isPending: false }),
  useApproveProjectRequirement: () => ({ mutate: mocks.approve, isPending: false }),
  useWithdrawProjectRequirementReview: () => ({ mutate: mocks.withdraw, isPending: false }),
  useLinkProjectRequirementIssue: () => ({ mutate: mocks.link, isPending: false }),
  useUnlinkProjectRequirementIssue: () => ({ mutate: mocks.unlink, isPending: false }),
  useCreateIssueForProjectRequirement: () => ({ mutate: mocks.createIssue, isPending: false }),
}));

describe("ProjectRequirementBaseline", () => {
  it("keeps stable item keys across edit, deletion, and addition", () => {
    const items = [{ key: "first", text: "First" }, { key: "second", text: "Second" }];
    expect(updateRequirementItem(items, "second", "Edited")[1]).toEqual({ key: "second", text: "Edited" });
    expect(removeRequirementItem(items, "first")).toEqual([{ key: "second", text: "Second" }]);
    const appended = appendRequirementItem(items);
    expect(appended[2]?.key).not.toBe("first");
    expect(appended[2]?.key).not.toBe("second");
  });

  it("shows the current draft separately from the effective approved revision", () => {
    mocks.inReview = false;
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" canApprove />);
    expect(screen.getByText("Effective v1")).toBeInTheDocument();
    expect(screen.getByText("Current coverage (v2)")).toBeInTheDocument();
    expect(screen.getByText(/0 done issues · 1 blocked issues/)).toBeInTheDocument();
    expect(screen.getByText("MUL-1 · Track it · blocked")).toBeInTheDocument();
    expect(screen.getByText("MUL-3 · Unknown · future_status")).toBeInTheDocument();
    expect(screen.getByText("Revision 2 · Draft")).toBeInTheDocument();
    expect(screen.getByText("Revision 1 · Approved")).toBeInTheDocument();
    expect(screen.getByText(`Approved by Lead One at ${formatRequirementAuditTime("2026-08-01T00:00:00Z")}`)).toBeInTheDocument();
  });

  it("lets members link, unlink, and create issue only for a saved tracked item", async () => {
    const user = userEvent.setup();
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" canApprove />);
    await user.selectOptions(screen.getByLabelText("Link existing issue"), "issue-2");
    expect(mocks.link).toHaveBeenCalledWith({ requirementKey: "goal-1", issueId: "issue-2", revision: 2 }, expect.anything());
    await user.click(screen.getByRole("button", { name: "Unlink MUL-1" }));
    expect(mocks.unlink).toHaveBeenCalledWith({ requirementKey: "goal-1", issueId: "issue-1", revision: 2 }, expect.anything());
    await user.click(screen.getByRole("button", { name: "Create issue" }));
    expect(mocks.createIssue).toHaveBeenCalledWith({ requirementKey: "goal-1", input: { revision: 2 } }, expect.anything());
    await user.click(screen.getAllByRole("button", { name: "Add item" })[0]!);
    expect(screen.getByText("Save the draft before linking an issue.")).toBeInTheDocument();
  });

  it("hides approval from a member and locks the revision while it is in review", () => {
    mocks.inReview = true;
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" canApprove={false} />);
    expect(screen.queryByRole("button", { name: "Approve" })).not.toBeInTheDocument();
    expect(screen.getByDisplayValue("New draft")).toBeDisabled();
  });

  it("shows approval controls to a project lead or workspace administrator", () => {
    mocks.inReview = true;
    renderWithI18n(<ProjectRequirementBaseline projectId="project-1" canApprove />);
    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
  });
});
