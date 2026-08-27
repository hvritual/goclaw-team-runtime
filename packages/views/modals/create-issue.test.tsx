import { describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import { renderWithI18n } from "../test/i18n";

const useIssueSimilarity = vi.hoisted(() => vi.fn());

vi.mock("@multica/core/issues/similarity", () => ({ useIssueSimilarity }));

import { CreateIssueSimilarityWarning } from "./create-issue";

describe("CreateIssueSimilarityWarning", () => {
  it("warns before creation without adding a blocking action", () => {
    useIssueSimilarity.mockReturnValue({
      inputPending: false,
      isFetching: false,
      isError: false,
      data: {
        ranking_version: "lexical-v1",
        candidates: [{
          id: "issue-1",
          workspace_id: "workspace-1",
          number: 1,
          identifier: "WSP-1",
          title: "Existing issue",
          description: null,
          status: "todo",
          priority: "none",
          assignee_type: null,
          assignee_id: null,
          creator_type: "member",
          creator_id: "member-1",
          parent_issue_id: null,
          project_id: null,
          position: 0,
          stage: null,
          start_date: null,
          due_date: null,
          metadata: {},
          properties: {},
          created_at: "2026-08-21T00:00:00Z",
          updated_at: "2026-08-21T00:00:00Z",
          score: 100,
          component_scores: { exact_normalized_title: 1 },
          same_project: false,
          closed: false,
        }],
        truncated: false,
        detector_available: true,
      },
    });

    renderWithI18n(
      <CreateIssueSimilarityWarning
        description=""
        enabled
        title="New issue"
      />,
    );

    expect(useIssueSimilarity).toHaveBeenCalledWith({
      title: "New issue",
      description: "",
      projectId: undefined,
      enabled: true,
    });
    expect(screen.getByRole("region", { name: "Possible duplicate issues" })).toBeInTheDocument();
    expect(screen.queryByRole("button")).not.toBeInTheDocument();
  });
});
