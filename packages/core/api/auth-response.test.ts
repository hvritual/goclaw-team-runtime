import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const USER = {
  id: "user-1",
  name: "owner",
  email: "owner@example.com",
  avatar_url: null,
  onboarded_at: null,
  onboarding_questionnaire: {},
  starter_content_state: null,
  language: null,
  profile_description: "",
  timezone: null,
  created_at: "2026-08-13T00:00:00Z",
  updated_at: "2026-08-13T00:00:00Z",
};

describe("auth API response boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("parses the verify-code login envelope", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ token: "session-token", user: USER }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").verifyCode("owner@example.com", "888888"))
      .resolves.toMatchObject({ token: "session-token", user: { id: "user-1" } });
  });

  it("rejects a malformed verify-code login envelope", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(
      JSON.stringify({ user: USER }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )));
    await expect(new ApiClient("http://localhost:8000").verifyCode("owner@example.com", "888888"))
      .rejects.toThrow("Invalid authentication response");
  });
});
