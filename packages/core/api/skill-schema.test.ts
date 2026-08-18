import { afterEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "./client";

const SKILL = {
  id: "skill-1",
  workspace_id: "workspace-1",
  version_id: "version-1",
  version: "1",
  name: "Release helper",
  description: "Ship safely",
  config: {},
  status: "draft" as const,
  revision: 1,
  created_by: "user-1",
  created_at: "2026-08-18T00:00:00Z",
  updated_at: "2026-08-18T00:00:00Z",
  archived: false,
};

describe("Skill API boundary", () => {
  afterEach(() => vi.unstubAllGlobals());

  it("fails closed for malformed lists and mutation responses", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify([{ ...SKILL, revision: "one" }]), {
          status: 200,
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...SKILL, version_id: 7 }), {
          status: 200,
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...SKILL, status: "unknown" }), {
          status: 201,
        })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.listSkills()).resolves.toEqual([]);
    await expect(client.getSkill("skill-1")).rejects.toThrow(
      "Invalid skill response"
    );
    await expect(
      client.createSkill({ name: "Release helper" })
    ).rejects.toThrow("Invalid skill response");
  });

  it("reads an exact historical version through the version query", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(SKILL), {
        status: 200,
      })
    );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(
      client.getSkill("skill-1", "version-1")
    ).resolves.toMatchObject({
      version_id: "version-1",
    });
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8000/api/skills/skill-1?version_id=version-1",
      expect.any(Object)
    );
  });

  it("sends revisioned lifecycle mutations through the exact Skill routes", async () => {
    const versioned = {
      ...SKILL,
      version_id: "version-2",
      version: "2",
      revision: 2,
    };
    const published = {
      ...versioned,
      status: "published" as const,
      revision: 3,
    };
    const deprecated = {
      ...published,
      status: "deprecated" as const,
      revision: 4,
    };
    const restored = { ...deprecated, revision: 6, archived: false };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(versioned), { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(published), { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify(deprecated), { status: 200 })
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify(restored), { status: 200 })
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(
      client.updateSkill("skill-1", { description: "v2", expected_revision: 1 })
    ).resolves.toMatchObject({ version: "2", content: "", files: [] });
    await expect(
      (client as any).publishSkill("skill-1", "version-2", 2)
    ).resolves.toMatchObject({ status: "published", revision: 3 });
    await expect(
      (client as any).deprecateSkill("skill-1", "version-2", 3)
    ).resolves.toMatchObject({ status: "deprecated", revision: 4 });
    await expect(
      (client as any).deleteSkill("skill-1", 4)
    ).resolves.toBeUndefined();
    await expect(
      (client as any).restoreSkill("skill-1", 5)
    ).resolves.toMatchObject({ archived: false, revision: 6 });

    expect(
      fetchMock.mock.calls.map(([url, init]) => [url, init?.method, init?.body])
    ).toEqual([
      [
        "http://localhost:8000/api/skills/skill-1",
        "PUT",
        JSON.stringify({ description: "v2", expected_revision: 1 }),
      ],
      [
        "http://localhost:8000/api/skills/skill-1/versions/version-2/publish",
        "POST",
        JSON.stringify({ expected_revision: 2 }),
      ],
      [
        "http://localhost:8000/api/skills/skill-1/versions/version-2/deprecate",
        "POST",
        JSON.stringify({ expected_revision: 3 }),
      ],
      [
        "http://localhost:8000/api/skills/skill-1",
        "DELETE",
        JSON.stringify({ expected_revision: 4 }),
      ],
      [
        "http://localhost:8000/api/skills/skill-1/restore",
        "POST",
        JSON.stringify({ expected_revision: 5 }),
      ],
    ]);
  });
});
