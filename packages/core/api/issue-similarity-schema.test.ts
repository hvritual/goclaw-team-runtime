import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const ISSUE_CANDIDATE = {
  id: "issue-1",
  workspace_id: "workspace-1",
  number: 41,
  identifier: "WSP-41",
  title: "Alpha beta delivery",
  description: "Delivery details",
  status: "todo",
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
  closed: false,
};

describe("Issue similarity API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("posts the pre-create request and preserves ranked candidates", async () => {
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response(JSON.stringify({
        ranking_version: "lexical-v1",
        candidates: [ISSUE_CANDIDATE],
        truncated: false,
        detector_available: true,
      }), { status: 200 }),
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.checkIssueSimilarity({
      title: "Alpha beta",
      description: "Delivery details",
      project_id: "project-1",
      include_closed: true,
    })).resolves.toEqual({
      ranking_version: "lexical-v1",
      candidates: [ISSUE_CANDIDATE],
      truncated: false,
      detector_available: true,
    });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/issues/similarity/check",
    );
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({
      method: "POST",
      body: JSON.stringify({
        title: "Alpha beta",
        description: "Delivery details",
        project_id: "project-1",
        include_closed: true,
      }),
    });
  });

  it("uses the canonical existing-Issue route and degrades malformed responses", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({
        ranking_version: "lexical-v1",
        candidates: [],
        truncated: false,
        // detector_available is deliberately absent: absence must never mean no duplicates.
      }), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.checkExistingIssueSimilarity("issue-1")).resolves.toEqual({
      ranking_version: "unavailable",
      candidates: [],
      truncated: false,
      detector_available: false,
    });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/issues/issue-1/similarity/check",
    );
    expect(fetchMock.mock.calls[0]?.[1]).toMatchObject({ method: "POST" });
  });
});
