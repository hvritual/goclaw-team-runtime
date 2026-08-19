import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";
import { setSchemaLogger } from "./schema";
import { noopLogger } from "../logger";

const RESOURCE = {
  id: "resource-1",
  workspace_id: "workspace-1",
  project_id: "project-1",
  resource_type: "url",
  resource_ref: { url: "https://example.com/docs" },
  label: "Docs",
  position: 0,
  status: "active",
  revision: 2,
  connection: {
    state: "unchecked",
  },
  created_at: "2026-08-19T00:00:00Z",
  created_by: "owner-1",
  updated_at: "2026-08-19T00:00:00Z",
  updated_by: "owner-1",
};

describe("Project Resource API boundary", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
    setSchemaLogger(noopLogger);
  });

  it("parses exact known and unknown resource projections", async () => {
    const future = {
      ...RESOURCE,
      id: "resource-future",
      resource_type: "artifact_registry",
      resource_ref: { provider: "future", opaque_id: "artifact-1" },
      label: "Future artifact",
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            resources: [RESOURCE, future],
            total: 2,
            revision: 7,
          }),
          { status: 200 }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(RESOURCE), { status: 201 })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...RESOURCE, revision: 8 }), {
          status: 200,
        })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(
      client.listProjectResources("project-1", { includeArchived: true })
    ).resolves.toEqual({
      resources: [RESOURCE, future],
      total: 2,
      revision: 7,
    });
    await expect(
      client.createProjectResource("project-1", {
        resource_type: "url",
        resource_ref: { url: "https://example.com/docs" },
        label: "Docs",
      })
    ).resolves.toEqual(RESOURCE);
    await expect(
      client.updateProjectResource("project-1", "resource-1", {
        action: "refresh",
        expected_revision: 7,
      })
    ).resolves.toMatchObject({ revision: 8 });

    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      "http://localhost:8000/api/projects/project-1/resources?include_archived=true"
    );
    expect(
      new Headers(fetchMock.mock.calls[1]?.[1]?.headers).get("Idempotency-Key")
    ).toBeTruthy();
    expect(fetchMock.mock.calls[2]?.[1]?.body).toBe(
      JSON.stringify({ action: "refresh", expected_revision: 7 })
    );
  });

  it("degrades a malformed list and rejects malformed mutation success bodies", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            resources: [{ ...RESOURCE, revision: "2" }],
            total: 1,
            revision: 2,
          }),
          { status: 200 }
        )
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({ ...RESOURCE, connection: { state: "secret" } }),
          {
            status: 201,
          }
        )
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...RESOURCE, unexpected: true }), {
          status: 200,
        })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listProjectResources("project-1")).resolves.toEqual({
      resources: [],
      total: 0,
      revision: 0,
    });
    await expect(
      client.createProjectResource("project-1", {
        resource_type: "url",
        resource_ref: { url: "https://example.com" },
      })
    ).rejects.toThrow("Invalid Project Resource response");
    await expect(
      client.updateProjectResource("project-1", "resource-1", {
        action: "update",
        expected_revision: 2,
        label: "Changed",
      })
    ).rejects.toThrow("Invalid Project Resource response");
  });

  it("does not expose a secret-bearing Resource URL from a malformed response", async () => {
    const secret = "project-resource-secret-do-not-log";
    const warn = vi.fn();
    setSchemaLogger({ ...noopLogger, warn });
    const fetchMock = vi.fn().mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          resources: [
            {
              ...RESOURCE,
              resource_ref: {
                url: `https://example.com/docs?access_token=${secret}`,
              },
            },
          ],
          total: 1,
          revision: 2,
        }),
        { status: 200 }
      )
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listProjectResources("project-1")).resolves.toEqual({
      resources: [],
      total: 0,
      revision: 0,
    });
    expect(warn).toHaveBeenCalledTimes(1);
    const serializedLog = JSON.stringify(warn.mock.calls);
    expect(serializedLog).not.toContain(secret);
    expect(serializedLog).not.toContain("access_token");
    expect(warn.mock.calls[0]?.[1]).toEqual(
      expect.objectContaining({
        endpoint: "GET /api/projects/:id/resources",
        receivedShape: { kind: "object", fieldCount: 3 },
      })
    );
    expect(warn.mock.calls[0]?.[1]).not.toHaveProperty("received");
  });

  it("sends archive revision and rejects a body masquerading as no-content success", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), { status: 200 })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(
      client.deleteProjectResource("project-1", "resource-1", 8)
    ).resolves.toBeUndefined();
    expect(fetchMock.mock.calls[0]?.[1]?.body).toBe(
      JSON.stringify({ expected_revision: 8 })
    );
    await expect(
      client.deleteProjectResource("project-1", "resource-1", 8)
    ).rejects.toThrow("Invalid Project Resource archive response");
  });
});
