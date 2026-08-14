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

const COMPLETED_USER = {
  id: "user-1",
  name: "Acme User",
  email: "user@example.com",
  avatar_url: null,
  onboarded_at: "2026-08-14T00:00:00Z",
  onboarding_questionnaire: {},
  starter_content_state: null,
  language: null,
  profile_description: "",
  timezone: null,
  created_at: "2026-08-14T00:00:00Z",
  updated_at: "2026-08-14T00:00:00Z",
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

  it("rejects a malformed created Workspace response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ ...WORKSPACE, settings: [] }),
      { status: 201, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").createWorkspace({
      name: "Acme",
      slug: "acme",
    })).rejects.toThrow("Invalid workspace response");
  });

  it("accepts an exact created Workspace response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify(WORKSPACE),
      { status: 201, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").createWorkspace({
      name: "Acme",
      slug: "acme",
    })).resolves.toEqual(WORKSPACE);
  });

  it("rejects a malformed onboarding completion response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ id: "user-1", onboarded_at: "2026-08-14T00:00:00Z" }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").markOnboardingComplete({
      completion_path: "full",
      workspace_id: "workspace-1",
    })).rejects.toThrow("Invalid onboarding completion response");
  });

  it("accepts an exact onboarding completion response", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify(COMPLETED_USER),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").markOnboardingComplete({
      completion_path: "full",
      workspace_id: "workspace-1",
    })).resolves.toEqual(COMPLETED_USER);
  });
});
