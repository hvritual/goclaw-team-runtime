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

const VALID_MEMBER = {
  id: "member-1",
  workspace_id: "workspace-1",
  user_id: "invitee-user",
  role: "member",
  created_at: "2026-08-02T12:00:00Z",
  name: "Invitee",
  email: "invitee@example.test",
  avatar_url: null,
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

describe("workspace invitation create API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("validates and returns the created invitation", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(VALID_INVITATION), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.createMember("workspace-1", {
        email: "invitee@example.test",
        role: "member",
      }),
    ).resolves.toEqual(VALID_INVITATION);
  });

  it("rejects a malformed created invitation", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...VALID_INVITATION, id: 42 }), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(
      client.createMember("workspace-1", { email: "invitee@example.test" }),
    ).rejects.toThrow("Invalid invitation response");
  });
});

describe("personal invitation API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("validates the top-level personal invitation array", async () => {
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

    await expect(client.listMyInvitations()).resolves.toEqual([
      VALID_INVITATION,
    ]);
  });

  it("falls back to an empty personal list for malformed data", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify([{ ...VALID_INVITATION, status: 42 }]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(client.listMyInvitations()).resolves.toEqual([]);
  });

  it("validates and returns personal invitation detail", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...VALID_INVITATION, status: "declined" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(client.getInvitation("invitation-1")).resolves.toEqual({
      ...VALID_INVITATION,
      status: "declined",
    });
  });

  it("rejects malformed personal invitation detail", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...VALID_INVITATION, id: 42 }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    const client = new ApiClient("http://localhost:3000");

    await expect(client.getInvitation("invitation-1")).rejects.toThrow(
      "Invalid invitation response",
    );
  });

  it("validates and returns the accepted membership", async () => {
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

    await expect(client.acceptInvitation("invitation-1")).resolves.toEqual(
      VALID_MEMBER,
    );
  });

  it("rejects a malformed accepted membership", async () => {
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

    await expect(client.acceptInvitation("invitation-1")).rejects.toThrow(
      "Invalid member response",
    );
  });
});
