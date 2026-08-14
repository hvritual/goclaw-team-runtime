import type {
  Issue,
  IssueMetadata,
  IssueMetadataResponse,
  IssueMetadataValue,
  CreateIssueRequest,
  MoveIssueRequest,
  UpdateIssueRequest,
  GroupedIssuesResponse,
  ListIssuesResponse,
  SearchIssuesResponse,
  SearchProjectsResponse,
  UpdateMeRequest,
  CreateMemberRequest,
  UpdateMemberRequest,
  ListIssuesParams,
  ListGroupedIssuesParams,
  IssueTableFacetsRequest,
  IssueTableFacetsResponse,
  IssueTableGroupsRequest,
  IssueTableGroupsResponse,
  IssueTableRowsRequest,
  IssueTableRowsResponse,
  IssueSubscriber,
  Comment,
  Reaction,
  IssueReaction,
  Workspace,
  WorkspaceRepo,
  WorkspacePermissionCatalog,
  MemberWithUser,
  User,
  Skill,
  SkillSummary,
  CreateSkillRequest,
  UpdateSkillRequest,
  PersonalAccessToken,
  CreatePersonalAccessTokenRequest,
  CreatePersonalAccessTokenResponse,
  TimelineEntry,
  Attachment,
  Project,
  CreateProjectRequest,
  UpdateProjectRequest,
  ListProjectsResponse,
  Task,
  CreateTaskRequest,
  UpdateTaskRequest,
  ListTasksResponse,
  ProjectResource,
  CreateProjectResourceRequest,
  UpdateProjectResourceRequest,
  ListProjectResourcesResponse,
  Label,
  IssueProperty,
  IssuePropertyValue,
  CreatePropertyRequest,
  UpdatePropertyRequest,
  ListPropertiesResponse,
  IssuePropertiesResponse,
  CreateLabelRequest,
  UpdateLabelRequest,
  ListLabelsResponse,
  IssueLabelsResponse,
  LabelResourceType,
  ResourceLabelsResponse,
  PinnedItem,
  CreatePinRequest,
  PinnedItemType,
  ReorderPinsRequest,
  Invitation,
  NotificationPreferenceResponse,
  NotificationPreferences,
  GitHubPullRequest,
  ListGitHubInstallationsResponse,
  ListGitHubRepositoriesResponse,
  GitHubConnectResponse,
  ListVCSConnectionsResponse,
  ConnectVCSRequest,
  ConnectVCSResponse,
  KnowledgeCandidate,
  KnowledgeCandidateListResponse,
  KnowledgeEntry,
  KnowledgeListResponse,
  ProposeKnowledgeRequest,
  ReviewKnowledgeRequest,
  ReviewKnowledgeResponse,
  CommentKnowledgeProposalResponse,
  AcceptanceConclusion,
  AcceptanceConclusionInput,
  AcceptanceConclusionListResponse,
  ProjectRetrospective,
  ProjectRetrospectiveInput,
  ProjectRetrospectiveListResponse,
  ProjectRequirementBaselineResponse,
  SaveProjectRequirementDraftRequest,
  ProjectRequirementTransitionRequest,
  ProjectRequirementCoverage,
  ProjectRequirementLinkRequest,
  ProjectRequirementCreateIssueRequest,
} from "../types";
import type { OnboardingCompletionPath } from "../onboarding/types";
import type { CreateFeedbackResponse, FeedbackKind } from "../feedback/types";
import { type Logger, noopLogger } from "../logger";
import { createRequestId } from "../utils";
import { getCurrentSlug, getCurrentWsId } from "../platform/workspace-storage";
import { parseWithFallback } from "./schema";
import {
  EMPTY_KNOWLEDGE_CANDIDATE_LIST,
  EMPTY_KNOWLEDGE_LIST,
  knowledgeCandidateListSchema,
  knowledgeCandidateSchema,
  knowledgeEntrySchema,
  knowledgeListSchema,
  reviewKnowledgeResponseSchema,
  commentKnowledgeProposalResponseSchema,
} from "../knowledge/schema";
import {
  EMPTY_ACCEPTANCE_CONCLUSION_LIST,
  EMPTY_RETROSPECTIVE_LIST,
  acceptanceConclusionListSchema,
  acceptanceConclusionSchema,
  projectRetrospectiveListSchema,
  projectRetrospectiveSchema,
} from "../implementation-knowledge/schema";
import {
  EMPTY_PROJECT_REQUIREMENT_BASELINE,
	EMPTY_PROJECT_REQUIREMENT_COVERAGE,
  projectRequirementBaselineResponseSchema,
	projectRequirementCoverageSchema,
} from "../project-requirements/schema";
import {
  AttachmentResponseSchema,
  ChildIssuesResponseSchema,
  CommentsListSchema,
  EMPTY_APP_CONFIG,
  EMPTY_ATTACHMENT,
  EMPTY_GROUPED_ISSUES_RESPONSE,
  EMPTY_ISSUE_TABLE_FACETS_RESPONSE,
  EMPTY_ISSUE_TABLE_GROUPS_RESPONSE,
  EMPTY_ISSUE_TABLE_ROWS_RESPONSE,
  EMPTY_LIST_ISSUES_RESPONSE,
  EMPTY_SEARCH_ISSUES_RESPONSE,
  IssueMetadataResponseSchema,
  EMPTY_SEARCH_PROJECTS_RESPONSE,
  EMPTY_LIST_PROJECTS_RESPONSE,
  EMPTY_PINS,
  EMPTY_TIMELINE_ENTRIES,
  EMPTY_USER,
  AppConfigSchema,
  type AppConfigResponse,
  GroupedIssuesResponseSchema,
  IssueTableFacetsResponseSchema,
  IssueTableGroupsResponseSchema,
  IssueTableRowsResponseSchema,
  IssueSchema,
  ListIssuesResponseSchema,
  CreateIssueResponseSchema,
  SearchIssuesResponseSchema,
  SearchProjectsResponseSchema,
  ListProjectsResponseSchema,
  ProjectSchema,
  PinSchema,
  PinsSchema,
  SubscribersListSchema,
  TimelineEntriesSchema,
  UserSchema,
  LoginResponseSchema,
  CreateFeedbackResponseSchema,
  EMPTY_CREATE_FEEDBACK_RESPONSE,
  NotificationPreferenceResponseSchema,
  EMPTY_NOTIFICATION_PREFERENCE_RESPONSE,
  LabelSchema,
  ListLabelsResponseSchema,
  IssuePropertySchema,
  ListPropertiesResponseSchema,
  IssuePropertiesResponseSchema,
  EMPTY_ISSUE_PROPERTY,
  EMPTY_LIST_PROPERTIES_RESPONSE,
  EMPTY_ISSUE_PROPERTIES_RESPONSE,
  EMPTY_ISSUE_PULL_REQUESTS_RESPONSE,
  IssuePullRequestsResponseSchema,
  ResourceLabelsResponseSchema,
  EMPTY_LABEL,
  EMPTY_LIST_LABELS_RESPONSE,
  EMPTY_RESOURCE_LABELS_RESPONSE,
  GitHubConnectResponseSchema,
  ListGitHubInstallationsResponseSchema,
  ListGitHubRepositoriesResponseSchema,
  EMPTY_GITHUB_CONNECT_RESPONSE,
  EMPTY_LIST_GITHUB_INSTALLATIONS_RESPONSE,
  EMPTY_LIST_GITHUB_REPOSITORIES_RESPONSE,
  WorkspacePermissionCatalogSchema,
  EMPTY_WORKSPACE_PERMISSION_CATALOG,
  MemberWithUserSchema,
  MemberListSchema,
  InvitationSchema,
  InvitationListSchema,
  WorkspaceListSchema,
  WorkspaceSchema,
  EMPTY_WORKSPACE_LIST,
  OnboardingCompletionResponseSchema,
} from "./schemas";

/** Identifies the calling client to the server.
 *  Sent on every HTTP request as X-Client-Platform / X-Client-Version /
 *  X-Client-OS so the backend can log, gate, or split metrics by client.
 *  See server/internal/middleware/client.go for the receiving end. */
export interface ApiClientIdentity {
  /** Logical client kind. Server expects: "web" | "desktop" | "cli". */
  platform?: string;
  /** Client/app version string (e.g. "0.1.0", git tag, commit). */
  version?: string;
  /** Coarse operating-system bucket (for example "macos", "windows", or "linux"). */
  os?: string;
}

export interface ApiClientOptions {
  logger?: Logger;
  onUnauthorized?: () => void;
  /** Identifies the client to the server. Sent as X-Client-* headers. */
  identity?: ApiClientIdentity;
}


export interface LoginResponse {
  token: string;
  user: User;
}

export class ApiError extends Error {
  readonly status: number;
  readonly statusText: string;
  // Raw decoded JSON body (when the server returned one). Carries structured
  // error fields like `code` so callers can branch on machine-readable
  // identifiers instead of pattern-matching the human-readable message.
  readonly body?: unknown;

  constructor(message: string, status: number, statusText: string, body?: unknown) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.statusText = statusText;
    this.body = body;
  }
}

// dispatchReasonCode extracts the stable, machine-readable admission reason
// (MUL-4525) from a blocked-trigger error's structured body, when present. UI
// callers localize a blocked/partial trigger from this code instead of pattern
// matching the human-readable message. Returns undefined for non-ApiErrors or
// bodies without a reason_code (older servers), so callers fall back to their
// generic failure toast.
export function dispatchReasonCode(err: unknown): string | undefined {
  if (err instanceof ApiError && err.body && typeof err.body === "object") {
    const code = (err.body as { reason_code?: unknown }).reason_code;
    if (typeof code === "string" && code.length > 0) return code;
  }
  return undefined;
}

// Thrown by getAttachmentTextContent when the server refuses to inline a
// file because it exceeds the 2 MB cap. UI maps to a "too large, please
// download" affordance with the Download CTA still available.
export class PreviewTooLargeError extends Error {
  constructor() {
    super("attachment too large for inline preview");
    this.name = "PreviewTooLargeError";
  }
}

// Thrown by getAttachmentTextContent when the server's text whitelist
// rejects the content type. Normally the client's isPreviewable() guard
// catches this earlier, but the two whitelists can drift — surfacing the
// 415 as a typed error makes the drift visible.
export class PreviewUnsupportedError extends Error {
  constructor() {
    super("attachment type not supported for inline preview");
    this.name = "PreviewUnsupportedError";
  }
}


export class ApiClient {
  private baseUrl: string;
  private token: string | null = null;
  private logger: Logger;
  private options: ApiClientOptions;

  constructor(baseUrl: string, options?: ApiClientOptions) {
    this.baseUrl = baseUrl;
    this.options = options ?? {};
    this.logger = options?.logger ?? noopLogger;
  }

  getBaseUrl(): string {
    return this.baseUrl;
  }

  setToken(token: string | null) {
    this.token = token;
  }

  private readCsrfToken(): string | null {
    if (typeof document === "undefined") return null;
    const match = document.cookie
      .split("; ")
      .find((c) => c.startsWith("multica_csrf="));
    return match ? match.split("=")[1] ?? null : null;
  }

  private authHeaders(): Record<string, string> {
    const headers: Record<string, string> = {};
    if (this.token) headers["Authorization"] = `Bearer ${this.token}`;
    const slug = getCurrentSlug();
    if (slug) headers["X-Workspace-Slug"] = slug;
    const csrf = this.readCsrfToken();
    if (csrf) headers["X-CSRF-Token"] = csrf;
    const id = this.options.identity;
    if (id?.platform) headers["X-Client-Platform"] = id.platform;
    if (id?.version) headers["X-Client-Version"] = id.version;
    if (id?.os) headers["X-Client-OS"] = id.os;
    return headers;
  }

  private handleUnauthorized() {
    this.token = null;
    // Workspace id is owned by the URL-driven workspace-storage singleton
    // (set by [workspaceSlug]/layout.tsx). On 401, the auth flow navigates
    // to /login which leaves the workspace route, and the next workspace
    // entry will overwrite the id. No clear needed here.
    this.options.onUnauthorized?.();
  }

  private async parseErrorMessage(res: Response, fallback: string): Promise<string> {
    try {
      const data = await res.json() as { error?: string };
      if (typeof data.error === "string" && data.error) return data.error;
    } catch {
      // Ignore non-JSON error bodies.
    }
    return fallback;
  }

  // Reads the response body once for both human-readable error message and
  // structured fields. The Response stream can only be consumed once, so
  // both pieces have to come from a single read.
  private async parseErrorBody(res: Response, fallback: string): Promise<{ message: string; body: unknown }> {
    try {
      const data = await res.json() as { error?: string };
      const message = typeof data.error === "string" && data.error ? data.error : fallback;
      return { message, body: data };
    } catch {
      return { message: fallback, body: undefined };
    }
  }

  // Sends the request with the standard headers (auth, CSRF, request id,
  // client identity) and runs the shared error path (401 → handleUnauthorized,
  // structured ApiError, status-aware log level). Returns the raw Response so
  // callers can decide how to decode the body — JSON for the typed `fetch<T>`
  // path, plain text for the attachment-preview proxy, etc.
  private async fetchRaw(
    path: string,
    init?: RequestInit & { extraHeaders?: Record<string, string> },
  ): Promise<Response> {
    const rid = createRequestId();
    const start = Date.now();
    const method = init?.method ?? "GET";

    const headers: Record<string, string> = {
      "X-Request-ID": rid,
      ...this.authHeaders(),
      ...(init?.extraHeaders ?? {}),
      ...((init?.headers as Record<string, string>) ?? {}),
    };

    this.logger.info(`→ ${method} ${path}`, { rid });

    const res = await fetch(`${this.baseUrl}${path}`, {
      ...init,
      headers,
      credentials: "include",
    });

    if (!res.ok) {
      if (res.status === 401) this.handleUnauthorized();
      const { message, body } = await this.parseErrorBody(res, `API error: ${res.status} ${res.statusText}`);
      const logLevel = res.status === 404 ? "warn" : "error";
      this.logger[logLevel](`← ${res.status} ${path}`, { rid, duration: `${Date.now() - start}ms`, error: message });
      throw new ApiError(message, res.status, res.statusText, body);
    }

    this.logger.info(`← ${res.status} ${path}`, { rid, duration: `${Date.now() - start}ms` });
    return res;
  }

  private async fetch<T>(path: string, init?: RequestInit): Promise<T> {
    const res = await this.fetchRaw(path, {
      ...init,
      extraHeaders: { "Content-Type": "application/json" },
    });
    // Handle 204 No Content
    if (res.status === 204) {
      return undefined as T;
    }
    return res.json() as Promise<T>;
  }

  /** Authenticated transport for the canonical control-plane process. */
  async requestControlPlane(path: string, init?: RequestInit): Promise<Response> {
    const parsed = new URL(path, "https://control-plane.invalid");
    if (
      parsed.origin !== "https://control-plane.invalid"
      || !parsed.pathname.startsWith("/v1/")
      || parsed.search !== ""
      || parsed.hash !== ""
      || path.includes("\\")
      || parsed.pathname !== path
    ) {
      throw new Error("control-plane path must be a normalized /v1/ path");
    }
    return this.fetchRaw(`/control-plane${path}`, init);
  }

  // Auth
  async sendCode(email: string): Promise<void> {
    await this.fetch("/auth/send-code", {
      method: "POST",
      body: JSON.stringify({ email }),
    });
  }

  async verifyCode(email: string, code: string): Promise<LoginResponse> {
    const raw = await this.fetch<unknown>("/auth/verify-code", {
      method: "POST",
      body: JSON.stringify({ email, code }),
    });
    const parsed = LoginResponseSchema.safeParse(raw);
    if (!parsed.success) throw new Error("Invalid authentication response");
    return parsed.data;
  }

  async googleLogin(code: string, redirectUri: string): Promise<LoginResponse> {
    return this.fetch("/auth/google", {
      method: "POST",
      body: JSON.stringify({ code, redirect_uri: redirectUri }),
    });
  }

  async logout(): Promise<void> {
    await this.fetch("/auth/logout", { method: "POST" });
  }

  async issueCliToken(): Promise<{ token: string }> {
    return this.fetch("/api/cli-token", { method: "POST" });
  }

  async getMe(): Promise<User> {
    const raw = await this.fetch<unknown>("/api/me");
    return parseWithFallback(raw, UserSchema, EMPTY_USER, {
      endpoint: "GET /api/me",
    });
  }

  async markOnboardingComplete(payload?: {
    completion_path?: OnboardingCompletionPath;
    workspace_id?: string;
  }): Promise<User> {
    const raw = await this.fetch<unknown>("/api/me/onboarding/complete", {
      method: "POST",
      body: payload ? JSON.stringify(payload) : undefined,
    });
    const parsed = OnboardingCompletionResponseSchema.safeParse(raw);
    if (!parsed.success) {
      throw new Error("Invalid onboarding completion response");
    }
    return parsed.data;
  }

  async updateMe(data: UpdateMeRequest): Promise<User> {
    const raw = await this.fetch<unknown>("/api/me", {
      method: "PATCH",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, UserSchema, EMPTY_USER, {
      endpoint: "PATCH /api/me",
    });
  }

  // Issues
  async listIssues(params?: ListIssuesParams): Promise<ListIssuesResponse> {
    const search = new URLSearchParams();
    if (params?.limit) search.set("limit", String(params.limit));
    if (params?.offset) search.set("offset", String(params.offset));
    if (params?.workspace_id) search.set("workspace_id", params.workspace_id);
    if (params?.q?.trim()) search.set("q", params.q.trim());
    if (params?.status) search.set("status", params.status);
    if (params?.statuses?.length) search.set("statuses", params.statuses.join(","));
    if (params?.priority) search.set("priority", params.priority);
    if (params?.priorities?.length) search.set("priorities", params.priorities.join(","));
    if (params?.assignee_id) search.set("assignee_id", params.assignee_id);
    if (params?.assignee_ids?.length) search.set("assignee_ids", params.assignee_ids.join(","));
    if (params?.assignee_types?.length) search.set("assignee_types", params.assignee_types.join(","));
    if (params?.creator_id) search.set("creator_id", params.creator_id);
    if (params?.project_id) search.set("project_id", params.project_id);
    if (params?.assignee_filters?.length) {
      search.set("assignee_filters", params.assignee_filters.map((f) => `${f.type}:${f.id}`).join(","));
    }
    if (params?.include_no_assignee) search.set("include_no_assignee", "true");
    if (params?.creator_filters?.length) {
      search.set("creator_filters", params.creator_filters.map((f) => `${f.type}:${f.id}`).join(","));
    }
    if (params?.project_ids?.length) search.set("project_ids", params.project_ids.join(","));
    if (params?.include_no_project) search.set("include_no_project", "true");
    if (params?.label_ids?.length) search.set("label_ids", params.label_ids.join(","));
    if (params?.top_level_only) search.set("top_level_only", "true");
    // No `.length` guard on purpose: an empty ids array must still send
    // `ids=` — the server treats a PRESENT-but-empty list as an empty window
    // (nothing running), while an absent param means no restriction.
    if (params?.ids) search.set("ids", params.ids.join(","));
    if (params?.metadata && Object.keys(params.metadata).length > 0) {
      search.set("metadata", JSON.stringify(params.metadata));
    }
    if (params?.properties && Object.keys(params.properties).length > 0) {
      search.set("properties", JSON.stringify(params.properties));
    }
    if (params?.open_only) search.set("open_only", "true");
    if (params?.scheduled) search.set("scheduled", "true");
    if (params?.date_field) search.set("date_field", params.date_field);
    if (params?.date_start) search.set("date_start", params.date_start);
    if (params?.date_end) search.set("date_end", params.date_end);
    if (params?.sort_by) search.set("sort", params.sort_by);
    if (params?.sort_direction) search.set("direction", params.sort_direction);
    // An ids facet can carry hundreds of UUIDs — enough to blow the ~8 KB
    // request-line cap of common reverse proxies.
    // Route those windows through the POST twin, which takes the SAME
    // key/value pairs as a JSON body.
    if (params?.ids) {
      const raw = await this.fetch<unknown>("/api/issues/query", {
        method: "POST",
        body: JSON.stringify(Object.fromEntries(search)),
      });
      return parseWithFallback(raw, ListIssuesResponseSchema, EMPTY_LIST_ISSUES_RESPONSE, {
        endpoint: "POST /api/issues/query",
      });
    }
    const path = `/api/issues?${search}`;
    const raw = await this.fetch<unknown>(path);
    return parseWithFallback(raw, ListIssuesResponseSchema, EMPTY_LIST_ISSUES_RESPONSE, {
      endpoint: "GET /api/issues",
    });
  }

  async listGroupedIssues(params: ListGroupedIssuesParams): Promise<GroupedIssuesResponse> {
    const search = new URLSearchParams({ group_by: params.group_by });
    if (params.limit) search.set("limit", String(params.limit));
    if (params.offset) search.set("offset", String(params.offset));
    if (params.workspace_id) search.set("workspace_id", params.workspace_id);
    if (params.statuses?.length) search.set("statuses", params.statuses.join(","));
    if (params.priorities?.length) search.set("priorities", params.priorities.join(","));
    if (params.assignee_types?.length) search.set("assignee_types", params.assignee_types.join(","));
    if (params.assignee_id) search.set("assignee_id", params.assignee_id);
    if (params.assignee_ids?.length) search.set("assignee_ids", params.assignee_ids.join(","));
    if (params.creator_id) search.set("creator_id", params.creator_id);
    if (params.project_id) search.set("project_id", params.project_id);
    if (params.metadata && Object.keys(params.metadata).length > 0) {
      search.set("metadata", JSON.stringify(params.metadata));
    }
    if (params.properties && Object.keys(params.properties).length > 0) {
      search.set("properties", JSON.stringify(params.properties));
    }
    if (params.assignee_filters?.length) {
      search.set("assignee_filters", params.assignee_filters.map((f) => `${f.type}:${f.id}`).join(","));
    }
    if (params.include_no_assignee) search.set("include_no_assignee", "true");
    if (params.creator_filters?.length) {
      search.set("creator_filters", params.creator_filters.map((f) => `${f.type}:${f.id}`).join(","));
    }
    if (params.project_ids?.length) search.set("project_ids", params.project_ids.join(","));
    if (params.include_no_project) search.set("include_no_project", "true");
    if (params.label_ids?.length) search.set("label_ids", params.label_ids.join(","));
    if (params.group_assignee_type) search.set("group_assignee_type", params.group_assignee_type);
    if (params.group_assignee_id) search.set("group_assignee_id", params.group_assignee_id);
    if (params.date_field) search.set("date_field", params.date_field);
    if (params.date_start) search.set("date_start", params.date_start);
    if (params.date_end) search.set("date_end", params.date_end);
    if (params.sort_by) search.set("sort", params.sort_by);
    if (params.sort_direction) search.set("direction", params.sort_direction);
    const raw = await this.fetch<unknown>(`/api/issues/grouped?${search}`);
    return parseWithFallback(raw, GroupedIssuesResponseSchema, EMPTY_GROUPED_ISSUES_RESPONSE, {
      endpoint: "GET /api/issues/grouped",
    });
  }

  async listIssueTableGroups(params: IssueTableGroupsRequest): Promise<IssueTableGroupsResponse> {
    const raw = await this.fetch<unknown>("/api/issues/table/groups", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return parseWithFallback(
      raw,
      IssueTableGroupsResponseSchema,
      EMPTY_ISSUE_TABLE_GROUPS_RESPONSE,
      { endpoint: "POST /api/issues/table/groups" },
    );
  }

  async listIssueTableRows(params: IssueTableRowsRequest): Promise<IssueTableRowsResponse> {
    const raw = await this.fetch<unknown>("/api/issues/table/rows", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return parseWithFallback(
      raw,
      IssueTableRowsResponseSchema,
      EMPTY_ISSUE_TABLE_ROWS_RESPONSE,
      { endpoint: "POST /api/issues/table/rows" },
    );
  }

  async listIssueTableFacets(params: IssueTableFacetsRequest): Promise<IssueTableFacetsResponse> {
    const raw = await this.fetch<unknown>("/api/issues/table/facets", {
      method: "POST",
      body: JSON.stringify(params),
    });
    return parseWithFallback(
      raw,
      IssueTableFacetsResponseSchema,
      EMPTY_ISSUE_TABLE_FACETS_RESPONSE,
      { endpoint: "POST /api/issues/table/facets" },
    );
  }

  async searchIssues(params: { q: string; limit?: number; offset?: number; include_closed?: boolean; signal?: AbortSignal }): Promise<SearchIssuesResponse> {
    const search = new URLSearchParams({ q: params.q });
    if (params.limit !== undefined) search.set("limit", String(params.limit));
    if (params.offset !== undefined) search.set("offset", String(params.offset));
    if (params.include_closed) search.set("include_closed", "true");
    const raw = await this.fetch<unknown>(
      `/api/issues/search?${search}`,
      params.signal ? { signal: params.signal } : undefined,
    );
    return parseWithFallback(raw, SearchIssuesResponseSchema, EMPTY_SEARCH_ISSUES_RESPONSE, {
      endpoint: "GET /api/issues/search",
    });
  }

  async searchProjects(params: { q: string; limit?: number; offset?: number; include_closed?: boolean; signal?: AbortSignal }): Promise<SearchProjectsResponse> {
    const search = new URLSearchParams({ q: params.q });
    if (params.limit !== undefined) search.set("limit", String(params.limit));
    if (params.offset !== undefined) search.set("offset", String(params.offset));
    if (params.include_closed) search.set("include_closed", "true");
    const raw = await this.fetch<unknown>(
      `/api/projects/search?${search}`,
      params.signal ? { signal: params.signal } : undefined,
    );
    return parseWithFallback(raw, SearchProjectsResponseSchema, EMPTY_SEARCH_PROJECTS_RESPONSE, {
      endpoint: "GET /api/projects/search",
    });
  }

  async getIssue(id: string): Promise<Issue> {
    const raw = await this.fetch<unknown>(`/api/issues/${id}`);
    const issue = parseWithFallback<Issue | null>(raw, IssueSchema.nullable(), null, {
      endpoint: "GET /api/issues/:id",
    });
    if (!issue) throw new Error("Invalid issue response");
    return issue;
  }

  async getIssueMetadata(id: string): Promise<IssueMetadata> {
    return this.parseIssueMetadata(
      await this.fetch<unknown>(`/api/issues/${id}/metadata`, { headers: this.issueMetadataWorkspaceHeader() }),
      "GET /api/issues/:id/metadata",
    );
  }

  async putIssueMetadata(id: string, key: string, value: IssueMetadataValue): Promise<IssueMetadata> {
    return this.parseIssueMetadata(
      await this.fetch<unknown>(`/api/issues/${id}/metadata/${encodeURIComponent(key)}`, {
        method: "PUT",
        headers: this.issueMetadataWorkspaceHeader(),
        body: JSON.stringify({ value }),
      }),
      "PUT /api/issues/:id/metadata/:key",
    );
  }

  async deleteIssueMetadata(id: string, key: string): Promise<IssueMetadata> {
    return this.parseIssueMetadata(
      await this.fetch<unknown>(`/api/issues/${id}/metadata/${encodeURIComponent(key)}`, {
        method: "DELETE",
        headers: this.issueMetadataWorkspaceHeader(),
      }),
      "DELETE /api/issues/:id/metadata/:key",
    );
  }

  private parseIssueMetadata(raw: unknown, endpoint: string): IssueMetadata {
    const response = parseWithFallback<IssueMetadataResponse | null>(
      raw,
      IssueMetadataResponseSchema.nullable(),
      null,
      { endpoint },
    );
    if (!response) throw new Error("Invalid issue metadata response");
    return response.metadata;
  }

  private issueMetadataWorkspaceHeader(): Record<string, string> {
    const workspaceId = getCurrentWsId();
    return workspaceId ? { "X-Workspace-ID": workspaceId } : {};
  }

  async createIssue(data: CreateIssueRequest): Promise<Issue> {
    // Parse through a schema (not a raw cast): the create modal keys its
    // label-attach compatibility fallback off `labels` being absent vs a
    // validated Label[], so an unvalidated wrong shape must not slip through.
    // Unlike list endpoints, a create that returns an unusable body is a
    // FAILED mutation, not a safe-empty read: fall back to null and reject so
    // the modal keeps the draft and shows a failure toast instead of a blank
    // "created" card pointing at an empty issue id. parseWithFallback already
    // logged the schema issues + raw payload; the empty message lets the modal
    // render its localized "failed to create" toast.
    const raw = await this.fetch<unknown>("/api/issues", {
      method: "POST",
      body: JSON.stringify(data),
    });
    const issue = parseWithFallback<Issue | null>(raw, CreateIssueResponseSchema, null, {
      endpoint: "POST /api/issues",
    });
    if (!issue) {
      throw new Error();
    }
    return issue;
  }

  async createFeedback(data: {
    message: string;
    url?: string;
    workspace_id?: string;
    kind?: FeedbackKind;
  }): Promise<CreateFeedbackResponse> {
    const raw = await this.fetch<unknown>("/api/feedback", {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, CreateFeedbackResponseSchema, EMPTY_CREATE_FEEDBACK_RESPONSE, {
      endpoint: "POST /api/feedback",
    });
  }

  async updateIssue(id: string, data: UpdateIssueRequest): Promise<Issue> {
    const { acceptanceConclusion, ...issueUpdate } = data;
    const raw = await this.fetch<unknown>(`/api/issues/${id}`, {
      method: "PUT",
      body: JSON.stringify({
        ...issueUpdate,
        ...(acceptanceConclusion ? {
          acceptance_conclusion: {
            result: acceptanceConclusion.result,
            rationale: acceptanceConclusion.rationale,
            evidence_refs: acceptanceConclusion.evidenceRefs,
          },
        } : {}),
      }),
    });
    const issue = parseWithFallback<Issue | null>(raw, IssueSchema.nullable(), null, {
      endpoint: "PUT /api/issues/:id",
    });
    if (!issue) throw new Error("Invalid issue update response");
    return issue;
  }

  async listIssueAcceptanceConclusions(id: string): Promise<AcceptanceConclusionListResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${id}/acceptance-conclusions`);
    return parseWithFallback(raw, acceptanceConclusionListSchema, EMPTY_ACCEPTANCE_CONCLUSION_LIST, {
      endpoint: "GET /api/issues/:id/acceptance-conclusions",
    });
  }

  async createIssueAcceptanceConclusion(
    id: string,
    input: AcceptanceConclusionInput,
  ): Promise<AcceptanceConclusion> {
    const raw = await this.fetch<unknown>(`/api/issues/${id}/acceptance-conclusions`, {
      method: "POST",
      body: JSON.stringify({
        result: input.result,
        rationale: input.rationale,
        evidence_refs: input.evidenceRefs,
      }),
    });
    const conclusion = parseWithFallback(raw, acceptanceConclusionSchema.nullable(), null, {
      endpoint: "POST /api/issues/:id/acceptance-conclusions",
    });
    if (!conclusion) throw new Error("Invalid acceptance conclusion response");
    return conclusion;
  }

  async moveIssue(id: string, data: MoveIssueRequest): Promise<Issue> {
    return this.fetch(`/api/issues/${id}/move`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async listChildIssues(id: string): Promise<{ issues: Issue[] }> {
    const raw = await this.fetch<unknown>(`/api/issues/${id}/children`);
    return parseWithFallback(raw, ChildIssuesResponseSchema, { issues: [] }, {
      endpoint: "GET /api/issues/:id/children",
    });
  }

  /** Batched variant — returns children for multiple parents in one request.
   *  Avoids an N-request fan-out in Swimlane (one per visible parent lane).
   *  parentIds must be non-empty; pass a sorted, deduplicated list so the
   *  React Query cache key is stable across renders. */
  async listChildrenByParents(parentIds: string[]): Promise<{ issues: Issue[] }> {
    const raw = await this.fetch<unknown>(
      `/api/issues/children?parent_ids=${parentIds.join(",")}`,
    );
    return parseWithFallback(raw, ChildIssuesResponseSchema, { issues: [] }, {
      endpoint: "GET /api/issues/children",
    });
  }

  async getChildIssueProgress(): Promise<{ progress: { parent_issue_id: string; total: number; done: number }[] }> {
    return this.fetch("/api/issues/child-progress");
  }

  async deleteIssue(id: string): Promise<void> {
    await this.fetch(`/api/issues/${id}`, { method: "DELETE" });
  }

  async batchUpdateIssues(issueIds: string[], updates: UpdateIssueRequest): Promise<{ updated: number }> {
    return this.fetch("/api/issues/batch-update", {
      method: "POST",
      body: JSON.stringify({ issue_ids: issueIds, updates }),
    });
  }

  async batchDeleteIssues(issueIds: string[]): Promise<{ deleted: number }> {
    return this.fetch("/api/issues/batch-delete", {
      method: "POST",
      body: JSON.stringify({ issue_ids: issueIds }),
    });
  }

  // Comments
  async listComments(issueId: string): Promise<Comment[]> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/comments`);
    return parseWithFallback(raw, CommentsListSchema, [], {
      endpoint: "GET /api/issues/:id/comments",
    });
  }

  async createComment(
    issueId: string,
    content: string,
    type?: string,
    parentId?: string,
    attachmentIds?: string[],
  ): Promise<Comment> {
    return this.fetch(`/api/issues/${issueId}/comments`, {
      method: "POST",
      body: JSON.stringify({
        content,
        type: type ?? "comment",
        ...(parentId ? { parent_id: parentId } : {}),
        ...(attachmentIds?.length ? { attachment_ids: attachmentIds } : {}),
      }),
    });
  }

  async listTimeline(issueId: string): Promise<TimelineEntry[]> {
    const raw = await this.fetch<unknown>(
      `/api/issues/${issueId}/timeline`,
    );
    return parseWithFallback(raw, TimelineEntriesSchema, EMPTY_TIMELINE_ENTRIES, {
      endpoint: "GET /api/issues/:id/timeline",
    });
  }

  async updateComment(commentId: string, content: string, attachmentIds?: string[]): Promise<Comment> {
    return this.fetch(`/api/comments/${commentId}`, {
      method: "PUT",
      body: JSON.stringify({
        content,
        attachment_ids: attachmentIds,
      }),
    });
  }

  async deleteComment(commentId: string): Promise<void> {
    await this.fetch(`/api/comments/${commentId}`, { method: "DELETE" });
  }

  async proposeCommentDecision(
    commentId: string,
  ): Promise<CommentKnowledgeProposalResponse> {
    const fallback: CommentKnowledgeProposalResponse = {
      queued: false,
      evidenceId: null,
      sourceRevision: "",
    };
    const raw = await this.fetch<unknown>(
      `/api/comments/${encodeURIComponent(commentId)}/knowledge-proposals`,
      { method: "POST" },
    );
    return parseWithFallback(raw, commentKnowledgeProposalResponseSchema, fallback, {
      endpoint: "POST /api/comments/:id/knowledge-proposals",
    });
  }

  async resolveComment(commentId: string): Promise<Comment> {
    return this.fetch(`/api/comments/${commentId}/resolve`, { method: "POST" });
  }

  async unresolveComment(commentId: string): Promise<Comment> {
    return this.fetch(`/api/comments/${commentId}/resolve`, { method: "DELETE" });
  }

  async addReaction(commentId: string, emoji: string): Promise<Reaction> {
    return this.fetch(`/api/comments/${commentId}/reactions`, {
      method: "POST",
      body: JSON.stringify({ emoji }),
    });
  }

  async removeReaction(commentId: string, emoji: string): Promise<void> {
    await this.fetch(`/api/comments/${commentId}/reactions`, {
      method: "DELETE",
      body: JSON.stringify({ emoji }),
    });
  }

  async addIssueReaction(issueId: string, emoji: string): Promise<IssueReaction> {
    return this.fetch(`/api/issues/${issueId}/reactions`, {
      method: "POST",
      body: JSON.stringify({ emoji }),
    });
  }

  async removeIssueReaction(issueId: string, emoji: string): Promise<void> {
    await this.fetch(`/api/issues/${issueId}/reactions`, {
      method: "DELETE",
      body: JSON.stringify({ emoji }),
    });
  }

  // Subscribers
  async listIssueSubscribers(issueId: string): Promise<IssueSubscriber[]> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/subscribers`);
    return parseWithFallback(raw, SubscribersListSchema, [], {
      endpoint: "GET /api/issues/:id/subscribers",
    });
  }

  async subscribeToIssue(issueId: string, userId?: string, userType?: string): Promise<void> {
    const body: Record<string, string> = {};
    if (userId) body.user_id = userId;
    if (userType) body.user_type = userType;
    await this.fetch(`/api/issues/${issueId}/subscribe`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  async unsubscribeFromIssue(issueId: string, userId?: string, userType?: string): Promise<void> {
    const body: Record<string, string> = {};
    if (userId) body.user_id = userId;
    if (userType) body.user_type = userType;
    await this.fetch(`/api/issues/${issueId}/unsubscribe`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }


  // Notification preferences
  //
  // `workspaceSlug` overrides the default `X-Workspace-Slug` header (which
  // follows the active workspace) so a caller can read a SPECIFIC workspace's
  // preferences — e.g. honoring the mute setting of the workspace an inbox
  // notification came from while the user is viewing a different one (#3766).
  async getNotificationPreferences(workspaceSlug?: string): Promise<NotificationPreferenceResponse> {
    const raw = await this.fetch<unknown>(
      "/api/notification-preferences",
      workspaceSlug ? { headers: { "X-Workspace-Slug": workspaceSlug } } : undefined,
    );
    return parseWithFallback(
      raw,
      NotificationPreferenceResponseSchema,
      EMPTY_NOTIFICATION_PREFERENCE_RESPONSE,
      { endpoint: "GET /api/notification-preferences" },
    );
  }

  async updateNotificationPreferences(
    preferences: NotificationPreferences,
    workspaceSlug?: string,
  ): Promise<NotificationPreferenceResponse> {
    const raw = await this.fetch<unknown>("/api/notification-preferences", {
      method: "PATCH",
      headers: workspaceSlug
        ? { "X-Workspace-Slug": workspaceSlug }
        : undefined,
      body: JSON.stringify({ preferences }),
    });
    return parseWithFallback(
      raw,
      NotificationPreferenceResponseSchema,
      EMPTY_NOTIFICATION_PREFERENCE_RESPONSE,
      { endpoint: "PATCH /api/notification-preferences" },
    );
  }

  // App Config
  async getConfig(): Promise<AppConfigResponse> {
    const raw = await this.fetch<unknown>("/api/config");
    return parseWithFallback<AppConfigResponse>(raw, AppConfigSchema, EMPTY_APP_CONFIG, {
      endpoint: "GET /api/config",
    });
  }

  // Workspaces
  async listWorkspaces(): Promise<Workspace[]> {
    const raw = await this.fetch<unknown>("/api/workspaces");
    return parseWithFallback<Workspace[]>(raw, WorkspaceListSchema, EMPTY_WORKSPACE_LIST, {
      endpoint: "GET /api/workspaces",
    });
  }

  async getWorkspace(id: string): Promise<Workspace> {
    return this.fetch(`/api/workspaces/${id}`);
  }

  async createWorkspace(data: { name: string; slug: string; description?: string; context?: string }): Promise<Workspace> {
    const raw = await this.fetch<unknown>("/api/workspaces", {
      method: "POST",
      body: JSON.stringify(data),
    });
    const parsed = WorkspaceSchema.safeParse(raw);
    if (!parsed.success) {
      throw new Error("Invalid workspace response");
    }
    return parsed.data;
  }

  async updateWorkspace(id: string, data: { name?: string; description?: string; context?: string; settings?: Record<string, unknown>; repos?: WorkspaceRepo[]; issue_prefix?: string; avatar_url?: string }): Promise<Workspace> {
    return this.fetch(`/api/workspaces/${id}`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  }

  // Members
  async listMembers(workspaceId: string): Promise<MemberWithUser[]> {
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/members`,
    );
    return parseWithFallback<MemberWithUser[]>(
      raw,
      MemberListSchema,
      [],
      { endpoint: "GET /api/workspaces/:id/members" },
    );
  }

  async getWorkspacePermissions(workspaceId: string): Promise<WorkspacePermissionCatalog> {
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/permissions`,
    );
    return parseWithFallback(
      raw,
      WorkspacePermissionCatalogSchema,
      EMPTY_WORKSPACE_PERMISSION_CATALOG,
      { endpoint: "GET /api/workspaces/:id/permissions" },
    );
  }

  async createMember(workspaceId: string, data: CreateMemberRequest): Promise<Invitation> {
    const raw = await this.fetch<unknown>(`/api/workspaces/${workspaceId}/members`, {
      method: "POST",
      body: JSON.stringify(data),
    });
    const invitation = parseWithFallback<Invitation | null>(
      raw,
      InvitationSchema.nullable(),
      null,
      { endpoint: "POST /api/workspaces/:id/members" },
    );
    if (!invitation) throw new Error("Invalid invitation response");
    return invitation;
  }

  async updateMember(
    workspaceId: string,
    memberId: string,
    data: UpdateMemberRequest,
  ): Promise<MemberWithUser> {
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/members/${memberId}`,
      {
        method: "PATCH",
        body: JSON.stringify(data),
      },
    );
    const member = parseWithFallback<MemberWithUser | null>(
      raw,
      MemberWithUserSchema.nullable(),
      null,
      { endpoint: "PATCH /api/workspaces/:id/members/:memberId" },
    );
    if (!member) throw new Error("Invalid member update response");
    return member;
  }

  async deleteMember(workspaceId: string, memberId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}/members/${memberId}`, {
      method: "DELETE",
    });
  }

  async leaveWorkspace(workspaceId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}/leave`, {
      method: "POST",
    });
  }

  // Invitations
  async listWorkspaceInvitations(workspaceId: string): Promise<Invitation[]> {
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/invitations`,
    );
    return parseWithFallback<Invitation[]>(raw, InvitationListSchema, [], {
      endpoint: "GET /api/workspaces/:id/invitations",
    });
  }

  async revokeInvitation(workspaceId: string, invitationId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}/invitations/${invitationId}`, {
      method: "DELETE",
    });
  }

  async listMyInvitations(): Promise<Invitation[]> {
    const raw = await this.fetch<unknown>("/api/invitations");
    return parseWithFallback<Invitation[]>(raw, InvitationListSchema, [], {
      endpoint: "GET /api/invitations",
    });
  }

  async getInvitation(invitationId: string): Promise<Invitation> {
    const raw = await this.fetch<unknown>(`/api/invitations/${invitationId}`);
    const invitation = parseWithFallback<Invitation | null>(
      raw,
      InvitationSchema.nullable(),
      null,
      { endpoint: "GET /api/invitations/:id" },
    );
    if (!invitation) throw new Error("Invalid invitation response");
    return invitation;
  }

  async acceptInvitation(invitationId: string): Promise<MemberWithUser> {
    const raw = await this.fetch<unknown>(`/api/invitations/${invitationId}/accept`, {
      method: "POST",
    });
    const member = parseWithFallback<MemberWithUser | null>(
      raw,
      MemberWithUserSchema.nullable(),
      null,
      { endpoint: "POST /api/invitations/:id/accept" },
    );
    if (!member) throw new Error("Invalid member response");
    return member;
  }

  async declineInvitation(invitationId: string): Promise<void> {
    await this.fetch(`/api/invitations/${invitationId}/decline`, {
      method: "POST",
    });
  }

  async deleteWorkspace(workspaceId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}`, {
      method: "DELETE",
    });
  }

  // Skills
  async listSkills(): Promise<SkillSummary[]> {
    return this.fetch("/api/skills");
  }

  async getSkill(id: string): Promise<Skill> {
    return this.fetch(`/api/skills/${id}`);
  }

  async createSkill(data: CreateSkillRequest): Promise<Skill> {
    return this.fetch("/api/skills", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateSkill(id: string, data: UpdateSkillRequest): Promise<Skill> {
    return this.fetch(`/api/skills/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async deleteSkill(id: string): Promise<void> {
    await this.fetch(`/api/skills/${id}`, { method: "DELETE" });
  }

  async importSkill(data: { url: string }): Promise<Skill> {
    return this.fetch("/api/skills/import", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }


  // Personal Access Tokens
  async listPersonalAccessTokens(): Promise<PersonalAccessToken[]> {
    return this.fetch("/api/tokens");
  }

  async createPersonalAccessToken(data: CreatePersonalAccessTokenRequest): Promise<CreatePersonalAccessTokenResponse> {
    return this.fetch("/api/tokens", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async revokePersonalAccessToken(id: string): Promise<void> {
    await this.fetch(`/api/tokens/${id}`, { method: "DELETE" });
  }

  // File Upload & Attachments
  async uploadFile(
    file: File,
    opts?: { issueId?: string; commentId?: string; chatSessionId?: string },
    // Optional abort signal so a module-level upload coordinator (MUL-5181)
    // can cancel an in-flight upload on logout. When aborted, `fetch` rejects
    // with an AbortError, which the coordinator distinguishes from a real
    // failure via `signal.aborted` / `err.name === "AbortError"`.
    signal?: AbortSignal,
  ): Promise<Attachment> {
    const formData = new FormData();
    formData.append("file", file);
    if (opts?.issueId) formData.append("issue_id", opts.issueId);
    if (opts?.commentId) formData.append("comment_id", opts.commentId);
    if (opts?.chatSessionId) formData.append("chat_session_id", opts.chatSessionId);

    const rid = createRequestId();
    const start = Date.now();
    this.logger.info("→ POST /api/upload-file", { rid });

    const res = await fetch(`${this.baseUrl}/api/upload-file`, {
      method: "POST",
      headers: this.authHeaders(),
      body: formData,
      credentials: "include",
      signal,
    });

    if (!res.ok) {
      if (res.status === 401) this.handleUnauthorized();
      const message = await this.parseErrorMessage(res, `Upload failed: ${res.status}`);
      this.logger.error(`← ${res.status} /api/upload-file`, { rid, duration: `${Date.now() - start}ms`, error: message });
      throw new Error(message);
    }

    this.logger.info(`← ${res.status} /api/upload-file`, { rid, duration: `${Date.now() - start}ms` });
    const raw = (await res.json()) as unknown;
    const attachment = parseWithFallback<Attachment | null>(raw, AttachmentResponseSchema.nullable(), null, {
      endpoint: "POST /api/upload-file",
    });
    if (!attachment) throw new Error("Invalid attachment response");
    return attachment;
  }


  async listAttachments(issueId: string): Promise<Attachment[]> {
    return this.fetch(`/api/issues/${issueId}/attachments`);
  }

  // Fetches a fresh attachment metadata record. The server re-signs
  // `download_url` on every call (30 min expiry), so the click-time
  // download flow uses this endpoint to avoid handing the user a stale
  // signed URL cached in TanStack Query.
  async getAttachment(id: string): Promise<Attachment> {
    const raw = await this.fetch<unknown>(`/api/attachments/${id}`);
    return parseWithFallback(raw, AttachmentResponseSchema, EMPTY_ATTACHMENT, {
      endpoint: "GET /api/attachments/{id}",
    });
  }

  async deleteAttachment(id: string): Promise<void> {
    await this.fetch(`/api/attachments/${id}`, { method: "DELETE" });
  }

  // Fetches the raw bytes of a text-previewable attachment.
  //
  // The endpoint sidesteps CloudFront CORS (not configured on the CDN) and
  // bypasses Content-Disposition: attachment for the `text/*` family, both
  // of which would otherwise prevent the renderer from getting the body.
  // The server always replies with `text/plain; charset=utf-8` for safety;
  // the original MIME ships back in the `X-Original-Content-Type` header so
  // the preview dispatcher can choose between markdown / html / plain code.
  //
  // Routes through `fetchRaw` so it inherits the standard auth headers,
  // 401 → handleUnauthorized recovery, request-id logging, and ApiError
  // shape. 413 / 415 are translated to typed `Preview*Error` instances so
  // the modal can render specific fallbacks instead of generic failure.
  async getAttachmentTextContent(
    id: string,
  ): Promise<{ text: string; originalContentType: string }> {
    let res: Response;
    try {
      res = await this.fetchRaw(`/api/attachments/${id}/content`);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 413) throw new PreviewTooLargeError();
        if (err.status === 415) throw new PreviewUnsupportedError();
      }
      throw err;
    }
    return {
      text: await res.text(),
      originalContentType: res.headers.get("X-Original-Content-Type") ?? "",
    };
  }

  // Fetches the raw bytes of an attachment through the unified download
  // endpoint.
  //
  // This is the last-resort inline-media path for deployments where the
  // server has no natively-loadable URL to offer. `GET /api/attachments/{id}`
  // only upgrades `download_url` to a signed storage URL under CloudFront
  // signing or presign mode; in **proxy** mode (self-hosted MinIO or any
  // storage endpoint on an internal host, which the default `auto` mode
  // classifies as proxy) it returns the auth-gated API path again. Clients
  // that cannot ride the session cookie on a native `<img>` resource fetch —
  // Desktop's file:// renderer, the mobile webview, split-origin web — get
  // the bytes here and render them from an object URL instead.
  //
  // Routes through `fetchRaw` so it inherits the standard auth headers,
  // 401 → handleUnauthorized recovery, request-id logging and ApiError shape.
  // Callers must only reach for this once the metadata refresh has shown
  // there is no signed URL: in the other modes the endpoint 302s to storage,
  // where CORS is not configured for a JS fetch.
  async getAttachmentBlob(id: string): Promise<Blob> {
    const res = await this.fetchRaw(`/api/attachments/${id}/download`);
    return res.blob();
  }

  // Projects
  async listProjects(params?: { status?: string }): Promise<ListProjectsResponse> {
    const search = new URLSearchParams();
    if (params?.status) search.set("status", params.status);
    const raw = await this.fetch<unknown>(`/api/projects?${search}`);
    return parseWithFallback(raw, ListProjectsResponseSchema, EMPTY_LIST_PROJECTS_RESPONSE, {
      endpoint: "GET /api/projects",
    });
  }

  async getProject(id: string): Promise<Project> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}`);
    const parsed = ProjectSchema.safeParse(raw);
    if (!parsed.success) throw new Error("Invalid project response");
    return parsed.data;
  }

  async createProject(data: CreateProjectRequest): Promise<Project> {
    const raw = await this.fetch<unknown>("/api/projects", {
      method: "POST",
      body: JSON.stringify(data),
    });
    const parsed = ProjectSchema.safeParse(raw);
    if (!parsed.success) throw new Error("Invalid project response");
    return parsed.data;
  }

  async updateProject(id: string, data: UpdateProjectRequest): Promise<Project> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
    const parsed = ProjectSchema.safeParse(raw);
    if (!parsed.success) throw new Error("Invalid project response");
    return parsed.data;
  }

  async listProjectRetrospectives(id: string): Promise<ProjectRetrospectiveListResponse> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/retrospectives`);
    return parseWithFallback(raw, projectRetrospectiveListSchema, EMPTY_RETROSPECTIVE_LIST, {
      endpoint: "GET /api/projects/:id/retrospectives",
    });
  }

  async createProjectRetrospective(
    id: string,
    input: ProjectRetrospectiveInput,
  ): Promise<ProjectRetrospective> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/retrospectives`, {
      method: "POST",
      body: JSON.stringify({
        summary: input.summary,
        successes: input.successes,
        problems: input.problems,
        lessons: input.lessons,
        follow_up_refs: input.followUpRefs,
      }),
    });
    const retrospective = parseWithFallback(raw, projectRetrospectiveSchema.nullable(), null, {
      endpoint: "POST /api/projects/:id/retrospectives",
    });
    if (!retrospective) throw new Error("Invalid project retrospective response");
    return retrospective;
  }

  async deleteProject(id: string): Promise<void> {
    await this.fetch(`/api/projects/${id}`, { method: "DELETE" });
  }

  async getProjectRequirementBaseline(id: string): Promise<ProjectRequirementBaselineResponse> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/requirement-baseline`);
    return parseWithFallback(raw, projectRequirementBaselineResponseSchema, EMPTY_PROJECT_REQUIREMENT_BASELINE, {
      endpoint: "GET /api/projects/:id/requirement-baseline",
    });
  }

  async getProjectRequirementCoverage(id: string): Promise<ProjectRequirementCoverage> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/requirement-baseline/coverage`);
    return parseWithFallback(raw, projectRequirementCoverageSchema, EMPTY_PROJECT_REQUIREMENT_COVERAGE, { endpoint: "GET /api/projects/:id/requirement-baseline/coverage" });
  }

  async linkProjectRequirementIssue(id: string, input: ProjectRequirementLinkRequest): Promise<void> {
    await this.fetch(`/api/projects/${id}/requirement-baseline/links`, { method: "POST", body: JSON.stringify({ requirement_key: input.requirementKey, issue_id: input.issueId, revision: input.revision }) });
  }

  async unlinkProjectRequirementIssue(id: string, input: ProjectRequirementLinkRequest): Promise<void> {
    await this.fetch(`/api/projects/${id}/requirement-baseline/links/${encodeURIComponent(input.requirementKey)}/${encodeURIComponent(input.issueId)}?revision=${input.revision}`, { method: "DELETE" });
  }

  async createIssueForProjectRequirement(id: string, requirementKey: string, input: ProjectRequirementCreateIssueRequest): Promise<Issue> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/requirement-baseline/items/${encodeURIComponent(requirementKey)}/issues`, { method: "POST", body: JSON.stringify({ revision: input.revision }) });
    const issue = parseWithFallback<Issue | null>(raw, CreateIssueResponseSchema, null, { endpoint: "POST /api/projects/:id/requirement-baseline/items/:key/issues" });
    if (!issue) throw new Error("Invalid project requirement issue response");
    return issue;
  }

  async saveProjectRequirementDraft(id: string, input: SaveProjectRequirementDraftRequest): Promise<ProjectRequirementBaselineResponse> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/requirement-baseline`, {
      method: "PUT", body: JSON.stringify({
        expected_revision: input.expectedRevision,
        content: {
          problem_statement: input.content.problemStatement,
          goals: input.content.goals,
          in_scope: input.content.inScope,
          out_of_scope: input.content.outOfScope,
          constraints: input.content.constraints,
          acceptance_criteria: input.content.acceptanceCriteria,
          dependencies: input.content.dependencies,
        },
        change_summary: input.changeSummary,
      }),
    });
    const response = parseWithFallback(raw, projectRequirementBaselineResponseSchema, EMPTY_PROJECT_REQUIREMENT_BASELINE, {
      endpoint: "PUT /api/projects/:id/requirement-baseline",
    });
    if (!response.baseline) throw new Error("Invalid project requirement draft response");
    return response;
  }

  async submitProjectRequirementReview(id: string, input: ProjectRequirementTransitionRequest): Promise<ProjectRequirementBaselineResponse> {
    return this.transitionProjectRequirement(id, "submit-review", input, "POST /api/projects/:id/requirement-baseline/submit-review");
  }

  async approveProjectRequirement(id: string, input: ProjectRequirementTransitionRequest): Promise<ProjectRequirementBaselineResponse> {
    return this.transitionProjectRequirement(id, "approve", input, "POST /api/projects/:id/requirement-baseline/approve");
  }

  async withdrawProjectRequirementReview(id: string, input: ProjectRequirementTransitionRequest): Promise<ProjectRequirementBaselineResponse> {
    return this.transitionProjectRequirement(id, "withdraw", input, "POST /api/projects/:id/requirement-baseline/withdraw");
  }

  private async transitionProjectRequirement(id: string, action: string, input: ProjectRequirementTransitionRequest, endpoint: string): Promise<ProjectRequirementBaselineResponse> {
    const raw = await this.fetch<unknown>(`/api/projects/${id}/requirement-baseline/${action}`, {
      method: "POST", body: JSON.stringify({ expected_revision: input.expectedRevision }),
    });
    const response = parseWithFallback(raw, projectRequirementBaselineResponseSchema, EMPTY_PROJECT_REQUIREMENT_BASELINE, { endpoint });
    if (!response.baseline) throw new Error("Invalid project requirement transition response");
    return response;
  }

  async listTasks(params?: {
    project_id?: string;
    issue_id?: string;
    status?: string;
  }): Promise<ListTasksResponse> {
    const search = new URLSearchParams();
    if (params?.project_id) search.set("project_id", params.project_id);
    if (params?.issue_id) search.set("issue_id", params.issue_id);
    if (params?.status) search.set("status", params.status);
    const query = search.toString();
    return this.fetch(`/api/tasks${query ? `?${query}` : ""}`);
  }

  async getTask(id: string): Promise<Task> {
    return this.fetch(`/api/tasks/${id}`);
  }

  async createTask(data: CreateTaskRequest): Promise<Task> {
    return this.fetch("/api/tasks", {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateTask(id: string, data: UpdateTaskRequest): Promise<Task> {
    return this.fetch(`/api/tasks/${id}`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
  }

  async deleteTask(id: string): Promise<void> {
    await this.fetch(`/api/tasks/${id}`, { method: "DELETE" });
  }

  async listKnowledge(params?: {
    query?: string;
    projectId?: string;
    kind?: string;
    limit?: number;
    cursor?: string;
  }): Promise<KnowledgeListResponse> {
    const search = new URLSearchParams();
    if (params?.query) search.set("query", params.query);
    if (params?.projectId) search.set("project_id", params.projectId);
    if (params?.kind) search.set("kind", params.kind);
    if (params?.limit) search.set("limit", String(params.limit));
    if (params?.cursor) search.set("cursor", params.cursor);
    const query = search.toString();
    const raw = await this.fetch<unknown>(
      `/api/knowledge${query ? `?${query}` : ""}`,
    );
    return parseWithFallback(
      raw,
      knowledgeListSchema,
      EMPTY_KNOWLEDGE_LIST,
      { endpoint: "GET /api/knowledge" },
    );
  }

  async getKnowledge(id: string): Promise<KnowledgeEntry> {
    const raw = await this.fetch<unknown>(
      `/api/knowledge/${encodeURIComponent(id)}`,
    );
    return parseWithFallback(
      raw,
      knowledgeEntrySchema,
      {
        id: "",
        workspaceId: "",
        projectId: null,
        candidateId: null,
        kind: "reference",
        status: "published",
        currentRevision: 0,
        revisions: [],
        createdAt: "",
        updatedAt: "",
      },
      { endpoint: "GET /api/knowledge/:id" },
    );
  }

  async proposeKnowledge(
    request: ProposeKnowledgeRequest,
  ): Promise<KnowledgeCandidate> {
    const raw = await this.fetch<unknown>("/api/knowledge/proposals", {
      method: "POST",
      body: JSON.stringify({
        project_id: request.projectId,
        knowledge_id: request.knowledgeId,
        kind: request.kind,
        title: request.title,
        content: request.content,
        reason: request.reason,
        source_refs: request.sourceRefs,
      }),
    });
    return parseWithFallback(
      raw,
      knowledgeCandidateSchema,
      {
        id: "",
        workspaceId: "",
        projectId: null,
        knowledgeId: request.knowledgeId ?? null,
        targetRevision: 0,
        kind: request.kind,
        title: request.title,
        content: request.content,
        reason: request.reason,
        status: "candidate",
        revision: 0,
        proposedBy: "",
        sourceRefs: [],
        createdAt: "",
        updatedAt: "",
      },
      { endpoint: "POST /api/knowledge/proposals" },
    );
  }

  async listKnowledgeCandidates(): Promise<KnowledgeCandidateListResponse> {
    const raw = await this.fetch<unknown>("/api/knowledge/candidates");
    return parseWithFallback(
      raw,
      knowledgeCandidateListSchema,
      EMPTY_KNOWLEDGE_CANDIDATE_LIST,
      { endpoint: "GET /api/knowledge/candidates" },
    );
  }

  async reviewKnowledgeCandidate(
    candidateId: string,
    request: ReviewKnowledgeRequest,
  ): Promise<ReviewKnowledgeResponse> {
    const raw = await this.fetch<unknown>(
      `/api/knowledge/candidates/${encodeURIComponent(candidateId)}/review`,
      {
        method: "POST",
        body: JSON.stringify({
          action: request.action,
          expected_revision: request.expectedRevision,
          rationale: request.rationale,
        }),
      },
    );
    return parseWithFallback(
      raw,
      reviewKnowledgeResponseSchema,
      {
        candidate: {
          id: "",
          workspaceId: "",
          projectId: null,
          knowledgeId: null,
          targetRevision: 0,
          kind: "reference",
          title: "",
          content: "",
          reason: "",
          status: "candidate",
          revision: 0,
          proposedBy: "",
          sourceRefs: [],
          createdAt: "",
          updatedAt: "",
        },
        entry: null,
      },
      { endpoint: "POST /api/knowledge/candidates/:id/review" },
    );
  }

  // Project resources
  async listProjectResources(
    projectId: string,
  ): Promise<ListProjectResourcesResponse> {
    return this.fetch(`/api/projects/${projectId}/resources`);
  }

  async createProjectResource(
    projectId: string,
    data: CreateProjectResourceRequest,
  ): Promise<ProjectResource> {
    return this.fetch(`/api/projects/${projectId}/resources`, {
      method: "POST",
      body: JSON.stringify(data),
    });
  }

  async updateProjectResource(
    projectId: string,
    resourceId: string,
    data: UpdateProjectResourceRequest,
  ): Promise<ProjectResource> {
    return this.fetch(`/api/projects/${projectId}/resources/${resourceId}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }

  async deleteProjectResource(
    projectId: string,
    resourceId: string,
  ): Promise<void> {
    await this.fetch(`/api/projects/${projectId}/resources/${resourceId}`, {
      method: "DELETE",
    });
  }

  // Labels
  async listLabels(resourceType: LabelResourceType = "issue"): Promise<ListLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/labels?resource_type=${resourceType}`);
    return parseWithFallback(raw, ListLabelsResponseSchema, EMPTY_LIST_LABELS_RESPONSE, {
      endpoint: "GET /api/labels",
    });
  }

  async getLabel(id: string): Promise<Label> {
    const raw = await this.fetch<unknown>(`/api/labels/${id}`);
    return parseWithFallback(raw, LabelSchema, EMPTY_LABEL, {
      endpoint: "GET /api/labels/{id}",
    });
  }

  async createLabel(data: CreateLabelRequest): Promise<Label> {
    const raw = await this.fetch<unknown>(`/api/labels`, {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, LabelSchema, EMPTY_LABEL, {
      endpoint: "POST /api/labels",
    });
  }

  async updateLabel(id: string, data: UpdateLabelRequest): Promise<Label> {
    const raw = await this.fetch<unknown>(`/api/labels/${id}`, {
      method: "PUT",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, LabelSchema, EMPTY_LABEL, {
      endpoint: "PUT /api/labels/{id}",
    });
  }

  async deleteLabel(id: string): Promise<void> {
    await this.fetch(`/api/labels/${id}`, { method: "DELETE" });
  }

  // Custom issue properties
  async listProperties(includeArchived = false): Promise<ListPropertiesResponse> {
    const suffix = includeArchived ? "?include_archived=true" : "";
    let raw: unknown;
    try {
      raw = await this.fetch<unknown>(`/api/properties${suffix}`);
    } catch (error) {
      // A backend predating custom properties 404s here (e.g. after a
      // server-only rollback). Treat it as an empty catalog: the property
      // UI sections disappear and the active-catalog reconciliation strips
      // persisted property sorts/filters, so no property params ever reach
      // the old server. Other errors keep normal query-error semantics.
      if (error instanceof Error && "status" in error && (error as { status?: number }).status === 404) {
        return EMPTY_LIST_PROPERTIES_RESPONSE;
      }
      throw error;
    }
    return parseWithFallback(raw, ListPropertiesResponseSchema, EMPTY_LIST_PROPERTIES_RESPONSE, {
      endpoint: "GET /api/properties",
    });
  }

  async createProperty(data: CreatePropertyRequest): Promise<IssueProperty> {
    const raw = await this.fetch<unknown>(`/api/properties`, {
      method: "POST",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, IssuePropertySchema, EMPTY_ISSUE_PROPERTY, {
      endpoint: "POST /api/properties",
    });
  }

  async updateProperty(id: string, data: UpdatePropertyRequest): Promise<IssueProperty> {
    const raw = await this.fetch<unknown>(`/api/properties/${id}`, {
      method: "PATCH",
      body: JSON.stringify(data),
    });
    return parseWithFallback(raw, IssuePropertySchema, EMPTY_ISSUE_PROPERTY, {
      endpoint: "PATCH /api/properties/{id}",
    });
  }

  async setIssueProperty(issueId: string, propertyId: string, value: IssuePropertyValue): Promise<IssuePropertiesResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/properties/${propertyId}`, {
      method: "PUT",
      body: JSON.stringify({ value }),
    });
    return parseWithFallback(raw, IssuePropertiesResponseSchema, EMPTY_ISSUE_PROPERTIES_RESPONSE, {
      endpoint: "PUT /api/issues/{id}/properties/{propertyId}",
    });
  }

  async unsetIssueProperty(issueId: string, propertyId: string): Promise<IssuePropertiesResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/properties/${propertyId}`, {
      method: "DELETE",
    });
    return parseWithFallback(raw, IssuePropertiesResponseSchema, EMPTY_ISSUE_PROPERTIES_RESPONSE, {
      endpoint: "DELETE /api/issues/{id}/properties/{propertyId}",
    });
  }

  async listLabelsForIssue(issueId: string): Promise<IssueLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/labels`);
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "GET /api/issues/{id}/labels",
    });
  }

  async attachLabel(issueId: string, labelId: string): Promise<IssueLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/labels`, {
      method: "POST",
      body: JSON.stringify({ label_id: labelId }),
    });
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "POST /api/issues/{id}/labels",
    });
  }

  async detachLabel(issueId: string, labelId: string): Promise<IssueLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/labels/${labelId}`, {
      method: "DELETE",
    });
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "DELETE /api/issues/{id}/labels/{labelId}",
    });
  }

  async listLabelsForResource(
    _resourceType: "skill",
    resourceId: string,
  ): Promise<ResourceLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/skills/${resourceId}/labels`);
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "GET /api/skills/{id}/labels",
    });
  }

  async attachLabelToResource(
    _resourceType: "skill",
    resourceId: string,
    labelId: string,
  ): Promise<ResourceLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/skills/${resourceId}/labels`, {
      method: "POST",
      body: JSON.stringify({ label_id: labelId }),
    });
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "POST /api/skills/{id}/labels",
    });
  }

  async detachLabelFromResource(
    _resourceType: "skill",
    resourceId: string,
    labelId: string,
  ): Promise<ResourceLabelsResponse> {
    const raw = await this.fetch<unknown>(`/api/skills/${resourceId}/labels/${labelId}`, {
      method: "DELETE",
    });
    return parseWithFallback(raw, ResourceLabelsResponseSchema, EMPTY_RESOURCE_LABELS_RESPONSE, {
      endpoint: "DELETE /api/skills/{id}/labels/{labelId}",
    });
  }

  // Pins
  async listPins(): Promise<PinnedItem[]> {
    const raw = await this.fetch<unknown>("/api/pins");
    return parseWithFallback(raw, PinsSchema, EMPTY_PINS, { endpoint: "GET /api/pins" });
  }

  async createPin(data: CreatePinRequest): Promise<PinnedItem> {
    const raw = await this.fetch<unknown>("/api/pins", {
      method: "POST",
      body: JSON.stringify(data),
    });
    const parsed = PinSchema.safeParse(raw);
    if (!parsed.success) throw new Error("Invalid pin response");
    return parsed.data;
  }

  async deletePin(itemType: PinnedItemType, itemId: string): Promise<void> {
    await this.fetch(`/api/pins/${itemType}/${itemId}`, { method: "DELETE" });
  }

  async reorderPins(data: ReorderPinsRequest): Promise<void> {
    await this.fetch("/api/pins/reorder", {
      method: "PUT",
      body: JSON.stringify(data),
    });
  }


  // GitHub integration
  async getGitHubConnectURL(
    workspaceId: string,
    returnTo?: "github" | "repositories",
  ): Promise<GitHubConnectResponse> {
    const search = new URLSearchParams();
    if (returnTo) search.set("return_to", returnTo);
    const suffix = search.size > 0 ? `?${search.toString()}` : "";
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/github/connect${suffix}`,
    );
    return parseWithFallback(
      raw,
      GitHubConnectResponseSchema,
      EMPTY_GITHUB_CONNECT_RESPONSE,
      { endpoint: "GET /api/workspaces/:id/github/connect" },
    );
  }

  async listGitHubInstallations(workspaceId: string): Promise<ListGitHubInstallationsResponse> {
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/github/installations`,
    );
    return parseWithFallback(
      raw,
      ListGitHubInstallationsResponseSchema,
      EMPTY_LIST_GITHUB_INSTALLATIONS_RESPONSE,
      { endpoint: "GET /api/workspaces/:id/github/installations" },
    );
  }

  async listGitHubInstallationRepositories(
    workspaceId: string,
    installationId: string,
    params: { page?: number; per_page?: number } = {},
  ): Promise<ListGitHubRepositoriesResponse> {
    const search = new URLSearchParams();
    if (params.page !== undefined) search.set("page", String(params.page));
    if (params.per_page !== undefined) search.set("per_page", String(params.per_page));
    const suffix = search.size > 0 ? `?${search.toString()}` : "";
    const raw = await this.fetch<unknown>(
      `/api/workspaces/${workspaceId}/github/installations/${installationId}/repositories${suffix}`,
    );
    return parseWithFallback(
      raw,
      ListGitHubRepositoriesResponseSchema,
      EMPTY_LIST_GITHUB_REPOSITORIES_RESPONSE,
      { endpoint: "GET /api/workspaces/:id/github/installations/:installationId/repositories" },
    );
  }

  async deleteGitHubInstallation(workspaceId: string, installationId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}/github/installations/${installationId}`, {
      method: "DELETE",
    });
  }

  async listIssuePullRequests(issueId: string): Promise<{ pull_requests: GitHubPullRequest[] }> {
    const raw = await this.fetch<unknown>(`/api/issues/${issueId}/pull-requests`);
    return parseWithFallback(
      raw,
      IssuePullRequestsResponseSchema,
      EMPTY_ISSUE_PULL_REQUESTS_RESPONSE,
      { endpoint: "GET /api/issues/:id/pull-requests" },
    );
  }

  // VCS integration (Forgejo / Gitea / GitLab)
  async listVCSConnections(workspaceId: string): Promise<ListVCSConnectionsResponse> {
    return this.fetch(`/api/workspaces/${workspaceId}/vcs/connections`);
  }

  async connectVCS(
    workspaceId: string,
    body: ConnectVCSRequest,
  ): Promise<ConnectVCSResponse> {
    return this.fetch(`/api/workspaces/${workspaceId}/vcs/connections`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  }

  async deleteVCSConnection(workspaceId: string, connectionId: string): Promise<void> {
    await this.fetch(`/api/workspaces/${workspaceId}/vcs/connections/${connectionId}`, {
      method: "DELETE",
    });
  }

  async rotateVCSWebhook(
    workspaceId: string,
    connectionId: string,
  ): Promise<ConnectVCSResponse> {
    return this.fetch(
      `/api/workspaces/${workspaceId}/vcs/connections/${connectionId}/rotate-webhook`,
      { method: "POST" },
    );
  }

}
