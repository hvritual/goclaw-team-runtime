import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { configStore } from "@multica/core/config";
import { KnowledgePage } from "./knowledge-page";

const role = { current: "member" as "owner" | "admin" | "member" };
const listMode = { current: "published" as "published" | "empty" | "error" };
const searchParamsRef = { current: new URLSearchParams() };
const listOptionsRef = {
  current: null as null | {
    filters: Record<string, unknown>;
    enabled: boolean;
  },
};
const reviewMutation = vi.fn();
const proposalMutation = vi.fn(async () => ({}));

vi.mock("../navigation", () => ({
  useNavigation: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    pathname: "/knowledge",
    searchParams: searchParamsRef.current,
  }),
}));
vi.mock("@multica/core/hooks", () => ({ useWorkspaceId: () => "workspace-1" }));
vi.mock("@multica/core/permissions", () => ({
  useCurrentMember: () => ({
    role: role.current,
    userId: "user-1",
    member: null,
    isLoading: false,
  }),
}));

const entry = {
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
      content: "Retry delivery after recovery.",
      createdBy: "admin-1",
      createdAt: "2026-08-18T00:00:00Z",
      sourceRefs: [
        {
          type: "acceptance_conclusion",
          id: "issue-1",
          revision: "sha256:abc",
          citation: "Acceptance passed",
          assetId: null,
          assetVersionId: null,
        },
      ],
    },
  ],
  citation: "Acceptance passed",
  matchedBy: "source",
  createdAt: "2026-08-18T00:00:00Z",
  updatedAt: "2026-08-18T00:00:00Z",
};

vi.mock("@multica/core/knowledge", () => ({
  knowledgeListOptions: (
    _workspaceId: string,
    filters: Record<string, unknown>,
    enabled: boolean
  ) => {
    listOptionsRef.current = { filters, enabled };
    return {
      queryKey: ["knowledge", "workspace-1", "list", filters],
      enabled,
      queryFn: async () => {
        if (listMode.current === "error") throw new Error("unavailable");
        return listMode.current === "empty"
          ? { entries: [], total: 0, nextCursor: null }
          : { entries: [entry], total: 1, nextCursor: null };
      },
    };
  },
  knowledgeDetailOptions: (
    _workspaceId: string,
    id: string,
    enabled: boolean
  ) => ({
    queryKey: ["knowledge", "workspace-1", "detail", id],
    enabled,
    queryFn: async () => ({
      ...entry,
      revisions: [
        ...entry.revisions,
        {
          ...entry.revisions[0],
          number: 2,
          supersedesRevision: 1,
          title: "Retain evidence through recovery",
        },
      ],
      currentRevision: 2,
      matchedBy: "detail",
    }),
  }),
  knowledgeCandidateListOptions: (_workspaceId: string, enabled: boolean) => ({
    queryKey: ["knowledge", "workspace-1", "candidates"],
    enabled,
    queryFn: async () => ({
      candidates: [
        {
          id: "candidate-2",
          workspaceId: "workspace-1",
          projectId: null,
          knowledgeId: null,
          targetRevision: 0,
          kind: "lesson",
          title: "Candidate evidence",
          content: "Candidate body",
          reason: "Reusable",
          status: "candidate",
          revision: 1,
          proposedBy: "user-2",
          sourceRefs: [
            {
              type: "acceptance_conclusion",
              id: "issue-2",
              revision: "sha256:def",
              citation: "Accepted",
            },
          ],
          createdAt: "2026-08-18T00:00:00Z",
          updatedAt: "2026-08-18T00:00:00Z",
        },
      ],
      total: 1,
      nextCursor: null,
    }),
  }),
  useProposeKnowledge: () => ({
    mutateAsync: proposalMutation,
    isPending: false,
    isError: false,
  }),
  useReviewKnowledge: () => ({
    mutate: reviewMutation,
    isPending: false,
    isError: false,
  }),
}));

vi.mock("../i18n", () => {
  const resources = {
    header: { title: "Knowledge", propose: "Propose knowledge" },
    search: { placeholder: "Search published knowledge" },
    detail: {
      title: "Knowledge history and sources",
      open: "View details",
      close: "Close details",
      loading: "Loading knowledge details",
      load_failed: "Knowledge details could not be loaded",
    },
    states: {
      empty: "No published knowledge yet",
      loading: "Loading knowledge",
      load_failed: "Knowledge could not be loaded",
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
      t: (selector: (value: typeof resources) => string) => selector(resources),
    }),
  };
});

describe("KnowledgePage S06B governed review surface", () => {
  beforeEach(() => {
    role.current = "member";
    listMode.current = "published";
    reviewMutation.mockReset();
    proposalMutation.mockClear();
    configStore.setState({
      configLoaded: true,
      featureFlags: { knowledge_query: true, knowledge_review: true },
    });
  });

  it("shows proposal but not the private review queue to a member", async () => {
    renderPage();
    expect(await screen.findByText("Propose knowledge")).toBeInTheDocument();
    expect(screen.queryByText("Review queue")).not.toBeInTheDocument();
    fireEvent.click(screen.getByText("Propose knowledge"));
    expect(screen.getByTestId("knowledge-proposal")).toBeInTheDocument();
  });

  it("shows state-valid review actions to an owner and sends exact revision", async () => {
    role.current = "owner";
    renderPage();
    expect(await screen.findByText("Candidate evidence")).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText("Add a review rationale"), {
      target: { value: "Independent review" },
    });
    fireEvent.click(screen.getByText("approve"));
    expect(reviewMutation).toHaveBeenCalledWith({
      candidateId: "candidate-2",
      action: "approve",
      expectedRevision: 1,
      rationale: "Independent review",
      emergency: false,
    });
  });
});

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <KnowledgePage />
    </QueryClientProvider>
  );
}

describe("KnowledgePage S06A query surface", () => {
  beforeEach(() => {
    role.current = "member";
    listMode.current = "published";
    searchParamsRef.current = new URLSearchParams();
    listOptionsRef.current = null;
    configStore.setState({
      configLoaded: true,
      featureFlags: { knowledge_query: true, knowledge_review: false },
    });
  });

  it("does not mount query or S06B controls before the installed flag", () => {
    configStore.setState({
      configLoaded: true,
      featureFlags: { knowledge_query: false, knowledge_review: false },
    });
    renderPage();
    expect(
      screen.getByText("Knowledge could not be loaded")
    ).toBeInTheDocument();
    expect(listOptionsRef.current?.enabled).toBe(false);
    expect(screen.queryByText("Propose knowledge")).not.toBeInTheDocument();
    expect(screen.queryByText("Review queue")).not.toBeInTheDocument();
  });

  it("renders trust projection and keeps all S06B controls absent", async () => {
    renderPage();
    expect(await screen.findByText("Retain evidence")).toBeInTheDocument();
    expect(screen.getByText("Acceptance passed")).toBeInTheDocument();
    expect(screen.queryByText("Propose knowledge")).not.toBeInTheDocument();
    expect(screen.queryByText("Review queue")).not.toBeInTheDocument();
  });

  it("passes source deep links to the server query and renders immutable detail", async () => {
    searchParamsRef.current = new URLSearchParams(
      "source_type=acceptance_conclusion&source_id=issue-1&source_revision=sha256%3Aabc"
    );
    renderPage();
    await screen.findByText("Retain evidence");
    expect(listOptionsRef.current?.filters).toMatchObject({
      sourceType: "acceptance_conclusion",
      sourceId: "issue-1",
      sourceRevision: "sha256:abc",
    });
    fireEvent.click(screen.getByText("View details"));
    expect(
      await screen.findByText("Retain evidence through recovery")
    ).toBeInTheDocument();
    expect(screen.getAllByText(/Acceptance passed/).length).toBeGreaterThan(0);
  });
});
