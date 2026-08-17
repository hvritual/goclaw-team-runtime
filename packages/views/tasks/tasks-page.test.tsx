import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/react";
import type { Task } from "@multica/core/types/task";
import { renderWithI18n } from "../test/i18n";
import { TasksPage } from "./tasks-page";

const mocks = vi.hoisted(() => ({
  query: { data: [] as Task[], isLoading: false, isError: false, error: null as Error | null, hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() },
  create: vi.fn(),
  update: vi.fn(),
  archive: vi.fn(),
  restore: vi.fn(),
  reorder: vi.fn(),
  mutationError: null as Error | null,
}));

vi.mock("@tanstack/react-query", () => ({ useInfiniteQuery: () => mocks.query }));
vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/tasks", () => ({
  taskInfiniteListOptions: () => ({ queryKey: ["tasks"] }),
  useCreateTask: () => ({ mutate: mocks.create, isPending: false, error: mocks.mutationError }),
  useUpdateTask: () => ({ mutate: mocks.update, isPending: false, error: mocks.mutationError }),
  useDeleteTask: () => ({ mutate: mocks.archive, isPending: false, error: mocks.mutationError }),
  useRestoreTask: () => ({ mutate: mocks.restore, isPending: false, error: mocks.mutationError }),
  useReorderTasks: () => ({ mutate: mocks.reorder, isPending: false, error: mocks.mutationError }),
}));

const TASK: Task = {
  id: "task-1", workspace_id: "workspace-1", project_id: null, issue_id: null,
  title: "Ship S02A", description: "", status: "in_progress", priority: "high",
  assignee_type: "agent", assignee_id: "agent-1", creator_type: "member", creator_id: "member-1",
  position: 10, revision: 2, start_date: null, due_date: "2026-08-20T00:00:00Z",
  completed_at: null, archived_at: null, restore_status: "", created_at: "2026-08-18T00:00:00Z", updated_at: "2026-08-18T00:00:00Z",
};

describe("TasksPage", () => {
  beforeEach(() => {
    mocks.query = { data: [], isLoading: false, isError: false, error: null, hasNextPage: false, isFetchingNextPage: false, fetchNextPage: vi.fn() };
    mocks.mutationError = null;
    vi.clearAllMocks();
  });

  it("renders a visible revision conflict state", () => {
    mocks.query = { ...mocks.query, data: [TASK], isLoading: false, isError: false, error: null };
    mocks.mutationError = new Error("409 revision_conflict");
    renderWithI18n(<TasksPage />);
    expect(screen.getByRole("alert")).toHaveTextContent("Task changed elsewhere. Refresh and try again.");
  });

  it("renders distinct empty, denied, and generic error states", () => {
    const empty = renderWithI18n(<TasksPage />);
    expect(screen.getByText("No tasks yet")).toBeInTheDocument();
    empty.unmount();

    mocks.query = { ...mocks.query, data: [], isLoading: false, isError: true, error: new Error("403 permission denied") };
    const denied = renderWithI18n(<TasksPage />);
    expect(screen.getByText("You do not have access to tasks")).toBeInTheDocument();
    denied.unmount();

    mocks.query = { ...mocks.query, data: [], isLoading: false, isError: true, error: new Error("network unavailable") };
    renderWithI18n(<TasksPage />);
    expect(screen.getByText("Tasks could not be loaded")).toBeInTheDocument();
  });

  it("sends the visible revision for lifecycle mutations", () => {
    mocks.query = { ...mocks.query, data: [TASK], isLoading: false, isError: false, error: null };
    renderWithI18n(<TasksPage />);

    fireEvent.change(screen.getByLabelText("In progress"), { target: { value: "done" } });
    expect(mocks.update).toHaveBeenCalledWith({ id: "task-1", status: "done", expected_revision: 2 });
    expect(screen.getByText("Agent agent-1")).toBeInTheDocument();
    expect(screen.getByText("Due Aug 20, 2026")).toBeInTheDocument();
  });

  it("archives a terminal task using its current revision", () => {
    mocks.query = { ...mocks.query, data: [{ ...TASK, status: "done", revision: 3, completed_at: "2026-08-20T00:00:00Z" }], isLoading: false, isError: false, error: null };
    renderWithI18n(<TasksPage />);
    fireEvent.click(screen.getByRole("button", { name: "Archive" }));
    expect(mocks.archive).toHaveBeenCalledWith({ id: "task-1", expectedRevision: 3 });
  });

  it("restores an archived task using its current revision", () => {
    mocks.query = { ...mocks.query, data: [{ ...TASK, status: "archived", revision: 4, restore_status: "done", archived_at: "2026-08-21T00:00:00Z" }], isLoading: false, isError: false, error: null };
    renderWithI18n(<TasksPage />);
    fireEvent.click(screen.getByRole("button", { name: "Restore" }));
    expect(mocks.restore).toHaveBeenCalledWith({ id: "task-1", expectedRevision: 4 });
  });

  it("reorders the visible list as one revision-checked command", () => {
    mocks.query = { ...mocks.query, data: [TASK, { ...TASK, id: "task-2", title: "Follow up", position: 20, revision: 5 }], isLoading: false, isError: false, error: null };
    renderWithI18n(<TasksPage />);
    fireEvent.click(screen.getAllByRole("button", { name: "Move down" })[0]!);
    expect(mocks.reorder).toHaveBeenCalledWith({
      items: [
        { id: "task-2", position: 10, expected_revision: 5 },
        { id: "task-1", position: 20, expected_revision: 2 },
      ],
    });
  });

  it("edits title, due date, and assignment with the visible revision", () => {
    mocks.query = { ...mocks.query, data: [TASK], isLoading: false, isError: false, error: null };
    renderWithI18n(<TasksPage />);
    fireEvent.click(screen.getByText("Edit task"));
    fireEvent.change(screen.getByLabelText("Task title"), { target: { value: "Ship Release 1" } });
    fireEvent.change(screen.getByLabelText("Due date"), { target: { value: "2026-08-25" } });
    fireEvent.change(screen.getByLabelText("Assignee ID"), { target: { value: "agent-2" } });
    fireEvent.submit(screen.getByRole("button", { name: "Save task" }).closest("form")!);
    expect(mocks.update).toHaveBeenCalledWith({
      id: "task-1",
      title: "Ship Release 1",
      due_date: "2026-08-25",
      assignee_type: "agent",
      assignee_id: "agent-2",
      expected_revision: 2,
    });
  });

  it("loads the next opaque-cursor page when more tasks exist", () => {
    mocks.query = { ...mocks.query, data: [TASK], hasNextPage: true };
    renderWithI18n(<TasksPage />);
    fireEvent.click(screen.getByRole("button", { name: "Load more tasks" }));
    expect(mocks.query.fetchNextPage).toHaveBeenCalledTimes(1);
  });
});
