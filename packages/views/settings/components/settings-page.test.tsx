// @vitest-environment jsdom

import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { I18nProvider } from "@multica/core/i18n/react";
import enCommon from "../../locales/en/common.json";
import enSettings from "../../locales/en/settings.json";

const roleRef = vi.hoisted(() => ({
  current: "admin" as "owner" | "admin" | "member",
}));
const searchRef = vi.hoisted(() => ({
  current: new URLSearchParams("tab=workspace"),
}));

vi.mock("@multica/ui/hooks/use-mobile", () => ({
  useIsMobile: () => false,
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({
    id: "workspace-1",
    name: "Enterprise Workspace",
  }),
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

vi.mock("../../navigation", () => ({
  useNavigation: () => ({
    pathname: "/enterprise/settings",
    searchParams: searchRef.current,
    replace: vi.fn(),
  }),
}));

vi.mock("./account-tab", () => ({ AccountTab: () => null }));
vi.mock("./preferences-tab", () => ({ PreferencesTab: () => null }));
vi.mock("./keyboard-shortcuts-tab", () => ({ KeyboardShortcutsTab: () => null }));
vi.mock("./issue-tab", () => ({ IssueTab: () => null }));
vi.mock("./tokens-tab", () => ({ TokensTab: () => null }));
vi.mock("./workspace-tab", () => ({
  WorkspaceTab: () => <div data-testid="WorkspaceTab-content">WorkspaceTab</div>,
}));
vi.mock("./members-tab", () => ({ MembersTab: () => null }));
vi.mock("./labels-tab", () => ({ LabelsTab: () => null }));
vi.mock("./properties-tab", () => ({ PropertiesTab: () => null }));

import { SettingsPage } from "./settings-page";

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

describe("SettingsPage enterprise administration navigation", () => {
  beforeEach(() => {
    roleRef.current = "admin";
    searchRef.current = new URLSearchParams("tab=workspace");
  });

  it("shows Workspace, Members, and Roles as sibling tabs for admins", () => {
    render(<SettingsPage />, { wrapper: Wrapper });

    expect(screen.getByRole("tab", { name: "Workspace" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Members" })).toBeInTheDocument();
    expect(screen.getByRole("tab", { name: "Roles" })).toBeInTheDocument();
  });

  it("hides permission management and safely falls back for regular members", () => {
    roleRef.current = "member";
    searchRef.current = new URLSearchParams("tab=roles");

    render(<SettingsPage />, { wrapper: Wrapper });

    expect(screen.queryByRole("tab", { name: "Roles" })).toBeNull();
    expect(screen.getByTestId("WorkspaceTab-content")).toBeInTheDocument();
  });
});
