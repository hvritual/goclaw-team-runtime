import { beforeEach, describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { configStore } from "@multica/core/config";

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/issues/stores/view-store-context", () => ({
  ViewStoreProvider: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("@multica/core/issues/stores/surface-view-store", () => ({
  getIssueSurfaceViewStore: () => ({}),
}));
vi.mock("@multica/core/issues/surface/scope", () => ({
  issueScopeKey: () => "workspace:all",
}));
vi.mock("../../i18n", () => ({ useT: () => ({ t: () => "translated" }) }));
vi.mock("../components/board-view", () => ({ BoardView: () => null }));
vi.mock("../components/gantt-view", () => ({ GanttView: () => null }));
vi.mock("../components/issues-header", () => ({ IssuesHeader: () => <div /> }));
vi.mock("../components/list-view", () => ({ ListView: () => <div data-testid="issue-list" /> }));
vi.mock("../components/swimlane-view", () => ({ SwimLaneView: () => null }));
vi.mock("../components/table-view", () => ({ TableView: () => null }));
vi.mock("../components/batch-action-toolbar", () => ({
  BatchActionToolbar: () => <div data-testid="batch-action-toolbar" />,
}));
vi.mock("../actions", () => ({
  IssueContextMenuProvider: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("./actions-context", () => ({
  IssueSurfaceActionsProvider: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("./selection-context", () => ({
  IssueSurfaceSelectionProvider: ({ children }: { children: React.ReactNode }) => children,
}));
vi.mock("./use-issue-surface-controller", () => ({
  useIssueSurfaceController: () => ({
    viewMode: "list",
    issues: [{ id: "issue-1" }],
    swimlaneIssues: [],
    assigneeGroups: null,
    isLoading: false,
    isEmpty: false,
    surfaceIssues: [],
    allowGantt: false,
    isRefreshing: false,
    facetCountsExact: true,
    tableFacetCounts: undefined,
    setActiveTableFacet: vi.fn(),
    createEnabled: false,
    visibleStatuses: [],
    hiddenStatuses: [],
    moveIssue: vi.fn(),
    childProgressMap: new Map(),
    projectMap: new Map(),
    loadMoreScope: undefined,
    loadMoreFilter: undefined,
    sort: undefined,
    projectId: undefined,
    statusPagination: undefined,
    groupBranches: undefined,
    filteredGanttIssues: [],
    activeFilters: [],
    actions: {},
    selection: {},
    openCreateIssue: vi.fn(),
  }),
}));

import { IssueSurface } from "./issue-surface";

describe("IssueSurface capability gates", () => {
  beforeEach(() => {
    configStore.getState().setFeatureFlags({ issue_batch: false });
  });

  it("does not mount batch controls while the runtime capability is disabled", () => {
    render(
      <IssueSurface
        scope={{ type: "workspace" }}
        modes={["list"]}
        batchToolbar="always"
      />,
    );

    expect(screen.getByTestId("issue-list")).toBeInTheDocument();
    expect(screen.queryByTestId("batch-action-toolbar")).not.toBeInTheDocument();
  });
});
