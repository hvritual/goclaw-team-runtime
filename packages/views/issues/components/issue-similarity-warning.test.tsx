import { describe, expect, it } from "vitest";
import { screen } from "@testing-library/react";
import type { IssueSimilarityCandidate } from "@multica/core/types";
import { renderWithI18n } from "../../test/i18n";
import { IssueSimilarityWarning } from "./issue-similarity-warning";

const CANDIDATE: IssueSimilarityCandidate = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 41,
  identifier: "WSP-41",
  title: "Alpha beta delivery",
  description: null,
  status: "done",
  priority: "none",
  assignee_type: null,
  assignee_id: null,
  creator_type: "member",
  creator_id: "member-1",
  parent_issue_id: null,
  project_id: "project-1",
  position: 0,
  stage: null,
  start_date: null,
  due_date: null,
  metadata: {},
  properties: {},
  created_at: "2026-08-21T00:00:00Z",
  updated_at: "2026-08-21T00:00:00Z",
  score: 110,
  component_scores: { exact_normalized_title: 1, same_project: 1 },
  same_project: true,
  closed: true,
};

describe("IssueSimilarityWarning", () => {
  it("shows candidates as an informational warning with closed and project labels", () => {
    renderWithI18n(
      <IssueSimilarityWarning
        result={{
          ranking_version: "lexical-v1",
          candidates: [CANDIDATE],
          truncated: true,
          detector_available: true,
        }}
      />,
    );

    expect(screen.getByRole("region", { name: "Possible duplicate issues" })).toBeInTheDocument();
    expect(screen.getByText("WSP-41")).toBeInTheDocument();
    expect(screen.getByText("Alpha beta delivery")).toBeInTheDocument();
    expect(screen.getByText("Same project")).toBeInTheDocument();
    expect(screen.getByText("Closed")).toBeInTheDocument();
    expect(screen.getByText("Showing the best available matches.")).toBeInTheDocument();
  });

  it("never represents an unavailable detector as an empty result", () => {
    renderWithI18n(
      <IssueSimilarityWarning
        result={{
          ranking_version: "unavailable",
          candidates: [],
          truncated: false,
          detector_available: false,
        }}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent(
      "Similarity checking is unavailable. You can still create or edit this issue.",
    );
    expect(screen.queryByText("No similar issues found")).not.toBeInTheDocument();
  });
});
