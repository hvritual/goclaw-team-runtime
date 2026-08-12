import { api } from "../api";
import { parseWithFallback } from "../api/schema";
import {
  TeamControlAppendResultSchema,
  TeamControlMembersResponseSchema,
  TeamControlProjectionSchema,
  TeamControlSessionEventSchema,
  TeamControlWorkspaceResponseSchema,
} from "./schemas";
import type {
  TeamControlAppendResult,
  TeamControlCommandInput,
  TeamControlMembersResponse,
  TeamControlProjection,
  TeamControlSessionEvent,
  TeamControlWorkspaceResponse,
} from "./types";

const emptyWorkspace = (workspaceId: string): TeamControlWorkspaceResponse => ({
  schema_version: 1,
  workspace: {
    id: workspaceId,
    name: "Unavailable workspace",
    state: "unknown",
    version: 1,
    created_at: "unknown",
    updated_at: "unknown",
  },
});

const emptyMembers = (): TeamControlMembersResponse => ({
  schema_version: 1,
  members: [],
});

export const emptyTeamControlProjection = (
  workspaceId: string,
  projectId: string,
): TeamControlProjection => ({
  schema_version: 1,
  workspace_id: workspaceId,
  project_id: projectId,
  head: 0,
  head_hash: "",
  nodes: {},
  edges: {},
  evidence: {},
  checks: {},
  acceptances: {},
});

function encodeSegment(value: string): string {
  return encodeURIComponent(value);
}

function projectPath(workspaceId: string, projectId: string): string {
  return `/v1/workspaces/${encodeSegment(workspaceId)}/projects/${encodeSegment(projectId)}`;
}

async function readJson(response: Response): Promise<unknown> {
  try {
    return await response.json();
  } catch {
    return undefined;
  }
}

export class TeamControlContractError extends Error {
  constructor(endpoint: string) {
    super(`Malformed Team Control response: ${endpoint}`);
    this.name = "TeamControlContractError";
  }
}

export async function getTeamControlWorkspace(
  workspaceId: string,
): Promise<TeamControlWorkspaceResponse> {
  const endpoint = `/v1/workspaces/${encodeSegment(workspaceId)}`;
  const response = await api.requestControlPlane(endpoint, {
    headers: { Accept: "application/json" },
  });
  return parseWithFallback(
    await readJson(response),
    TeamControlWorkspaceResponseSchema,
    emptyWorkspace(workspaceId),
    { endpoint: "team-control.workspace" },
  );
}

export async function listTeamControlMembers(
  workspaceId: string,
): Promise<TeamControlMembersResponse> {
  const endpoint = `/v1/workspaces/${encodeSegment(workspaceId)}/members`;
  const response = await api.requestControlPlane(endpoint, {
    headers: { Accept: "application/json" },
  });
  return parseWithFallback(
    await readJson(response),
    TeamControlMembersResponseSchema,
    emptyMembers(),
    { endpoint: "team-control.members" },
  );
}

export async function getTeamControlProjection(
  workspaceId: string,
  projectId: string,
): Promise<TeamControlProjection> {
  const endpoint = `${projectPath(workspaceId, projectId)}/projection`;
  const response = await api.requestControlPlane(endpoint, {
    headers: { Accept: "application/json" },
  });
  return parseWithFallback(
    await readJson(response),
    TeamControlProjectionSchema,
    emptyTeamControlProjection(workspaceId, projectId),
    { endpoint: "team-control.projection" },
  );
}

export async function executeTeamControlCommand(
  workspaceId: string,
  projectId: string,
  command: TeamControlCommandInput,
): Promise<TeamControlAppendResult> {
  const endpoint = `${projectPath(workspaceId, projectId)}/commands`;
  const response = await api.requestControlPlane(endpoint, {
    method: "POST",
    headers: {
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      type: command.type,
      command_id: command.commandId ?? createTeamControlCommandId(),
      expected_head: command.expectedHead,
      payload: command.payload,
    }),
  });
  const parsed = TeamControlAppendResultSchema.safeParse(await readJson(response));
  if (!parsed.success) throw new TeamControlContractError("team-control.command");
  return parsed.data;
}

export function createTeamControlCommandId(): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return crypto.randomUUID();
  }
  return `cmd-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
}

export interface TeamControlStreamOptions {
  after?: number;
  signal: AbortSignal;
  onOpen?: () => void;
  onEvent: (event: TeamControlSessionEvent) => void;
}

export async function streamTeamControlEvents(
  workspaceId: string,
  projectId: string,
  options: TeamControlStreamOptions,
): Promise<number> {
  const headers: Record<string, string> = { Accept: "text/event-stream" };
  if (options.after && options.after > 0) {
    headers["Last-Event-ID"] = String(options.after);
  }
  const response = await api.requestControlPlane(
    `${projectPath(workspaceId, projectId)}/events`,
    { headers, signal: options.signal },
  );
  if (!response.body) throw new TeamControlContractError("team-control.events");
  options.onOpen?.();

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let cursor = options.after ?? 0;
  try {
    while (true) {
      const { done, value } = await reader.read();
      buffer += decoder.decode(value, { stream: !done });
      const frames = buffer.split("\n\n");
      buffer = frames.pop() ?? "";
      for (const frame of frames) {
        const event = parseSSEFrame(frame);
        if (!event || event.sequence <= cursor) continue;
        cursor = event.sequence;
        options.onEvent(event);
      }
      if (done) break;
    }
  } finally {
    reader.releaseLock();
  }
  return cursor;
}

export function parseSSEFrame(frame: string): TeamControlSessionEvent | null {
  let eventName = "message";
  const data: string[] = [];
  for (const rawLine of frame.split(/\r?\n/)) {
    if (rawLine.startsWith(":")) continue;
    const separator = rawLine.indexOf(":");
    const field = separator === -1 ? rawLine : rawLine.slice(0, separator);
    const value = separator === -1 ? "" : rawLine.slice(separator + 1).replace(/^ /, "");
    if (field === "event") eventName = value;
    if (field === "data") data.push(value);
  }
  if (eventName !== "session" || data.length === 0) return null;
  try {
    const parsed = TeamControlSessionEventSchema.safeParse(JSON.parse(data.join("\n")));
    return parsed.success ? parsed.data : null;
  } catch {
    return null;
  }
}
