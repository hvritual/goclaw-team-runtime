import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const WORKSPACE = {
  id: "workspace-1",
  name: "Acme",
  slug: "acme",
  description: null,
  context: null,
  settings: {},
  repos: [],
  issue_prefix: "ACM",
  avatar_url: null,
  created_at: "2026-08-13T00:00:00Z",
  updated_at: "2026-08-13T00:00:00Z",
};

describe("workspace list API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("accepts the existing top-level Workspace array", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify([WORKSPACE]),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").listWorkspaces())
      .resolves.toEqual([WORKSPACE]);
  });

  it("falls back to an empty list for a malformed Workspace", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify([{ ...WORKSPACE, settings: [] }]),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").listWorkspaces())
      .resolves.toEqual([]);
  });
});
