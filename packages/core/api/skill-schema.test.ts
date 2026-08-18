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
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(SKILL), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([]), { status: 200 }));
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
    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8000/api/skills/skill-1/files?version_id=version-1",
      expect.any(Object)
    );
  });

  it("strictly reads versioned file manifests and content", async () => {
    const manifest = {
      id: "manifest-1", skill_id: "skill-1", version_id: "version-1",
      path: "SKILL.md", space_object_id: "object-1", media_type: "text/markdown",
      size_bytes: 6, checksum: "a".repeat(64), created_at: "2026-08-18T00:00:00Z",
    };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify([manifest]), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ ...manifest, content: "# Demo" }), { status: 200 }))
      .mockResolvedValueOnce(new Response("# Demo", { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    await expect(client.listSkillFiles("skill-1", "version-1")).resolves.toEqual([manifest]);
    await expect(client.getSkillFile("skill-1", "SKILL.md", "version-1")).resolves.toMatchObject({ content: "# Demo" });
    await expect(client.downloadSkillFile("skill-1", "SKILL.md", "version-1")).resolves.toBeInstanceOf(Blob);
    expect(fetchMock.mock.calls[2]?.[0]).toBe("http://localhost:8000/api/skills/skill-1/files/SKILL.md?version_id=version-1&download=true");
  });

  it("previews and commits an archive as multipart data without forcing JSON content type", async () => {
    const preview = {
      preview_token: "preview-token", name: "Archive skill", description: "Imported",
      warnings: [],
      checksum: "a".repeat(64), total_bytes: 6,
      files: [{ path: "SKILL.md", media_type: "text/markdown", checksum: "b".repeat(64), size_bytes: 6 }],
    };
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify(preview), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(SKILL), { status: 201 }))
      .mockResolvedValueOnce(new Response(JSON.stringify(SKILL), { status: 200 }))
      .mockResolvedValueOnce(new Response(JSON.stringify([]), { status: 200 }));
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");
    const file = new File(["archive"], "skill.zip", { type: "application/zip" });

    await expect(client.previewSkillImportArchive(file)).resolves.toEqual(preview);
    await expect(client.commitSkillImportArchive(file, preview.preview_token, "new_version")).resolves.toMatchObject({ id: "skill-1" });

    const previewInit = fetchMock.mock.calls[0]?.[1] as RequestInit;
    const commitInit = fetchMock.mock.calls[1]?.[1] as RequestInit;
    expect(previewInit.body).toBeInstanceOf(FormData);
    expect(commitInit.body).toBeInstanceOf(FormData);
    expect((previewInit.headers as Record<string, string>)["Content-Type"]).toBeUndefined();
    expect((commitInit.headers as Record<string, string>)["Idempotency-Key"]).toBeTruthy();
    expect((commitInit.body as FormData).get("preview_token")).toBe("preview-token");
    expect((commitInit.body as FormData).get("conflict_mode")).toBe("new_version");
  });

  it("validates provenance and audit history strictly", async () => {
    const valid = {
      skill_id: "skill-1",
      provenance: {
        origin_workspace_id: "workspace-1",
        created_by: "user-1",
        created_at: "2026-08-18T00:00:00Z",
      },
      audit: [
        {
          id: "audit-1",
          version_id: "version-1",
          workspace_id: "workspace-1",
          actor_type: "member",
          actor_id: "user-1",
          action: "skill.created",
          created_at: "2026-08-18T00:00:00Z",
        },
      ],
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(valid), { status: 200 })
      )
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            ...valid,
            audit: [{ ...valid.audit[0], actor_id: 7 }],
          }),
          { status: 200 }
        )
      );
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(client.getSkillHistory("skill-1")).resolves.toEqual(valid);
    await expect(client.getSkillHistory("skill-1")).rejects.toThrow(
      "Invalid skill history response"
    );
  });

  it("rejects unavailable file mutations before issuing a request", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const client = new ApiClient("http://localhost:8000");

    await expect(
      client.updateSkill("skill-1", { content: "# body", expected_revision: 1 })
    ).rejects.toThrow("Skill file updates are unavailable");
    await expect(
      client.updateSkill("skill-1", { files: [], expected_revision: 1 })
    ).rejects.toThrow("Skill file updates are unavailable");
    expect(fetchMock).not.toHaveBeenCalled();
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
