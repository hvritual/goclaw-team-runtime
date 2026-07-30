// @vitest-environment jsdom

import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const replace = vi.hoisted(() => vi.fn());
const roleRef = vi.hoisted(() => ({
  current: "admin" as "owner" | "admin" | "member",
}));
const queryRef = vi.hoisted(() => ({
  enabled: true,
  data: {
    roles: [
      { key: "owner" },
      { key: "admin" },
      { key: "member" },
      { key: "auditor" },
    ],
    capabilities: [
      {
        key: "workspace.update",
        domain: "workspace",
        access: {
          owner: "allowed",
          admin: "allowed",
          member: "denied",
        },
      },
      {
        key: "member.manage_owner",
        domain: "member",
        access: {
          owner: "allowed",
          admin: "denied",
          member: "denied",
        },
      },
      {
        key: "project.delete",
        domain: "project",
        access: {
          owner: "allowed",
          admin: "allowed",
          member: "denied",
        },
      },
      {
        key: "issue.update",
        domain: "issue",
        access: {
          owner: "allowed",
          admin: "allowed",
          member: "allowed",
        },
      },
      {
        key: "task.run",
        domain: "task",
        access: {
          owner: "conditional",
          admin: "conditional",
          member: "conditional",
        },
      },
      {
        key: "skill.update",
        domain: "skill",
        access: {
          owner: "allowed",
          admin: "allowed",
          member: "conditional",
        },
      },
      {
        key: "automation.run",
        domain: "automation",
        access: {
          owner: "allowed",
          admin: "conditional",
          member: "scoped",
        },
      },
    ],
  },
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: (options: { enabled?: boolean }) => {
    queryRef.enabled = options.enabled !== false;
    return {
      data: options.enabled === false ? undefined : queryRef.data,
      isLoading: false,
      isError: false,
    };
  },
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({ id: "workspace-1" }),
}));

vi.mock("@multica/core/permissions", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@multica/core/permissions")>();
  return {
    ...actual,
    useCurrentMember: () => ({
      userId: "user-1",
      role: roleRef.current,
      isLoading: false,
    }),
  };
});

vi.mock("@multica/core/workspace/queries", () => ({
  workspacePermissionOptions: (_wsId: string, enabled: boolean) => ({
    queryKey: ["permissions"],
    enabled,
  }),
}));

vi.mock("../../navigation", () => ({
  useNavigation: () => ({
    pathname: "/enterprise/settings",
    searchParams: new URLSearchParams("tab=permissions"),
    replace,
  }),
}));

import { PermissionsTab } from "./permissions-tab";

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

describe("PermissionsTab", () => {
  beforeEach(() => {
    replace.mockClear();
    roleRef.current = "admin";
    queryRef.enabled = true;
  });

  it("renders the server-owned six-domain role matrix for an admin", () => {
    render(<PermissionsTab />, { wrapper: Wrapper });

    expect(screen.getByText("Current role: admin")).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "owner" })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: /admin/ })).toBeInTheDocument();
    expect(screen.getByRole("columnheader", { name: "member" })).toBeInTheDocument();
    expect(screen.getByText("Workspace")).toBeInTheDocument();
    expect(screen.getByText("Members")).toBeInTheDocument();
    expect(screen.getByText("Projects")).toBeInTheDocument();
    expect(screen.getByText("Issues")).toBeInTheDocument();
    expect(screen.getByText("Tasks")).toBeInTheDocument();
    expect(screen.getByText("Skills")).toBeInTheDocument();
    expect(
      screen.queryByRole("columnheader", { name: "auditor" }),
    ).toBeNull();
    expect(screen.queryByText("automation")).toBeNull();
    expect(screen.getByText("Manage owner role")).toBeInTheDocument();
    expect(screen.getAllByText("Conditional").length).toBeGreaterThan(0);
  });

  it("links role assignment back to member management", async () => {
    const user = userEvent.setup();
    render(<PermissionsTab />, { wrapper: Wrapper });

    await user.click(screen.getByRole("button", { name: "Manage member roles" }));

    expect(replace).toHaveBeenCalledWith("/enterprise/settings?tab=members");
  });

  it("does not request or render permission information for a regular member", () => {
    roleRef.current = "member";

    const { container } = render(<PermissionsTab />, { wrapper: Wrapper });

    expect(queryRef.enabled).toBe(false);
    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByText("Permissions")).toBeNull();
  });
});
