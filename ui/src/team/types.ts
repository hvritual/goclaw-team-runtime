export type PageID =
  | "overview"
  | "chat"
  | "spec"
  | "work"
  | "quality"
  | "reviews"
  | "memory"
  | "approvals"
  | "development"
  | "team"
  | "progress"
  | "harness";

export interface WebSession {
  principal_id: string;
  csrf_token: string;
  expires_at: string;
}

export interface LoginInput {
  gatewayToken: string;
  userToken: string;
}

export interface RPCErrorShape {
  code: number;
  message: string;
  data?: unknown;
}

export interface RPCMessage {
  jsonrpc?: string;
  id?: string;
  method?: string;
  params?: { data?: unknown; [key: string]: unknown };
  result?: unknown;
  error?: RPCErrorShape;
}

export interface ChatEvent {
  run_id: string;
  seq: number;
  state: "delta" | "thinking" | "tool" | "final" | "error";
  content?: string;
  project_id?: string;
  topic_id?: string;
  timestamp?: string;
}

export interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  content: string;
  timestamp?: string;
  pending?: boolean;
  error?: boolean;
  transient?: boolean;
}

export interface ChatHistory {
  project_id: string;
  topic_id: string;
  messages: Array<ChatMessage & { role: "user" | "assistant" }>;
  has_more: boolean;
}

export interface ItemEnvelope<T> {
  items: T[];
  total?: number;
}

export function collectionItems<T>(
  value: T[] | ItemEnvelope<T> | null | undefined,
): T[] {
  if (!value) return [];
  return Array.isArray(value) ? value : (value.items ?? []);
}

export interface TeamProject {
  id: string;
  team_id: string;
  key: string;
  name: string;
  description?: string;
  status: "active" | "archived";
  updated_at: string;
}

export interface TeamMember {
  id: string;
  display_name: string;
  role?: string;
  status: "active" | "away" | "offline" | "disabled";
  business_domains?: string[];
  capacity?: {
    planned_points?: number;
    active_work?: number;
    queued_work?: number;
    blocked_work?: number;
    utilization_percent?: number;
  };
  runner_ids?: string[];
  last_seen_at?: string;
}

export interface TeamWorkItem {
  id: string;
  project_id: string;
  title: string;
  issue_id?: string;
  instructions?: string;
  status:
    | "pending"
    | "ready"
    | "in_progress"
    | "blocked"
    | "verifying"
    | "done"
    | "cancelled";
  priority?: "p0" | "p1" | "p2" | "p3" | "p4";
  assignee_id?: string;
  business_domain?: string;
  estimate_points?: number;
  due_at?: string;
  depends_on?: string[];
  component_ids?: string[];
  verification_commands?: string[][];
  task_id?: string;
  blocked_reason?: string;
  updated_at: string;
}

export interface TeamIssue {
  id: string;
  project_id: string;
  title: string;
  type: "bug" | "task" | "improvement" | "risk";
  description?: string;
  status:
    | "new"
    | "triaged"
    | "assigned"
    | "in_progress"
    | "blocked"
    | "verifying"
    | "resolved"
    | "closed"
    | "reopened"
    | "cancelled";
  severity: "critical" | "high" | "medium" | "low";
  priority?: "p0" | "p1" | "p2" | "p3" | "p4";
  reporter_id?: string;
  module?: string;
  environment?: string;
  labels?: string[];
  component_ids?: string[];
  reproduction?: string;
  expected?: string;
  actual?: string;
  external_issue_id?: string;
  due_at?: string;
  sla_minutes?: number;
  sla_deadline?: string;
  duplicate_of?: string;
  regression_of?: string;
  introduced_by_commit?: string;
  fixed_by_commit?: string;
  release_id?: string;
  reopen_count?: number;
  resolution?: string;
  owner_id?: string;
  work_item_id?: string;
  regression_case_id?: string;
  updated_at: string;
}

export interface TeamRepository {
  id: string;
  project_id: string;
  name: string;
  remote_url?: string;
  local_path?: string;
  default_branch: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TeamArtifact {
  id: string;
  project_id: string;
  resource_type: ResourceType;
  kind:
    | "diff"
    | "evidence"
    | "build"
    | "log"
    | "report"
    | "trace"
    | "commit"
    | "pull_request"
    | "package"
    | "other";
  name: string;
  uri: string;
  sha256?: string;
  content_type?: string;
  metadata?: Record<string, string>;
  created_by: string;
  created_at: string;
}

export type ResourceType =
  | "issue"
  | "task"
  | "work_item"
  | "run"
  | "trace"
  | "commit"
  | "pull_request"
  | "ci"
  | "release"
  | "regression_case"
  | "spec"
  | "artifact"
  | "document"
  | "component"
  | "repository"
  | "policy";

export interface TeamCorrelation {
  id: string;
  project_id: string;
  source_type: ResourceType;
  source_id: string;
  target_type: ResourceType;
  target_id: string;
  relation: string;
  metadata?: Record<string, string>;
  created_by: string;
  created_at: string;
}

export interface TeamDocumentRecord {
  id: string;
  project_id: string;
  key: string;
  title: string;
  kind:
    | "prd"
    | "adr"
    | "design"
    | "runbook"
    | "api"
    | "test_plan"
    | "report"
    | "knowledge"
    | "other";
  status: "draft" | "active" | "superseded" | "archived";
  uri: string;
  revision?: string;
  sha256?: string;
  owner_id?: string;
  supersedes?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TeamComponentRecord {
  id: string;
  project_id: string;
  repository_id?: string;
  name: string;
  kind: "service" | "library" | "app" | "module" | "device" | "other";
  root_path?: string;
  description?: string;
  owner_ids?: string[];
  dependency_ids?: string[];
  metadata?: Record<string, string>;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TeamPolicyBundle {
  id: string;
  name: string;
  scope: "team" | "project" | "repository" | "component";
  scope_id: string;
  team_id: string;
  project_id?: string;
  version: number;
  priority: number;
  enabled: boolean;
  rules: Record<string, unknown>;
  hash: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface TeamRunnerTask {
  id: string;
  project_id: string;
  status: string;
  runner_id?: string;
  assignee_id?: string;
  attempt?: number;
  lease_expires_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface TeamAssignment {
  id: string;
  project_id: string;
  target_type: "issue" | "work_item";
  target_id: string;
  user_id: string;
  role: "owner" | "contributor" | "reviewer";
  status: "active" | "released";
  updated_at: string;
}

export interface TeamRunner {
  id: string;
  member_id?: string;
  display_name?: string;
  status: "online" | "busy" | "draining" | "offline";
  capabilities?: string[];
  current_work_id?: string;
  lease?: {
    id: string;
    work_id?: string;
    acquired_at?: string;
    renewed_at?: string;
    expires_at: string;
  };
  last_seen_at?: string;
}

export interface TeamPolicyStatus {
  project_id: string;
  effective_version?: string;
  compliant: boolean;
  drift_count: number;
  checked_at?: string;
  layers?: Array<{
    scope: "global" | "domain" | "project";
    id: string;
    version: string;
    checksum?: string;
    compliant?: boolean;
  }>;
  violations?: Array<{
    code: string;
    message: string;
    severity?: "error" | "warning";
  }>;
}

export interface TeamDocument {
  id: string;
  title: string;
  path: string;
  kind?: string;
  owner_id?: string;
  status?: "draft" | "review" | "approved" | "stale";
  linked_work_ids?: string[];
  review_at?: string;
  updated_at?: string;
}

export interface TeamDocsSummary {
  project_id: string;
  total: number;
  approved: number;
  review_due: number;
  stale: number;
  unlinked: number;
  items?: TeamDocument[];
}

export interface TeamComponent {
  id: string;
  name: string;
  kind?: string;
  version?: string;
  owner_id?: string;
  status?: "active" | "deprecated" | "experimental";
  reuse_count?: number;
  updated_at?: string;
}

export interface TeamComponentsSummary {
  project_id: string;
  total: number;
  reusable: number;
  deprecated: number;
  pending_review: number;
  items?: TeamComponent[];
}

export interface KnowledgeProposal {
  id: string;
  project_id: string;
  target_path: string;
  base_sha256?: string;
  base_revision?: string;
  proposed_content: string;
  reason: string;
  evidence_trace_id?: string;
  status: "pending" | "approved" | "rejected";
  created_by: string;
  created_at: string;
}

export type CatalogMemoryStatus =
  | "pending"
  | "active"
  | "rejected"
  | "superseded"
  | "withdrawn"
  | "quarantined";

export interface CatalogMemoryRecord {
  id: string;
  project_id: string;
  title: string;
  abstract?: string;
  content: string;
  kind: string;
  status: CatalogMemoryStatus;
  subjects?: string[];
  provenance: {
    source_uri: string;
    source_kind?: string;
    source_revision?: string;
    source_sha256?: string;
    trace_id?: string;
  };
  confidence: number;
  review_at?: string;
  expires_at?: string;
  version: number;
  created_by: string;
  created_at: string;
}

export interface CatalogMemorySearchResult {
  record: CatalogMemoryRecord;
  score: number;
  matched_by?: string[];
  citation: string;
  warnings?: string[];
  review_due: boolean;
  expired: boolean;
}

export interface CatalogMemoryStats {
  total_records: number;
  by_status: Partial<Record<CatalogMemoryStatus, number>>;
  review_due: number;
  expired: number;
  authorities: number;
  unresolved_contradictions: number;
  usage_last_30_days: number;
}

export interface HarnessExperiment {
  id: string;
  base_version: string;
  candidate_version: string;
  status:
    | "draft"
    | "validated"
    | "human_approved"
    | "active"
    | "rejected"
    | "rolled_back";
  change_manifest: {
    target_components: string[];
    root_cause: string;
    change_summary: string;
  };
  created_at: string;
}

export interface HarnessStatus {
  active: {
    version: string;
    previous_version?: string;
    activated_at: string;
    activated_by: string;
    experiment_id?: string;
  };
  manifest: {
    version: string;
    name: string;
    description?: string;
    model_profile?: string;
    components: Record<string, string>;
  };
  project: string;
}

export interface HarnessTrace {
  id: string;
  project_id: string;
  topic_id: string;
  status: string;
  task_id?: string;
  work_item_id?: string;
  input?: string;
  output?: string;
  error?: string;
  started_at: string;
  duration_ms: number;
  harness_version: string;
  tool_calls?: Array<{ name: string }>;
}

export type DevReviewKind = "scenario" | "capacity" | "risk" | "cost";

export interface DevTask {
  id: string;
  project_id: string;
  title: string;
  status:
    | "review_pending"
    | "ready_to_freeze"
    | "blocked"
    | "frozen"
    | "running"
    | "checking"
    | "repair_pending"
    | "awaiting_acceptance"
    | "done"
    | "failed"
    | "cancelled";
  goal: { objective: string; non_goals?: string[]; success_tests?: string[] };
  scope: {
    allowed_paths?: string[];
    denied_paths?: string[];
    max_changed_files: number;
    max_changed_lines: number;
  };
  reviews: Record<
    DevReviewKind,
    {
      kind: DevReviewKind;
      decision: "pending" | "approved" | "rejected";
      reviewer?: string;
      comment?: string;
    }
  >;
  compile: {
    revision: number;
    base_ref: string;
    base_commit?: string;
    execution_bundle_hash?: string;
  };
  branch?: string;
  repair_count: number;
  last_gate?: {
    passed: boolean;
    verdict: string;
    reasons?: string[];
    evidence_path: string;
    evidence_sha256?: string;
    evaluated_at: string;
  };
  last_evidence?: string;
  created_at: string;
  updated_at: string;
}

export interface OuroborosQuestion {
  id: string;
  dimension: "goal" | "constraint" | "success" | "context";
  text: string;
  why?: string;
  blocking: boolean;
}

export interface OuroborosSession {
  id: string;
  project_id: string;
  topic_id?: string;
  title: string;
  repo_path: string;
  base_ref: string;
  raw_request: string;
  brownfield: boolean;
  status:
    | "interviewing"
    | "clarification_required"
    | "seed_ready"
    | "awaiting_seed_approval"
    | "approved"
    | "compiled"
    | "evaluated"
    | "evolution_pending"
    | "converged"
    | "blocked"
    | "rejected"
    | "cancelled";
  rounds: Array<{
    number: number;
    questions: OuroborosQuestion[];
    answers?: Array<{ question_id: string; text: string }>;
    assessment: {
      overall: number;
      threshold: number;
      ready: boolean;
      ready_streak: number;
      required_ready_streak: number;
      summary: string;
      unresolved?: string[];
    };
    created_at: string;
  }>;
  pending_seed_hash?: string;
  active_seed_hash?: string;
  compiled_tasks?: Array<{
    seed_hash: string;
    generation: number;
    task_id: string;
    compiled_at: string;
  }>;
  last_error?: string;
  created_at: string;
  updated_at: string;
}

export interface GovernanceInput {
  reviewerToken: string;
  rationale: string;
  counterargument: string;
  evidenceRefs: string[];
}
