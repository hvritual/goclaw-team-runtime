import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

const state = vi.hoisted(() => ({
  user: { id: "u1" } as { id: string } | null,
  workspace: { id: "ws-1", slug: "acme" } as {
    id: string;
    slug: string;
  } | null,
  workspaces: [{ id: "ws-1", slug: "acme" }],
  validateWorkspaceSlugs: vi.fn(),
}));

vi.mock("@multica/core/auth", () => ({
  useAuthStore: (selector: (value: { user: typeof state.user; isLoading: boolean }) => unknown) =>
    selector({ user: state.user, isLoading: false }),
}));

vi.mock("@multica/core/platform", () => ({
  setCurrentWorkspace: vi.fn(),
}));

vi.mock("@multica/core/workspace", () => ({
  workspaceBySlugOptions: () => ({
    queryKey: ["workspace-by-slug"],
    queryFn: async () => state.workspace,
  }),
  workspaceListOptions: () => ({
    queryKey: ["workspace-list"],
    queryFn: async () => state.workspaces,
  }),
}));

vi.mock("@multica/core/paths", () => ({
  WorkspaceSlugProvider: ({ children }: { children: React.ReactNode }) => children,
}));

vi.mock("@multica/views/workspace/use-workspace-seen", () => ({
  useWorkspaceSeen: () => false,
}));

vi.mock("@/stores/tab-store", () => ({
  useTabStore: {
    getState: () => ({
      validateWorkspaceSlugs: state.validateWorkspaceSlugs,
    }),
  },
}));

import { WorkspaceRouteLayout } from "./workspace-route-layout";

function renderLayout() {
  const qc = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  qc.setQueryData(["workspace-by-slug"], state.workspace);
  qc.setQueryData(["workspace-list"], state.workspaces);

  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={["/acme/issues"]}>
        <Routes>
          <Route path=":workspaceSlug/*" element={<WorkspaceRouteLayout />}>
            <Route path="*" element={<div>Workspace content</div>} />
          </Route>
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  state.user = { id: "u1" };
  state.workspace = { id: "ws-1", slug: "acme" };
  state.workspaces = [{ id: "ws-1", slug: "acme" }];
  state.validateWorkspaceSlugs.mockClear();
});

describe("WorkspaceRouteLayout", () => {
  it("renders workspace routes after the slug resolves", () => {
    renderLayout();
    expect(screen.getByText("Workspace content")).toBeInTheDocument();
  });

  it("does not render workspace content for an unavailable workspace", () => {
    state.workspace = null;
    renderLayout();
    expect(screen.queryByText("Workspace content")).toBeNull();
  });
});
