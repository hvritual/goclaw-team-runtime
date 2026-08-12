/**
 * @vitest-environment jsdom
 */
import "@testing-library/jest-dom/vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { ApiError } from "@multica/core/api";
import { renderWithI18n } from "../test/i18n";

const mocks = vi.hoisted(() => ({
  mutateAsync: vi.fn(),
  push: vi.fn(),
  queryResults: [] as unknown[],
}));

vi.mock("@tanstack/react-query", () => ({
  useQueries: () => mocks.queryResults,
}));

vi.mock("@multica/core/team-control", () => ({
  createTeamControlCommandId: () => "command-ui-intent",
  parseTeamControlProblem: (value: unknown) => value && typeof value === "object" && "detail" in value ? value : null,
  TeamControlRunQueuePayloadSchema: {
    safeParse: (value: { id?: string; workspace_ref?: string; max_attempts?: number }) => ({
      success: Boolean(value.id && value.workspace_ref?.startsWith("worktree://") && (value.max_attempts ?? 0) >= 1),
    }),
  },
  isTeamControlConflict: (error: unknown) => Boolean(error && typeof error === "object" && (error as { status?: number }).status === 409),
  teamControlWorkspaceOptions: () => ({}),
  teamControlMembersOptions: () => ({}),
  teamControlProjectionOptions: () => ({}),
  useTeamControlEvents: () => "connected",
  useTeamControlCommand: () => ({
    mutateAsync: mocks.mutateAsync,
    isPending: false,
    error: null,
    reset: vi.fn(),
  }),
}));

vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({ projectDetail: (id: string) => `/acme/projects/${id}` }),
}));

vi.mock("../navigation", () => ({
  useNavigation: () => ({ push: mocks.push }),
}));

vi.mock("sonner", () => ({ toast: { success: vi.fn() } }));

import { TeamControlView } from "./team-control-page";

const projection = {
  schema_version: 1 as const,
  workspace_id: "ws-1",
  project_id: "project-1",
  head: 3,
  head_hash: "a".repeat(64),
  nodes: {
    "requirement-1": {
      id: "requirement-1",
      kind: "requirement",
      revision: 1,
      state: "clarifying",
      creator_id: "member-1",
      assignee_ids: [],
      executor_ids: [],
      data: { request: "Preserve the audit trail" },
    },
  },
  edges: {}, evidence: {}, checks: {}, acceptances: {},
};

function successfulQueries() {
  mocks.queryResults = [
    { data: { schema_version: 1, workspace: { name: "Acme" } }, isLoading: false, error: null },
    { data: { schema_version: 1, members: [{ id: "member-1", kind: "human", role: "owner" }] }, isLoading: false, error: null },
    { data: projection, isLoading: false, error: null },
  ];
}

describe("TeamControlView", () => {
  beforeEach(() => {
    mocks.mutateAsync.mockReset().mockResolvedValue({ head: 4 });
    mocks.push.mockReset();
    successfulQueries();
  });

  it("renders the governed projection and trusted member surface", async () => {
    const user = userEvent.setup();
    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);

    expect(screen.getByRole("heading", { name: "Team Control" })).toBeInTheDocument();
    expect(screen.getByText("Preserve the audit trail")).toBeInTheDocument();
    expect(screen.getByText("Head 3")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Members" }));
    expect(screen.getByText("member-1")).toBeInTheDocument();
    expect(screen.getByText("owner")).toBeInTheDocument();
  });

  it("submits a typed command with the current authoritative Head", async () => {
    const user = userEvent.setup();
    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);

    await user.click(screen.getByRole("tab", { name: "Requirements" }));
    await user.click(screen.getByRole("button", { name: "Start requirement" }));
    expect(screen.getByRole("dialog", { name: "Start requirement" })).toBeInTheDocument();
    await user.clear(screen.getByLabelText("ID"));
    await user.type(screen.getByLabelText("ID"), "requirement-2");
    await user.type(screen.getByLabelText("Summary"), "Add a safe retry policy");
    await user.click(screen.getByRole("button", { name: "Submit command" }));

    expect(mocks.mutateAsync).toHaveBeenCalledWith({
      type: "requirement.start",
      commandId: expect.any(String),
      expectedHead: 3,
      payload: { id: "requirement-2", text: "Add a safe retry policy" },
    });
  });

  it("reuses the same command id when an uncertain response is retried", async () => {
    const user = userEvent.setup();
    mocks.mutateAsync.mockRejectedValueOnce(new Error("connection reset")).mockResolvedValueOnce({ head: 4 });
    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);
    await user.click(screen.getByRole("tab", { name: "Requirements" }));
    await user.click(screen.getByRole("button", { name: "Start requirement" }));
    await user.clear(screen.getByLabelText("ID"));
    await user.type(screen.getByLabelText("ID"), "requirement-2");
    await user.type(screen.getByLabelText("Summary"), "Preserve intent");
    await user.click(screen.getByRole("button", { name: "Submit command" }));
    await user.click(screen.getByRole("button", { name: "Submit command" }));

    const first = mocks.mutateAsync.mock.calls[0]?.[0] as { commandId: string };
    const second = mocks.mutateAsync.mock.calls[1]?.[0] as { commandId: string };
    expect(first.commandId).toBeTruthy();
    expect(second.commandId).toBe(first.commandId);
  });

  it("renders an explicit denied state without exposing project data", () => {
    mocks.queryResults = [
      { data: undefined, isLoading: false, error: null },
      { data: undefined, isLoading: false, error: null },
      { data: undefined, isLoading: false, error: new ApiError("denied", 403, "Forbidden") },
    ];

    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);

    expect(screen.getByText("Team Control access denied")).toBeInTheDocument();
    expect(screen.queryByText("Preserve the audit trail")).not.toBeInTheDocument();
  });

  it.each([0, 1])("renders workspace/member authorization failures explicitly", (queryIndex) => {
    successfulQueries();
    mocks.queryResults[queryIndex] = {
      data: undefined,
      isLoading: false,
      error: new ApiError("denied", 403, "Forbidden"),
    };

    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);

    expect(screen.getByText("Team Control access denied")).toBeInTheDocument();
    expect(screen.queryByText("Preserve the audit trail")).not.toBeInTheDocument();
  });

  it.each([0, 1])("renders workspace/member service failures explicitly", (queryIndex) => {
    successfulQueries();
    mocks.queryResults[queryIndex] = {
      data: undefined,
      isLoading: false,
      error: new ApiError("unavailable", 503, "Unavailable"),
    };

    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);

    expect(screen.getByText("Team Control unavailable")).toBeInTheDocument();
    expect(screen.queryByText("Preserve the audit trail")).not.toBeInTheDocument();
  });

  it("renders an explicit empty trusted-member state", async () => {
    const user = userEvent.setup();
    successfulQueries();
    mocks.queryResults[1] = {
      data: { schema_version: 1, members: [] },
      isLoading: false,
      error: null,
    };

    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);
    await user.click(screen.getByRole("tab", { name: "Members" }));

    expect(screen.getByText("No trusted members")).toBeInTheDocument();
  });

  it("does not offer a run label that the command contract cannot persist", async () => {
    const user = userEvent.setup();
    renderWithI18n(<TeamControlView workspaceId="ws-1" projectId="project-1" />);
    await user.click(screen.getByRole("tab", { name: "Runner" }));
    await user.click(screen.getByRole("button", { name: "Queue run" }));

    expect(screen.queryByLabelText("Run label")).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Summary")).not.toBeInTheDocument();
  });
});
