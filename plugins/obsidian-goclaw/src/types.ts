export type ConnectionState = "disconnected" | "connecting" | "connected" | "error";

export interface GoClawSettings {
  gatewayUrl: string;
  projectId: string;
  topicId: string;
  autoConnect: boolean;
  secretKey: string;
  userSecretKey: string;
  reviewerId: string;
  reviewerSecretKey: string;
}

export const DEFAULT_SETTINGS: GoClawSettings = {
  gatewayUrl: "ws://127.0.0.1:28789/ws",
  projectId: "default",
  topicId: "inbox",
  autoConnect: true,
  secretKey: "goclaw.gateway.token",
  userSecretKey: "goclaw.team.user-token",
  reviewerId: "obsidian-user",
  reviewerSecretKey: "goclaw.governance.reviewer-token"
};

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
  pending?: boolean;
  error?: boolean;
}

export interface KnowledgeProposal {
  id: string;
  project_id: string;
  target_path: string;
  base_sha256?: string;
  proposed_content: string;
  reason: string;
  evidence_trace_id?: string;
  status: "pending" | "approved" | "rejected";
  created_by: string;
  created_at: string;
  reviewed_by?: string;
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
  collection: string;
  work_id: string;
  expression_id: string;
  manifestation_id: string;
  item_id: string;
  title: string;
  abstract?: string;
  content: string;
  kind: string;
  status: CatalogMemoryStatus;
  subjects?: string[];
  facets?: Record<string, string[]>;
  authority_ids?: string[];
  relations?: Array<{ type: string; target_id: string; note?: string }>;
  provenance: {
    source_uri: string;
    source_kind?: string;
    source_revision?: string;
    source_sha256?: string;
    trace_id?: string;
  };
  evidence_refs?: string[];
  confidence: number;
  review_at?: string;
  expires_at?: string;
  version: number;
  checksum: string;
  created_by: string;
  created_at: string;
  reviewed_by?: string;
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

export interface ChangeManifest {
  target_components: string[];
  evidence_trace_ids?: string[];
  root_cause: string;
  change_summary: string;
  expected_fix_tags?: string[];
  possible_regressions?: string[];
}

export interface HarnessExperiment {
  id: string;
  base_version: string;
  candidate_version: string;
  candidate_path: string;
  status: "draft" | "validated" | "human_approved" | "active" | "rejected" | "rolled_back";
  change_manifest: ChangeManifest;
  validation_report?: string;
  created_at: string;
  reviewed_by?: string;
}

export interface HarnessStatus {
  active: {
    version: string;
    previous_version?: string;
    activated_at: string;
    activated_by: string;
    experiment_id?: string;
    decision?: {
      reviewer_id: string;
      decision: string;
      rationale: string;
      created_at: string;
    };
  };
  manifest: {
    version: string;
    name: string;
    description?: string;
    model_profile?: string;
    minimum_golden: number;
    minimum_holdout: number;
    components: Record<string, string>;
  };
  project: string;
}

export interface HarnessTrace {
  id: string;
  project_id: string;
  topic_id: string;
  status: string;
  input?: string;
  output?: string;
  error?: string;
  started_at: string;
  duration_ms: number;
  harness_version: string;
  tool_calls?: Array<{ name: string }>;
}

export interface ProjectScopeParams {
  project_id: string;
}

export interface ItemEnvelope<T> {
  items: T[];
  total?: number;
}

export type TeamMemberStatus = "active" | "away" | "offline" | "disabled";

export interface TeamMember {
  id: string;
  display_name: string;
  role?: string;
  status: TeamMemberStatus;
  business_domains?: string[];
  project_ids?: string[];
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

export type WorkItemStatus =
  | "backlog"
  | "ready"
  | "in_progress"
  | "blocked"
  | "in_review"
  | "done"
  | "cancelled";

export interface TeamWorkItem {
  id: string;
  project_id: string;
  title: string;
  kind?: "feature" | "task" | "bug" | "document" | "research";
  status: WorkItemStatus;
  priority?: "critical" | "high" | "medium" | "low";
  assignee_id?: string;
  business_domain?: string;
  task_id?: string;
  issue_id?: string;
  source_refs?: string[];
  blocked_reason?: string;
  updated_at: string;
}

export type IssueStatus =
  | "open"
  | "triaged"
  | "in_progress"
  | "in_review"
  | "resolved"
  | "closed"
  | "reopened";

export type IssueSeverity = "critical" | "high" | "medium" | "low";

export interface TeamIssue {
  id: string;
  project_id: string;
  title: string;
  status: IssueStatus;
  severity: IssueSeverity;
  priority?: "urgent" | "high" | "medium" | "low";
  owner_id?: string;
  work_item_id?: string;
  external_url?: string;
  regression_case_id?: string;
  updated_at: string;
}

export type RunnerStatus = "online" | "busy" | "draining" | "offline";

export interface TeamRunner {
  id: string;
  member_id?: string;
  display_name?: string;
  status: RunnerStatus;
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
  kind?: "go_module" | "ui_component" | "api" | "schema" | "template" | string;
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

export interface TeamControlRPC {
  "team.members": {
    params: ProjectScopeParams;
    result: TeamMember[] | ItemEnvelope<TeamMember>;
  };
  "work.items": {
    params: ProjectScopeParams & { limit?: number };
    result: TeamWorkItem[] | ItemEnvelope<TeamWorkItem>;
  };
  "issue.list": {
    params: ProjectScopeParams & { limit?: number };
    result: TeamIssue[] | ItemEnvelope<TeamIssue>;
  };
  "runner.list": {
    params: ProjectScopeParams;
    result: TeamRunner[] | ItemEnvelope<TeamRunner>;
  };
  "policy.status": {
    params: ProjectScopeParams;
    result: TeamPolicyStatus;
  };
  "docs.summary": {
    params: ProjectScopeParams & { limit?: number };
    result: TeamDocsSummary;
  };
  "components.summary": {
    params: ProjectScopeParams & { limit?: number };
    result: TeamComponentsSummary;
  };
}

export type DevReviewKind = "scenario" | "capacity" | "risk" | "cost";

export interface DevReviewRecord {
  kind: DevReviewKind;
  decision: "pending" | "approved" | "rejected";
  reviewer?: string;
  comment?: string;
  decided_at?: string;
}

export interface DevDoneGateResult {
  passed: boolean;
  verdict: string;
  reasons?: string[];
  evidence_path: string;
  evidence_sha256?: string;
  evaluated_at: string;
}

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
  request: { raw_request: string };
  goal: { objective: string; non_goals?: string[]; success_tests?: string[] };
  scope: {
    allowed_paths?: string[];
    denied_paths?: string[];
    max_changed_files: number;
    max_changed_lines: number;
  };
  reviews: Record<DevReviewKind, DevReviewRecord>;
  compile: {
    revision: number;
    base_ref: string;
    base_commit?: string;
    execution_bundle_hash?: string;
  };
  branch?: string;
  worktree_path?: string;
  repair_count: number;
  last_gate?: DevDoneGateResult;
  last_evidence?: string;
  created_at: string;
  updated_at: string;
}

export type OuroborosStatus =
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

export interface OuroborosQuestion {
  id: string;
  dimension: "goal" | "constraint" | "success" | "context";
  text: string;
  why?: string;
  blocking: boolean;
}

export interface OuroborosAssessment {
  round: number;
  overall: number;
  threshold: number;
  ready: boolean;
  ready_streak: number;
  required_ready_streak: number;
  summary: string;
  unresolved?: string[];
  human_decision_required?: boolean;
  score_spread?: number;
  distinct_models?: number;
  gray_zone?: boolean;
  calibration_version?: string;
}

export interface OuroborosRound {
  number: number;
  questions: OuroborosQuestion[];
  answers?: Array<{ question_id: string; text: string }>;
  assessment: OuroborosAssessment;
  created_at: string;
}

export interface OuroborosSeedReference {
  hash: string;
  id: string;
  generation: number;
  parent_hash?: string;
  approved: boolean;
  approved_by?: string;
  comment?: string;
  created_at: string;
}

export interface OuroborosEvaluation {
  id: string;
  task_id: string;
  passed: boolean;
  mechanical: { passed: boolean; score: number; summary: string };
  semantic: { passed: boolean; score: number; summary: string };
  consensus: { passed: boolean; score: number; summary: string };
  score_spread?: number;
  distinct_models?: number;
  human_decision_required?: boolean;
  human_disposition?: {
    accepted: boolean;
    decision: {
      reviewer_id: string;
      decision: string;
      rationale: string;
      counterargument?: string;
      evidence_refs?: string[];
      created_at: string;
    };
  };
  created_at: string;
}

export interface OuroborosEvolution {
  id: string;
  candidate_seed_hash?: string;
  candidate_generation?: number;
  ontology_similarity: number;
  convergence_threshold: number;
  status: "pending" | "approved" | "rejected" | "converged" | "blocked";
  action: string;
  reasons?: string[];
  knowledge_gaps?: string[];
  possible_regressions?: string[];
  oscillation_detected?: boolean;
  hard_cap_reached?: boolean;
  created_at: string;
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
  status: OuroborosStatus;
  rounds: OuroborosRound[];
  seed_history?: OuroborosSeedReference[];
  active_seed_hash?: string;
  pending_seed_hash?: string;
  compiled_tasks?: Array<{
    seed_hash: string;
    generation: number;
    task_id: string;
    compiled_at: string;
  }>;
  evaluations?: OuroborosEvaluation[];
  pending_evolution?: OuroborosEvolution;
  last_evolution?: OuroborosEvolution;
  last_error?: string;
  blocked_reasons?: string[];
  decision_conflicts?: Array<{
    id: string;
    description: string;
    claim_ids?: string[];
    status: "open" | "resolved";
    resolution?: string;
  }>;
  outcomes?: Array<{
    id: string;
    supersedes_id?: string;
    kind: "passed" | "failed" | "cancelled" | "rolled_back" | "no_feedback";
    evaluation_id?: string;
    reason: string;
    created_at: string;
  }>;
  reference_class?: {
    total: number;
    passed: number;
    failed: number;
    cancelled: number;
    rolled_back: number;
    no_feedback: number;
    pass_rate: number;
    failure_rate: number;
  };
  created_at: string;
  updated_at: string;
}

export interface OuroborosSeed {
  id: string;
  session_id: string;
  generation: number;
  title: string;
  goal: string;
  constraints: string[];
  acceptance_criteria: Array<{
    id: string;
    description: string;
    verify_command?: string[];
  }>;
  plan: {
    summary: string;
    milestones: Array<{
      id: string;
      title: string;
      work_items: Array<{
        id: string;
        title: string;
        instructions: string;
        criteria_ids?: string[];
      }>;
    }>;
  };
  scope: {
    allowed_paths: string[];
    denied_paths?: string[];
    max_changed_files: number;
    max_changed_lines: number;
    allow_new_dependency: boolean;
  };
  risk: {
    level: "low" | "medium" | "high";
    forbidden?: string[];
    rollback: string;
    human_escalates?: string[];
  };
  cost: {
    max_repair_attempts: number;
    max_input_tokens?: number;
    max_output_tokens?: number;
  };
  alternatives: Array<{
    id: string;
    title: string;
    summary: string;
    tradeoffs?: string[];
    selected: boolean;
  }>;
  falsifiers: Array<{
    criterion_id: string;
    condition: string;
    evidence_required: string;
  }>;
  cost_of_inaction: string[];
  kill_conditions: Array<{
    id: string;
    condition: string;
    metric: string;
    threshold: string;
    action: string;
  }>;
  pre_mortem: string[];
  predictions: Array<{
    id: string;
    claim: string;
    expected_outcome: string;
    horizon: string;
    confidence: number;
  }>;
  ambiguity_score: number;
  hash: string;
}
