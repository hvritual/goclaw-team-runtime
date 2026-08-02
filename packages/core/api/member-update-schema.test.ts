import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const VALID_MEMBER = {
  id: "member-1",
  workspace_id: "workspace-1",
  user_id: "user-1",
  role: "admin",
  created_at: "2026-08-02T00:00:00Z",
  name: "Member",
  email: "member@example.test",
  avatar_url: null,
};

describe("member update API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("accepts the existing member response contract", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(VALID_MEMBER), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.updateMember("workspace-1", "member-1", { role: "admin" }),
    ).resolves.toEqual(VALID_MEMBER);
  });

  it("rejects a malformed member response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...VALID_MEMBER, id: 42 }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.updateMember("workspace-1", "member-1", { role: "admin" }),
    ).rejects.toThrow("Invalid member update response");
  });
});
