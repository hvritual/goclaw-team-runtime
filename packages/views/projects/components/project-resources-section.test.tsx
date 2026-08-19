import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiError } from "@multica/core/api";
import { renderWithI18n } from "../../test/i18n";
import type { ProjectResource, ListProjectResourcesResponse } from "@multica/core/types";

const state = vi.hoisted(() => ({
  query: {
    data: undefined as ListProjectResourcesResponse | undefined,
    isLoading: false,
    error: null as Error | null,
  },
  create: { mutateAsync: vi.fn(), isPending: false },
  update: { mutateAsync: vi.fn(), isPending: false },
  archive: { mutateAsync: vi.fn(), isPending: false },
  options: vi.fn(() => ({ queryKey: ["project-resources"] })),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => state.query,
}));

vi.mock("@multica/core/projects", () => ({
  projectResourcesOptions: state.options,
  useCreateProjectResource: () => state.create,
  useUpdateProjectResource: () => state.update,
  useDeleteProjectResource: () => state.archive,
}));

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/paths", () => ({
  useCurrentWorkspace: () => ({
    repos: [{ url: "https://github.com/acme/runtime" }],
  }),
}));

vi.mock("@multica/ui/components/ui/popover", () => ({
  Popover: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  PopoverTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  PopoverContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

vi.mock("@multica/ui/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ render }: { render: React.ReactNode }) => <>{render}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <div role="tooltip">{children}</div>,
}));

vi.mock("@multica/ui/components/ui/button", () => ({
  Button: ({ children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props}>{children}</button>
  ),
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

import { ProjectResourcesSection } from "./project-resources-section";

const activeGithub: ProjectResource = {
  id: "resource-github",
  workspace_id: "workspace-1",
  project_id: "project-1",
  resource_type: "github_repo",
  resource_ref: { url: "https://github.com/acme/runtime", ref: "main" },
  label: "Runtime repository with an intentionally very long display label that must remain readable",
  position: 0,
  status: "active",
  revision: 7,
  connection: { state: "available", checked_at: "2026-08-19T00:00:00Z" },
  created_at: "2026-08-19T00:00:00Z",
  created_by: "owner-1",
  updated_at: "2026-08-19T00:00:00Z",
  updated_by: "owner-1",
};

const activeURL: ProjectResource = {
  ...activeGithub,
  id: "resource-url",
  resource_type: "url",
  resource_ref: { url: "https://docs.example.com/runtime" },
  label: "Runtime docs",
  position: 1,
  connection: { state: "degraded", diagnostic_code: "provider_rate_limited" },
};

const archivedUnknown: ProjectResource = {
  ...activeGithub,
  id: "resource-future",
  resource_type: "artifact_registry",
  resource_ref: {
    opaque_id: "artifact-1",
    url: "https://example.com/artifact?access_token=secret",
  },
  label: undefined,
  position: 2,
  status: "archived",
  connection: { state: "unavailable", diagnostic_code: "connection_not_configured" },
  archived_at: "2026-08-19T01:00:00Z",
  archived_by: "owner-1",
};

describe("ProjectResourcesSection", () => {
  beforeEach(() => {
    state.query.data = { resources: [], total: 0, revision: 0 };
    state.query.isLoading = false;
    state.query.error = null;
    state.create.mutateAsync.mockReset().mockResolvedValue(activeGithub);
    state.update.mutateAsync.mockReset().mockResolvedValue(activeGithub);
    state.archive.mutateAsync.mockReset().mockResolvedValue(undefined);
    state.options.mockClear();
  });

  it("renders loading, empty, denied, and generic error states", () => {
    state.query.isLoading = true;
    const loading = renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );
    expect(loading.getByText("Loading resources…")).toBeInTheDocument();
    loading.unmount();

    state.query.isLoading = false;
    const empty = renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );
    expect(empty.getByText("No resources attached.")).toBeInTheDocument();
    empty.unmount();

    state.query.error = new ApiError("denied", 403, "Forbidden");
    const denied = renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );
    expect(denied.getByText("You do not have access to project resources.")).toBeInTheDocument();
    denied.unmount();

    state.query.error = new Error("network secret must not render");
    const failed = renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );
    expect(failed.getByText("Could not load project resources.")).toBeInTheDocument();
    expect(failed.queryByText(/network secret/)).toBeNull();
  });

  it("renders known, archived, connection, unknown-type, and long-text states", () => {
    state.query.data = {
      resources: [activeGithub, activeURL, archivedUnknown],
      total: 3,
      revision: 9,
    };
    renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );

    expect(screen.getByRole("link", { name: activeGithub.label })).toHaveAttribute(
      "href",
      "https://github.com/acme/runtime",
    );
    expect(screen.getByRole("link", { name: "Runtime docs" })).toHaveAttribute(
      "href",
      "https://docs.example.com/runtime",
    );
    expect(screen.getAllByText("artifact_registry")).not.toHaveLength(0);
    expect(screen.queryByText(/access_token/)).toBeNull();
    expect(screen.getByText("Archived")).toBeInTheDocument();
    expect(screen.getByText("Available")).toBeInTheDocument();
    expect(screen.getByText("Degraded")).toBeInTheDocument();
    expect(screen.getByText("Unavailable")).toBeInTheDocument();
    expect(state.options).toHaveBeenCalledWith("workspace-1", "project-1", true);
  });

  it("sends authoritative revision for refresh, reorder, archive, and restore", async () => {
    const user = userEvent.setup();
    state.query.data = {
      resources: [activeGithub, activeURL, archivedUnknown],
      total: 3,
      revision: 9,
    };
    renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );

    await user.click(screen.getByRole("button", { name: `Refresh ${activeGithub.label}` }));
    expect(state.update.mutateAsync).toHaveBeenCalledWith({
      resourceId: "resource-github",
      data: { action: "refresh", expected_revision: 9 },
    });

    await user.click(screen.getByRole("button", { name: "Move Runtime docs up" }));
    expect(state.update.mutateAsync).toHaveBeenCalledWith({
      resourceId: "resource-url",
      data: {
        action: "reorder",
        expected_revision: 9,
        before_resource_id: "resource-github",
      },
    });

    await user.click(screen.getByRole("button", { name: `Archive ${activeGithub.label}` }));
    expect(state.archive.mutateAsync).toHaveBeenCalledWith({
      resourceId: "resource-github",
      expectedRevision: 9,
    });

    await user.click(screen.getByRole("button", { name: "Restore artifact_registry" }));
    expect(state.update.mutateAsync).toHaveBeenCalledWith({
      resourceId: "resource-future",
      data: { action: "restore", expected_revision: 9 },
    });
  });

  it("keeps management controls hidden for readers and creates a generic URL", async () => {
    const user = userEvent.setup();
    state.query.data = { resources: [activeURL], total: 1, revision: 4 };
    const reader = renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage={false} />,
    );
    expect(reader.queryByRole("button", { name: "Add resource" })).toBeNull();
    expect(reader.queryByRole("button", { name: "Archive Runtime docs" })).toBeNull();
    expect(reader.getByRole("link", { name: "Runtime docs" })).toBeInTheDocument();
    reader.unmount();

    state.query.data = { resources: [], total: 0, revision: 0 };
    renderWithI18n(
      <ProjectResourcesSection projectId="project-1" canManage />,
    );
    await user.selectOptions(screen.getByRole("combobox", { name: "Resource type" }), "url");
    await user.type(screen.getByRole("textbox", { name: "Resource URL" }), "https://example.com/guide");
    await user.click(screen.getByRole("button", { name: "Add URL" }));
    expect(state.create.mutateAsync).toHaveBeenCalledWith({
      resource_type: "url",
      resource_ref: { url: "https://example.com/guide" },
    });
  });
});
