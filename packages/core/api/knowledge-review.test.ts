import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const CANDIDATE = {
  id: "candidate-1",
  workspace_id: "workspace-1",
  project_id: null,
  knowledge_id: null,
  target_revision: 0,
  kind: "lesson",
  title: "Retain",
  content: "Retain this.",
  reason: "Reusable",
  status: "candidate",
  revision: 1,
  proposed_by: "user-1",
  source_refs: [
    {
      type: "acceptance_conclusion",
      id: "issue-1",
      revision: "sha256:abc",
      citation: "Accepted",
      asset_id: null,
      asset_version_id: null,
    },
  ],
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
};

describe("Knowledge review API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("sends reusable idempotency and strict source wire fields", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(JSON.stringify(CANDIDATE), { status: 201 })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    await expect(
      client.proposeKnowledge({
        idempotencyKey: "proposal-key",
        kind: "lesson",
        title: "Retain",
        content: "Retain this.",
        reason: "Reusable",
        sourceRefs: [
          {
            type: "acceptance_conclusion",
            id: "issue-1",
            revision: "sha256:abc",
            citation: "Accepted",
          },
        ],
      })
    ).resolves.toMatchObject({ id: "candidate-1", revision: 1 });
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit;
    expect(init.headers).toMatchObject({ "Idempotency-Key": "proposal-key" });
    expect(JSON.parse(String(init.body))).toMatchObject({
      source_refs: [{ asset_id: null, asset_version_id: null }],
    });
  });

  it("fails closed for malformed proposal, queue, and review success", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...CANDIDATE, revision: 0 }), {
          status: 201,
        })
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            candidates: [{ ...CANDIDATE, extra: true }],
            total: 1,
            next_cursor: null,
          }),
          { status: 200 }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ candidate: CANDIDATE, entry: null, extra: true }),
          { status: 200 }
        )
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    const request = {
      idempotencyKey: "key",
      kind: "lesson" as const,
      title: "Title",
      content: "Body",
      reason: "Reason",
    };
    await expect(client.proposeKnowledge(request)).rejects.toThrow(
      "Invalid Knowledge proposal response"
    );
    await expect(client.listKnowledgeCandidates()).rejects.toThrow(
      "Invalid Knowledge candidate response"
    );
    await expect(
      client.reviewKnowledgeCandidate("candidate-1", {
        action: "publish",
        expectedRevision: 1,
        rationale: "Evidence complete",
      })
    ).rejects.toThrow("Invalid Knowledge review response");
  });

  it("binds candidate pagination and emergency review request fields", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            candidates: [CANDIDATE],
            total: 1,
            next_cursor: null,
          }),
          { status: 200 }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            candidate: { ...CANDIDATE, status: "in_review", revision: 2 },
            entry: null,
          }),
          { status: 200 }
        )
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    await client.listKnowledgeCandidates(25, "cursor-1");
    await client.reviewKnowledgeCandidate("candidate-1", {
      action: "approve",
      expectedRevision: 1,
      rationale: "Emergency reason",
      emergency: true,
    });
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/knowledge/candidates?limit=25&cursor=cursor-1"
    );
    expect(
      JSON.parse(String((fetchMock.mock.calls[1]?.[1] as RequestInit).body))
    ).toEqual({
      action: "approve",
      expected_revision: 1,
      rationale: "Emergency reason",
      emergency: true,
    });
  });
});
