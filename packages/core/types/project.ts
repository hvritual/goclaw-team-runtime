export type ProjectStatus = "planned" | "in_progress" | "paused" | "completed" | "cancelled";

export type ProjectPriority = "urgent" | "high" | "medium" | "low" | "none";

export interface Project {
  id: string;
  workspace_id: string;
  title: string;
  description: string | null;
  icon: string | null;
  status: ProjectStatus;
  priority: ProjectPriority;
  lead_type: "member" | null;
  lead_id: string | null;
  // Calendar days ("YYYY-MM-DD"), no time-of-day or timezone — same contract as
  // issue.start_date / issue.due_date.
  start_date: string | null;
  due_date: string | null;
  created_at: string;
  updated_at: string;
  issue_count: number;
  done_count: number;
  resource_count: number;
}

export interface CreateProjectRequest {
  title: string;
  description?: string;
  icon?: string;
  status?: ProjectStatus;
  priority?: ProjectPriority;
  lead_type?: "member";
  lead_id?: string;
  start_date?: string;
  due_date?: string;
  // Resources to attach in the same transaction as the project. Server returns
  // 4xx (and rolls back) if any one is invalid or duplicate.
  resources?: CreateProjectResourceRequest[];
}

export interface UpdateProjectRequest {
  title?: string;
  description?: string | null;
  icon?: string | null;
  status?: ProjectStatus;
  priority?: ProjectPriority;
  lead_type?: "member" | null;
  lead_id?: string | null;
  // Omit the key to leave the date untouched; send null (or "") to clear it.
  start_date?: string | null;
  due_date?: string | null;
}

export interface ListProjectsResponse {
  projects: Project[];
  total: number;
}

// ProjectResource is a typed pointer from a project to an external resource.
// The resource_ref shape depends on resource_type. New types add a case in
// validateAndNormalizeResourceRef on the server and a renderer in the UI.
//
// Known types (UI must default-case unknown server-side additions).
export type KnownProjectResourceType = "github_repo" | "url";
export type ProjectResourceType = KnownProjectResourceType | (string & {});

export interface GithubRepoResourceRef extends Record<string, unknown> {
  url: string;
  ref?: string;
}

export interface UrlResourceRef extends Record<string, unknown> {
  url: string;
}

export type ProjectResourceRef = Record<string, unknown>;

export interface ProjectResourceConnection {
  state: "unchecked" | "available" | "degraded" | "unavailable";
  diagnostic_code?: string;
  checked_at?: string;
}

export interface ProjectResource {
  id: string;
  project_id: string;
  workspace_id: string;
  resource_type: ProjectResourceType;
  resource_ref: ProjectResourceRef;
  label?: string;
  position: number;
  status: "active" | "archived";
  revision: number;
  connection: ProjectResourceConnection;
  created_at: string;
  created_by: string;
  updated_at: string;
  updated_by: string;
  archived_at?: string;
  archived_by?: string;
}

export interface CreateProjectResourceRequest {
  resource_type: KnownProjectResourceType;
  resource_ref: GithubRepoResourceRef | UrlResourceRef;
  label?: string;
}

export type UpdateProjectResourceRequest =
  | {
      action: "update";
      expected_revision: number;
      resource_ref?: GithubRepoResourceRef | UrlResourceRef;
      label?: string;
    }
  | {
      action: "reorder";
      expected_revision: number;
      before_resource_id?: string;
    }
  | {
      action: "restore" | "refresh";
      expected_revision: number;
    };

export interface ListProjectResourcesResponse {
  resources: ProjectResource[];
  total: number;
  revision: number;
}
