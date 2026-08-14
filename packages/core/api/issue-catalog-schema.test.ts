import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

describe("Issue catalog API boundaries", () => {
  afterEach(() => vi.unstubAllGlobals());

  const timestamp = "2026-08-15T01:02:03Z";
  const label = {
    id: "label-1", workspace_id: "workspace-1", resource_type: "issue",
    name: "Priority", description: "Triage", color: "#112233", usage_count: 1,
    created_at: timestamp, updated_at: timestamp,
  };
  const property = {
    id: "property-1", workspace_id: "workspace-1", name: "Estimate", type: "number",
    description: "", icon: "hash", config: { options: [] }, position: 0,
    archived: false, archived_at: null, usage_count: 1,
    created_at: timestamp, updated_at: timestamp,
  };
  const conclusion = {
    id: "conclusion-1", workspace_id: "workspace-1", issue_id: "issue-1",
    result: "accepted", rationale: "Ready", evidence_refs: ["trace://one"],
    actor_id: "user-1", created_at: timestamp, updated_at: timestamp,
  };

  it("rejects malformed label, property, and acceptance list responses", async () => {
    const responses = [
      { labels: [{ id: 42 }], total: 1 },
      { properties: [{ id: "property-1", type: null }], total: 1 },
      { acceptance_conclusions: [{ id: "conclusion-1", result: "maybe" }], total: 1 },
    ];
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listLabels()).rejects.toThrow("Invalid label list response");
    await expect(client.listProperties()).rejects.toThrow("Invalid property list response");
    await expect(client.listIssueAcceptanceConclusions("issue-1")).rejects.toThrow(
      "Invalid acceptance conclusion list response",
    );
  });

  it("accepts exact catalog lists, bags, and mutation responses", async () => {
    const responses = [
      { labels: [label], total: 1 },
      label,
      { labels: [label] },
      { properties: [property], total: 1 },
      property,
      { properties: { "property-1": 2.5 } },
      { acceptance_conclusions: [conclusion], total: 1 },
      conclusion,
    ];
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listLabels()).resolves.toEqual({ labels: [label], total: 1 });
    await expect(client.createLabel({ name: "Priority", color: "#112233", resource_type: "issue" })).resolves.toEqual(label);
    await expect(client.attachLabel("issue-1", "label-1")).resolves.toEqual({ labels: [label] });
    await expect(client.listProperties()).resolves.toEqual({ properties: [property], total: 1 });
    await expect(client.createProperty({ name: "Estimate", type: "number" })).resolves.toEqual(property);
    await expect(client.setIssueProperty("issue-1", "property-1", 2.5)).resolves.toEqual({ properties: { "property-1": 2.5 } });
    await expect(client.listIssueAcceptanceConclusions("issue-1")).resolves.toEqual({
      acceptanceConclusions: [{
        id: "conclusion-1", workspaceId: "workspace-1", issueId: "issue-1",
        result: "accepted", rationale: "Ready", evidenceRefs: ["trace://one"],
        actorId: "user-1", createdAt: timestamp, updatedAt: timestamp,
      }],
      total: 1,
    });
    await expect(client.createIssueAcceptanceConclusion("issue-1", {
      result: "accepted", rationale: "Ready", evidenceRefs: ["trace://one"],
    })).resolves.toMatchObject({ id: "conclusion-1", result: "accepted" });
  });

  it("accepts the retained empty config shape for non-select properties", async () => {
    const retainedProperty = { ...property, config: {} };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ properties: [retainedProperty], total: 1 }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));

    await expect(new ApiClient("http://localhost:3000").listProperties()).resolves.toEqual({
      properties: [retainedProperty],
      total: 1,
    });
  });

  it("rejects missing envelopes and malformed mutation bags", async () => {
    const responses = [
      {},
      {},
      {},
      { properties: { "property-1": { nested: true } } },
      {},
    ];
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listLabels()).rejects.toThrow("Invalid label list response");
    await expect(client.listLabelsForIssue("issue-1")).rejects.toThrow("Invalid Issue labels response");
    await expect(client.listProperties()).rejects.toThrow("Invalid property list response");
    await expect(client.setIssueProperty("issue-1", "property-1", 2.5)).rejects.toThrow("Invalid Issue properties response");
    await expect(client.listIssueAcceptanceConclusions("issue-1")).rejects.toThrow(
      "Invalid acceptance conclusion list response",
    );
  });

  it("rejects invalid known catalog fields instead of applying compatibility defaults", async () => {
    const responses = [
      { labels: [{ ...label, resource_type: "service" }], total: 1 },
      { id: "label-1", workspace_id: "workspace-1", name: "Missing fields", color: "#112233", created_at: timestamp, updated_at: timestamp },
      { properties: [{ ...property, type: "relation" }], total: 1 },
      { ...property, position: undefined },
      { acceptance_conclusions: [{ ...conclusion, evidence_refs: undefined }], total: 1 },
      { ...conclusion, created_at: "yesterday" },
    ];
    vi.stubGlobal("fetch", vi.fn().mockImplementation(async () => new Response(
      JSON.stringify(responses.shift()),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listLabels()).rejects.toThrow("Invalid label list response");
    await expect(client.createLabel({ name: "Missing fields", color: "#112233" })).rejects.toThrow(
      "Invalid label response",
    );
    await expect(client.listProperties()).rejects.toThrow("Invalid property list response");
    await expect(client.createProperty({ name: "Estimate", type: "number" })).rejects.toThrow(
      "Invalid property response",
    );
    await expect(client.listIssueAcceptanceConclusions("issue-1")).rejects.toThrow(
      "Invalid acceptance conclusion list response",
    );
    await expect(client.createIssueAcceptanceConclusion("issue-1", {
      result: "accepted", rationale: "Ready", evidenceRefs: [],
    })).rejects.toThrow("Invalid acceptance conclusion response");
  });
});
