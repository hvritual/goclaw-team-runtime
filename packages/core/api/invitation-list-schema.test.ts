import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const VALID_INVITATION = {
  id: "invitation-1",
  workspace_id: "workspace-1",
  inviter_id: "owner-user",
  invitee_email: "invitee@example.test",
  invitee_user_id: null,
  role: "member",
  status: "pending",
  created_at: "2026-08-02T00:00:00Z",
  updated_at: "2026-08-02T00:00:00Z",
  expires_at: "2026-08-09T00:00:00Z",
  workspace_name: "Acme",
  inviter_name: "Owner",
  inviter_email: "owner@example.test",
};

describe("workspace invitation list API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("accepts the existing top-level invitation array", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([VALID_INVITATION]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.listWorkspaceInvitations("workspace-1"),
    ).resolves.toEqual([VALID_INVITATION]);
  });

  it("preserves future role and status strings", async () => {
    const futureInvitation = {
      ...VALID_INVITATION,
      role: "viewer",
      status: "scheduled",
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([futureInvitation]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.listWorkspaceInvitations("workspace-1"),
    ).resolves.toEqual([futureInvitation]);
  });

  it("falls back to an empty list for malformed invitation data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([{ ...VALID_INVITATION, id: 42 }]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.listWorkspaceInvitations("workspace-1"),
    ).resolves.toEqual([]);
  });
});
