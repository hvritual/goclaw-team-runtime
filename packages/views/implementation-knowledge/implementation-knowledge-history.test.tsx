// @vitest-environment jsdom
import { beforeEach, describe, expect, it, vi } from "vitest";
import { screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiError } from "@multica/core/api";
import type { ProjectRetrospective } from "@multica/core/types/implementation-knowledge";
import { renderWithI18n } from "../test/i18n";

const state = vi.hoisted(() => ({
  query: {
    data: { retrospectives: [] } as { retrospectives: ProjectRetrospective[] } | undefined,
    isLoading: false,
    error: null as Error | null,
    hasNextPage: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  },
  create: { mutate: vi.fn(), isPending: false },
  update: { mutate: vi.fn(), isPending: false },
  archive: { mutate: vi.fn(), isPending: false },
  target: { mutate: vi.fn(), isPending: false },
  options: vi.fn(() => ({ queryKey: ["retrospectives"] })),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: () => ({ data: undefined }),
  useInfiniteQuery: () => state.query,
}));

vi.mock("@multica/core/implementation-knowledge", () => ({
  acceptanceConclusionListOptions: vi.fn(),
  projectRetrospectiveInfiniteListOptions: state.options,
  useCreateProjectRetrospective: () => state.create,
  useUpdateProjectRetrospective: () => state.update,
  useArchiveProjectRetrospective: () => state.archive,
  useCreateProjectRetrospectiveTarget: () => state.target,
}));

vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/paths", () => ({
  useWorkspacePaths: () => ({
    taskDetail: (id: string) => `/acme/tasks/${id}`,
    issueDetail: (id: string) => `/acme/issues/${id}`,
    knowledge: () => "/acme/knowledge",
  }),
}));
vi.mock("../navigation", () => ({
  AppLink: ({ href, children, ...props }: React.AnchorHTMLAttributes<HTMLAnchorElement>) => (
    <a href={href} {...props}>{children}</a>
  ),
}));
vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }));

import { ProjectRetrospectiveHistory } from "./implementation-knowledge-history";

const revision = {
  revision: 2,
  status: "published" as const,
  action: "publish_revision" as const,
  content: {
    summary: "Release learning",
    successes: ["Small batches"],
    problems: ["Late review"],
    lessons: ["Review sooner"],
    actionItems: [
      { id: "action-open", title: "Follow up" },
      { id: "action-pending", title: "Resume target" },
      { id: "action-linked", title: "Linked issue" },
    ],
  },
  participants: [
    { memberId: "member-1", role: "participant" as const },
    { memberId: "member-2", role: "facilitator" as const },
  ],
  actorId: "member-2",
  createdAt: "2026-08-20T00:00:00Z",
};

const published: ProjectRetrospective = {
  id: "retro-1",
  workspaceId: "workspace-1",
  projectId: "project-1",
  status: "published",
  currentRevision: 2,
  publishedRevision: 2,
  createdBy: "member-1",
  createdAt: "2026-08-19T00:00:00Z",
  updatedAt: "2026-08-20T00:00:00Z",
  current: revision,
  history: [
    { ...revision, revision: 1, status: "superseded", action: "publish", createdAt: "2026-08-19T00:00:00Z" },
    revision,
  ],
  actionLinks: [
    {
      retrospectiveId: "retro-1",
      actionItemId: "action-pending",
      sourceRevision: 2,
      state: "pending",
      targetKind: "task",
      createdBy: "member-2",
      createdAt: "2026-08-20T01:00:00Z",
    },
    {
      retrospectiveId: "retro-1",
      actionItemId: "action-linked",
      sourceRevision: 2,
      state: "linked",
      targetKind: "issue",
      targetId: "issue-1",
      createdBy: "member-2",
      createdAt: "2026-08-20T01:00:00Z",
    },
  ],
  access: { canEdit: false, canPublish: true, canArchive: true },
};

const members = [
  { id: "member-1", workspace_id: "workspace-1", user_id: "user-1", role: "member" as const, created_at: "2026-08-01T00:00:00Z", name: "Alex", email: "alex@example.com", avatar_url: null },
  { id: "member-2", workspace_id: "workspace-1", user_id: "user-2", role: "member" as const, created_at: "2026-08-01T00:00:00Z", name: "Taylor", email: "taylor@example.com", avatar_url: null },
];

describe("ProjectRetrospectiveHistory", () => {
  beforeEach(() => {
    state.query.data = { retrospectives: [] };
    state.query.isLoading = false;
    state.query.error = null;
    state.query.hasNextPage = false;
    state.query.isFetchingNextPage = false;
    state.query.fetchNextPage.mockReset().mockResolvedValue(undefined);
    state.create.mutate.mockReset();
    state.update.mutate.mockReset();
    state.archive.mutate.mockReset();
    state.target.mutate.mockReset();
    state.options.mockClear();
  });

  it("renders loading, true empty, denied, and malformed/error as distinct states", () => {
    state.query.isLoading = true;
    const loading = renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);
    expect(loading.getByText("Loading retrospectives…")).toBeInTheDocument();
    loading.unmount();

    state.query.isLoading = false;
    const empty = renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);
    expect(empty.getByText("No retrospectives yet.")).toBeInTheDocument();
    empty.unmount();

    state.query.error = new ApiError("denied", 403, "Forbidden");
    const denied = renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);
    expect(denied.getByRole("alert")).toHaveTextContent("You do not have access to retrospectives.");
    expect(denied.queryByText("No retrospectives yet.")).toBeNull();
    denied.unmount();

    state.query.error = new Error("malformed private payload must not render");
    const failed = renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);
    expect(failed.getByRole("alert")).toHaveTextContent("Could not load retrospectives.");
    expect(failed.queryByText(/private payload/)).toBeNull();
  });

  it("uses only server access, renders immutable states, and creates default Task or explicit Issue targets", async () => {
    const user = userEvent.setup();
    state.query.data = { retrospectives: [published] };
    renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);

    expect(screen.queryByRole("button", { name: "Edit draft" })).toBeNull();
    expect(screen.getByRole("button", { name: "Publish revision" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Archive" })).toBeInTheDocument();
    expect(screen.getByText("Superseded")).toBeInTheDocument();
    expect(screen.getAllByText("Published").length).toBeGreaterThan(0);
    expect(screen.getByText("Target pending")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Issue issue-1" })).toHaveAttribute("href", "/acme/issues/issue-1");

    await user.click(screen.getByRole("button", { name: "Create Task: Follow up" }));
    expect(state.target.mutate).toHaveBeenCalledWith(
      { retrospectiveId: "retro-1", actionItemId: "action-open" },
      expect.any(Object),
    );
    await user.click(screen.getByRole("button", { name: "Create Issue: Follow up" }));
    expect(state.target.mutate).toHaveBeenCalledWith(
      { retrospectiveId: "retro-1", actionItemId: "action-open", targetKind: "issue" },
      expect.any(Object),
    );
    expect(state.options).toHaveBeenCalledWith("workspace-1", "project-1", true);
  });

  it("shows draft controls only when the server grants them", () => {
    state.query.data = {
      retrospectives: [{
        ...published,
        id: "retro-draft",
        status: "draft",
        publishedRevision: undefined,
        current: { ...revision, status: "draft", action: "save_draft" },
        history: [{ ...revision, status: "draft", action: "save_draft" }],
        actionLinks: [],
        access: { canEdit: true, canPublish: false, canArchive: false },
      }],
    };
    renderWithI18n(<ProjectRetrospectiveHistory projectId="project-1" members={members} />);

    expect(screen.getByRole("button", { name: "Edit draft" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Publish" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Archive" })).toBeNull();
  });
});
