// @vitest-environment jsdom

import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const roleRef = vi.hoisted(() => ({
  current: "owner" as "owner" | "admin" | "member",
}));
const invitationQueryRef = vi.hoisted(() => ({
  enabled: true,
  state: "success" as "success" | "loading" | "error" | "empty",
}));

const memberFixtures = [
  {
    id: "member-owner",
    workspace_id: "workspace-1",
    user_id: "user-owner",
    role: "owner" as const,
    created_at: "2026-01-01T00:00:00Z",
    name: "Alice Owner",
    email: "alice@example.com",
    avatar_url: null,
  },
  {
    id: "member-admin",
    workspace_id: "workspace-1",
    user_id: "user-admin",
    role: "admin" as const,
    created_at: "2026-01-02T00:00:00Z",
    name: "Bob Admin",
    email: "bob@example.com",
    avatar_url: null,
  },
  {
    id: "member-regular",
    workspace_id: "workspace-1",
    user_id: "user-member",
    role: "member" as const,
    created_at: "2026-01-03T00:00:00Z",
    name: "Carol Member",
    email: "carol@example.com",
    avatar_url: null,
  },
];

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { queryKey: string[]; enabled?: boolean }) => {
    if (options.queryKey.includes("invitations")) {
      invitationQueryRef.enabled = options.enabled !== false;
      const disabled = options.enabled === false;
      return {
        data:
          disabled ||
          invitationQueryRef.state === "loading" ||
          invitationQueryRef.state === "error"
            ? undefined
            : invitationQueryRef.state === "empty"
              ? []
            : [
                {
                  id: "invitation-1",
                  workspace_id: "workspace-1",
                  inviter_id: "user-owner",
                  invitee_email: "pending@example.com",
                  invitee_user_id: null,
                  role: "member",
                  status: "pending",
                  created_at: "2026-01-04T00:00:00Z",
                  updated_at: "2026-01-04T00:00:00Z",
                  expires_at: "2026-01-11T00:00:00Z",
                },
              ],
        isLoading: !disabled && invitationQueryRef.state === "loading",
        isError: !disabled && invitationQueryRef.state === "error",
      };
    }
    return {
      data: memberFixtures.map((member, index) =>
        index === 0 ? { ...member, role: roleRef.current } : member,
      ),
      isLoading: false,
      isError: false,
    };
  },
  useQueryClient: () => ({ invalidateQueries: vi.fn() }),
}));

vi.mock("@multica/core/workspace/queries", () => ({
  memberListOptions: () => ({ queryKey: ["members"] }),
  invitationListOptions: (_wsId: string, enabled: boolean) => ({
    queryKey: ["invitations"],
    enabled,
  }),
  workspaceKeys: {
    members: () => ["members"],
    invitations: () => ["invitations"],
  },
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector: (state: { user: { id: string } }) => unknown) =>
    selector({ user: { id: "user-owner" } }),
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({
    id: "workspace-1",
    name: "Enterprise Workspace",
  }),
}));

vi.mock("@multica/core/api", () => ({
  api: {
    createMember: vi.fn(),
    updateMember: vi.fn(),
    deleteMember: vi.fn(),
    revokeInvitation: vi.fn(),
  },
}));

vi.mock("../../common/actor-avatar", () => ({
  ActorAvatar: ({ actorId }: { actorId: string }) => (
    <span data-testid={`avatar-${actorId}`} />
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { MembersTab } from "./members-tab";

function Wrapper({ children }: { children: ReactNode }) {
  return (
    <I18nProvider
      locale="en"
      resources={{ en: { common: enCommon, settings: enSettings } }}
    >
      {children}
    </I18nProvider>
  );
}

describe("MembersTab enterprise management", () => {
  beforeEach(() => {
    roleRef.current = "owner";
    invitationQueryRef.enabled = true;
    invitationQueryRef.state = "success";
  });

  afterEach(() => {
    cleanup();
  });

  it("searches members by name or email and filters by role", async () => {
    const user = userEvent.setup();
    render(<MembersTab />, { wrapper: Wrapper });

    const search = screen.getByPlaceholderText("Search by name or email");
    expect(screen.getByRole("combobox", { name: "Filter by role" })).toBeInTheDocument();

    await user.type(search, "bob@example.com");

    expect(screen.getByText("Bob Admin")).toBeInTheDocument();
    expect(screen.queryByText("Alice Owner")).toBeNull();
    expect(screen.queryByText("Carol Member")).toBeNull();

    await user.clear(search);
    await user.click(screen.getByRole("combobox", { name: "Filter by role" }));
    await user.click(await screen.findByRole("option", { name: "Admin" }));

    expect(screen.getByText("Bob Admin")).toBeInTheDocument();
    expect(screen.queryByText("Alice Owner")).toBeNull();
    expect(screen.queryByText("Carol Member")).toBeNull();
  });

  it("does not request or show pending invitations for regular members", () => {
    roleRef.current = "member";
    render(<MembersTab />, { wrapper: Wrapper });

    expect(invitationQueryRef.enabled).toBe(false);
    expect(screen.queryByText("Invite member")).toBeNull();
    expect(screen.queryByText("pending@example.com")).toBeNull();
  });

  it("shows pending invitation loading, error, and empty states", () => {
    invitationQueryRef.state = "loading";
    const loading = render(<MembersTab />, { wrapper: Wrapper });
    expect(
      screen.getByText("Loading pending invitations..."),
    ).toBeInTheDocument();
    loading.unmount();

    invitationQueryRef.state = "error";
    const failed = render(<MembersTab />, { wrapper: Wrapper });
    expect(
      screen.getByText("Unable to load pending invitations."),
    ).toBeInTheDocument();
    failed.unmount();

    invitationQueryRef.state = "empty";
    render(<MembersTab />, { wrapper: Wrapper });
    expect(screen.getByText("No pending invitations.")).toBeInTheDocument();
  });
});
