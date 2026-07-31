import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { KnowledgePage } from "./knowledge-page";

const role = { current: "member" as "owner" | "admin" | "member" };
const listMode = {
  current: "empty" as "empty" | "error" | "loading" | "published",
};
const proposeMutate = vi.fn();
const isProjectLead = { current: false };

vi.mock("@multica/core/hooks", () => ({
  useWorkspaceId: () => "workspace-1",
}));

vi.mock("@multica/core/permissions", () => ({
  useCurrentMember: () => ({
    role: role.current,
    userId: "user-1",
    member: null,
    isLoading: false,
  }),
}));

vi.mock("@multica/core/projects/queries", () => ({
  projectLeadershipListOptions: () => ({
    queryKey: ["projects", "workspace-1", "list"],
    queryFn: async () =>
      isProjectLead.current
        ? [{ projectId: "project-1", leadType: "member", leadId: "user-1" }]
        : [],
  }),
}));

vi.mock("@multica/core/knowledge", () => ({
  knowledgeListOptions: () => ({
    queryKey: ["knowledge", "workspace-1", "list"],
    queryFn: async () => {
      if (listMode.current === "error") throw new Error("unavailable");
      if (listMode.current === "loading") {
        return new Promise(() => undefined);
      }
      if (listMode.current === "published") {
        return {
          entries: [
            {
              id: "knowledge-1",
              workspaceId: "workspace-1",
              projectId: null,
              candidateId: "candidate-1",
              kind: "lesson",
              status: "published",
              currentRevision: 1,
              revisions: [
                {
                  number: 1,
                  supersedesRevision: 0,
                  title: "Retain evidence",
                  content: "Retry delivery after the knowledge store recovers.",
                  createdBy: "admin-1",
                  createdAt: "2026-07-31T00:00:00Z",
                  sourceRefs: [],
                },
              ],
              createdAt: "2026-07-31T00:00:00Z",
              updatedAt: "2026-07-31T00:00:00Z",
            },
          ],
          total: 1,
          nextCursor: null,
        };
      }
      return { entries: [], total: 0, nextCursor: null };
    },
  }),
  knowledgeDetailOptions: () => ({
    queryKey: ["knowledge", "workspace-1", "detail", "knowledge-1"],
    queryFn: async () => ({
      id: "knowledge-1",
      workspaceId: "workspace-1",
      projectId: "project-1",
      candidateId: "candidate-1",
      kind: "lesson",
      status: "published",
      currentRevision: 2,
      revisions: [
        {
          number: 1,
          supersedesRevision: 0,
          title: "Retain evidence",
          content: "Retry delivery after the knowledge store recovers.",
          createdBy: "admin-1",
          createdAt: "2026-07-31T00:00:00Z",
          sourceRefs: [],
        },
        {
          number: 2,
          supersedesRevision: 1,
          title: "Retain evidence through recovery",
          content: "Retry delivery and retain the original evidence.",
          createdBy: "lead-1",
          createdAt: "2026-08-01T00:00:00Z",
          sourceRefs: [
            {
              type: "issue",
              id: "issue-1",
              revision: "4",
              uri: "multica://issues/issue-1",
              checksum: "sha256:test",
            },
          ],
        },
      ],
      createdAt: "2026-07-31T00:00:00Z",
      updatedAt: "2026-08-01T00:00:00Z",
    }),
  }),
  knowledgeCandidateListOptions: (_workspaceId: string, enabled: boolean) => ({
    queryKey: ["knowledge", "workspace-1", "candidates"],
    queryFn: async () => ({ candidates: [], total: 0, nextCursor: null }),
    enabled,
  }),
  useProposeKnowledge: () => ({
    mutate: proposeMutate,
    isPending: false,
  }),
  useReviewKnowledge: () => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}));

vi.mock("../i18n", () => {
  const resources = {
    header: {
      title: "Knowledge",
      propose: "Propose knowledge",
    },
    search: { placeholder: "Search published knowledge" },
    tabs: { published: "Published", review_queue: "Review queue" },
    proposal: {
      title: "New knowledge proposal",
      revision_title: "Propose a knowledge revision",
      title_placeholder: "A concise title",
      content_placeholder: "What should the team retain?",
      reason_placeholder: "Why should this become governed knowledge?",
      submit: "Submit proposal",
      cancel: "Cancel",
    },
    detail: {
      title: "Knowledge history and sources",
      open: "View details",
      close: "Close details",
      loading: "Loading knowledge details",
      load_failed: "Knowledge details could not be loaded",
      propose_revision: "Propose revision",
      revision: "Revision {{number}}",
      revision_count: "{{count}} revisions",
      supersedes: "Supersedes revision {{number}}",
      sources: "Sources",
    },
    states: {
      empty: "No published knowledge yet",
      empty_candidates: "No candidates need review",
      loading: "Loading knowledge",
      load_failed: "Knowledge could not be loaded",
    },
    source: { count: "{{count}} sources" },
    review: {
      rationale_placeholder: "Add a review rationale",
      approve: "Approve",
      reject: "Reject",
      quarantine: "Quarantine",
    },
    kinds: {
      goal: "Goal",
      decision: "Decision",
      constraint: "Constraint",
      requirement: "Requirement",
      procedure: "Procedure",
      lesson: "Lesson",
      reference: "Reference",
    },
  };
  return {
    useT: () => ({
      t: (
        selector: (value: typeof resources) => string,
        options?: { count?: number; number?: number },
      ) =>
        selector(resources)
          .replace("{{count}}", String(options?.count ?? 0))
          .replace("{{number}}", String(options?.number ?? 0)),
    }),
  };
});

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <KnowledgePage />
    </QueryClientProvider>,
  );
}

describe("KnowledgePage permissions", () => {
  beforeEach(() => {
    role.current = "member";
    listMode.current = "empty";
    proposeMutate.mockReset();
    isProjectLead.current = false;
  });

  it("keeps review governance hidden from ordinary members", async () => {
    renderPage();

    expect(await screen.findByText("Knowledge")).toBeInTheDocument();
    expect(screen.getByText("Propose knowledge")).toBeInTheDocument();
    expect(screen.queryByText("Review queue")).not.toBeInTheDocument();
  });

  it("shows the review queue to workspace admins", async () => {
    role.current = "admin";
    renderPage();

    expect(await screen.findByText("Review queue")).toBeInTheDocument();
  });

  it("shows the review queue to a project lead without exposing unrelated governance", async () => {
    isProjectLead.current = true;
    renderPage();

    expect(await screen.findByText("Review queue")).toBeInTheDocument();
  });

  it("renders loading, error, empty, and published content states", async () => {
    listMode.current = "loading";
    const loading = renderPage();
    expect(await screen.findByText("Loading knowledge")).toBeInTheDocument();
    loading.unmount();

    listMode.current = "error";
    const failed = renderPage();
    expect(
      await screen.findByText("Knowledge could not be loaded"),
    ).toBeInTheDocument();
    failed.unmount();

    listMode.current = "empty";
    const empty = renderPage();
    expect(
      await screen.findByText("No published knowledge yet"),
    ).toBeInTheDocument();
    empty.unmount();

    listMode.current = "published";
    renderPage();
    expect(await screen.findByText("Retain evidence")).toBeInTheDocument();
    expect(
      screen.getByText("Retry delivery after the knowledge store recovers."),
    ).toBeInTheDocument();
  });

  it("lets an ordinary member submit a proposal without exposing governance", async () => {
    renderPage();
    fireEvent.click(await screen.findByText("Propose knowledge"));
    fireEvent.change(screen.getByPlaceholderText("A concise title"), {
      target: { value: "Recovery lesson" },
    });
    fireEvent.change(
      screen.getByPlaceholderText("What should the team retain?"),
      {
        target: { value: "Keep source evidence until delivery succeeds." },
      },
    );
    fireEvent.change(
      screen.getByPlaceholderText(
        "Why should this become governed knowledge?",
      ),
      {
        target: { value: "Observed during project delivery." },
      },
    );
    fireEvent.click(screen.getByText("Submit proposal"));

    expect(proposeMutate).toHaveBeenCalledWith(
      {
        kind: "lesson",
        title: "Recovery lesson",
        content: "Keep source evidence until delivery succeeds.",
        reason: "Observed during project delivery.",
      },
      expect.any(Object),
    );
    expect(screen.queryByText("Review queue")).not.toBeInTheDocument();
  });

  it("shows immutable history and proposes a revision from the detail view", async () => {
    listMode.current = "published";
    renderPage();

    fireEvent.click(await screen.findByText("View details"));
    expect(
      await screen.findByText("Knowledge history and sources"),
    ).toBeInTheDocument();
    expect(screen.getByText("Revision 2")).toBeInTheDocument();
    expect(screen.getByText(/multica:\/\/issues\/issue-1/)).toBeInTheDocument();

    fireEvent.click(screen.getAllByText("Propose revision")[0]!);
    expect(
      await screen.findByText("Propose a knowledge revision"),
    ).toBeInTheDocument();
    fireEvent.change(
      screen.getByPlaceholderText(
        "Why should this become governed knowledge?",
      ),
      { target: { value: "A restore drill verified the updated steps." } },
    );
    fireEvent.click(screen.getByText("Submit proposal"));

    expect(proposeMutate).toHaveBeenCalledWith(
      expect.objectContaining({
        knowledgeId: "knowledge-1",
        projectId: "project-1",
        title: "Retain evidence through recovery",
      }),
      expect.any(Object),
    );
  });
});
