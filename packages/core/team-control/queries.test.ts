import { afterEach, describe, expect, it, vi } from "vitest";
import { QueryClient } from "@tanstack/react-query";
import { setApiInstance } from "../api";
import type { ApiClient } from "../api/client";
import {
  teamControlKeys,
  teamControlProjectionOptions,
} from "./queries";

afterEach(() => {
  vi.restoreAllMocks();
});

describe("Team Control query model", () => {
  it("scopes every project projection key by workspace and project", () => {
    expect(teamControlKeys.projection("ws-a", "project-a")).toEqual([
      "team-control", "ws-a", "project", "project-a", "projection",
    ]);
    expect(teamControlKeys.projection("ws-a", "project-a"))
      .not.toEqual(teamControlKeys.projection("ws-b", "project-a"));
  });

  it("deduplicates concurrent projection requests through React Query", async () => {
    const requestControlPlane = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      schema_version: 1,
      workspace_id: "ws-1",
      project_id: "project-1",
      head: 0,
      head_hash: "",
      nodes: {}, edges: {}, evidence: {}, checks: {}, acceptances: {},
    }), { status: 200 }));
    setApiInstance({ requestControlPlane } as unknown as ApiClient);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const options = teamControlProjectionOptions("ws-1", "project-1");

    await Promise.all([client.fetchQuery(options), client.fetchQuery(options)]);

    expect(requestControlPlane).toHaveBeenCalledTimes(1);
  });
});
