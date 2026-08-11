package controlplane

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

var RequirementReviewPolicies = []string{"review.scenario", "review.capacity", "review.risk", "review.cost"}

type RequirementData struct {
	Request      string `json:"request"`
	Intent       string `json:"intent,omitempty"`
	SolutionADR  string `json:"solution_adr,omitempty"`
	ChangeReason string `json:"change_reason,omitempty"`
}

type QualityData struct {
	Summary      string    `json:"summary"`
	Severity     string    `json:"severity,omitempty"`
	Reproduction string    `json:"reproduction,omitempty"`
	Probability  int       `json:"probability,omitempty"`
	Impact       int       `json:"impact,omitempty"`
	ResponsePlan string    `json:"response_plan,omitempty"`
	ReviewDueAt  time.Time `json:"review_due_at,omitempty"`
}

type ReviewFindingData struct {
	RuleID       string `json:"rule_id"`
	Summary      string `json:"summary"`
	ModelFinding bool   `json:"model_finding"`
}

type KnowledgeData struct {
	Title       string   `json:"title"`
	SourceIDs   []string `json:"source_ids"`
	EvidenceIDs []string `json:"evidence_ids"`
	DedupKey    string   `json:"dedup_key"`
	Version     int64    `json:"version"`
}

type RunData struct {
	Attempt      int       `json:"attempt"`
	MaxAttempts  int       `json:"max_attempts"`
	LeaseOwner   string    `json:"lease_owner,omitempty"`
	LeaseUntil   time.Time `json:"lease_until,omitempty"`
	WorkspaceRef string    `json:"workspace_ref"`
	SecretRefs   []string  `json:"secret_refs,omitempty"`
}

type P2Flows struct{ kernel *DeliveryKernel }

func NewP2Flows(kernel *DeliveryKernel) (*P2Flows, error) {
	if kernel == nil {
		return nil, invalid("new P2 flows", "kernel", "is required")
	}
	return &P2Flows{kernel: kernel}, nil
}

func (f *P2Flows) StartRequirement(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, request string) (AppendResult, error) {
	if strings.TrimSpace(request) == "" {
		return AppendResult{}, invalid("start requirement", "request", "is required")
	}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "requirement", Revision: 1, State: "clarifying", CreatorID: actor.ID}, RequirementData{Request: strings.TrimSpace(request)})
}

func (f *P2Flows) FinalizeIntent(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, intent string) (AppendResult, error) {
	return f.updateRequirement(ctx, actor, commandID, projectID, head, id, "solution", func(data *RequirementData) error {
		if strings.TrimSpace(intent) == "" {
			return invalid("finalize intent", "intent", "is required")
		}
		data.Intent = strings.TrimSpace(intent)
		return nil
	})
}

func (f *P2Flows) ProposeSolution(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, adr string) (AppendResult, error) {
	return f.updateRequirement(ctx, actor, commandID, projectID, head, id, "review", func(data *RequirementData) error {
		if strings.TrimSpace(adr) == "" {
			return invalid("propose solution", "solution_adr", "is required")
		}
		data.SolutionADR = strings.TrimSpace(adr)
		return nil
	})
}

func (f *P2Flows) FreezeRequirement(ctx context.Context, actor Actor, commandID, projectID, id string, revision, head int64) (AppendResult, error) {
	return f.kernel.AcceptDone(ctx, actor, commandID, projectID, id, revision, head, RequirementReviewPolicies)
}

func (f *P2Flows) ChangeIntent(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, reason string) (AppendResult, error) {
	return f.updateRequirement(ctx, actor, commandID, projectID, head, id, "clarifying", func(data *RequirementData) error {
		if strings.TrimSpace(reason) == "" {
			return invalid("change intent", "reason", "is required")
		}
		data.ChangeReason = strings.TrimSpace(reason)
		data.Intent = ""
		data.SolutionADR = ""
		return nil
	})
}

func (f *P2Flows) CreateRequirementTask(ctx context.Context, actor Actor, nodeCommandID, edgeCommandID, projectID string, head int64, requirementID, taskID, assigneeID string) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	requirement, ok := projection.Nodes[requirementID]
	if !ok || requirement.Kind != "requirement" || requirement.State != "done" {
		return AppendResult{}, invariant("create requirement task", "requirement must be frozen")
	}
	if _, err := f.upsert(ctx, actor, nodeCommandID, projectID, head, WorkNode{ID: taskID, Kind: "task", Revision: 1, State: "draft", CreatorID: actor.ID, AssigneeIDs: []string{assigneeID}}, map[string]string{"requirement_id": requirementID}); err != nil {
		return AppendResult{}, err
	}
	return f.kernel.AddEdge(ctx, actor, edgeCommandID, projectID, head+1, WorkEdge{ID: "trace-" + requirementID + "-" + taskID, From: requirementID, To: taskID, Kind: "trace"})
}

func (f *P2Flows) CreateDefect(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string, data QualityData) (AppendResult, error) {
	if data.Summary == "" || data.Severity == "" || data.Reproduction == "" {
		return AppendResult{}, invalid("create defect", "defect", "summary, severity, and reproduction are required")
	}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "defect", Revision: 1, State: "open", CreatorID: actor.ID}, data)
}

func (f *P2Flows) CreateRisk(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string, data QualityData) (AppendResult, error) {
	if data.Summary == "" || data.Probability < 1 || data.Probability > 5 || data.Impact < 1 || data.Impact > 5 || data.ResponsePlan == "" || data.ReviewDueAt.IsZero() {
		return AppendResult{}, invalid("create risk", "risk", "summary, probability, impact, response, and review due date are required")
	}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "risk", Revision: 1, State: "open", CreatorID: actor.ID}, data)
}

func (f *P2Flows) CloseQualityItem(ctx context.Context, actor Actor, commandID, projectID, id string, revision, head int64) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok {
		return AppendResult{}, notFound("close quality item", id)
	}
	policy := "quality.defect.verify"
	if node.Kind == "risk" {
		var data QualityData
		if err := json.Unmarshal(node.Data, &data); err != nil {
			return AppendResult{}, invariant("close quality item", "invalid risk data")
		}
		if !data.ReviewDueAt.IsZero() && f.kernel.now().After(data.ReviewDueAt) {
			return AppendResult{}, invariant("close quality item", "risk review is overdue")
		}
		policy = "quality.risk.review"
	}
	return f.kernel.AcceptDone(ctx, actor, commandID, projectID, id, revision, head, []string{policy})
}

func (f *P2Flows) CreateReviewFinding(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string, data ReviewFindingData) (AppendResult, error) {
	if data.RuleID == "" || data.Summary == "" {
		return AppendResult{}, invalid("create review finding", "finding", "rule and summary are required")
	}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "review_finding", Revision: 1, State: "open", CreatorID: actor.ID}, data)
}

func (f *P2Flows) ResolveFinding(ctx context.Context, actor Actor, commandID, projectID, id string, revision, head int64) (AppendResult, error) {
	return f.kernel.AcceptDone(ctx, actor, commandID, projectID, id, revision, head, []string{"review.finding.resolve"})
}

func (f *P2Flows) CreateKnowledgeCandidate(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string, data KnowledgeData) (AppendResult, error) {
	if data.Title == "" || data.DedupKey == "" || len(data.SourceIDs) == 0 || len(data.EvidenceIDs) == 0 {
		return AppendResult{}, invalid("create knowledge candidate", "candidate", "title, dedup key, sources, and evidence are required")
	}
	data.Version = 1
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	for _, node := range projection.Nodes {
		if node.Kind != "knowledge_candidate" {
			continue
		}
		var existing KnowledgeData
		if json.Unmarshal(node.Data, &existing) == nil && existing.DedupKey == data.DedupKey {
			return AppendResult{}, conflict("create knowledge candidate", "dedup key already exists")
		}
	}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "knowledge_candidate", Revision: 1, State: "candidate", CreatorID: actor.ID}, data)
}

func (f *P2Flows) PublishKnowledge(ctx context.Context, actor Actor, commandID, projectID, id string, revision, head int64) (AppendResult, error) {
	return f.kernel.AcceptDone(ctx, actor, commandID, projectID, id, revision, head, []string{"knowledge.publish"})
}

func (f *P2Flows) InvalidateKnowledge(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok || node.Kind != "knowledge_candidate" {
		return AppendResult{}, notFound("invalidate knowledge", id)
	}
	var data KnowledgeData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("invalidate knowledge", "invalid data")
	}
	node.Revision++
	node.State = "invalidated"
	data.Version++
	return f.upsert(ctx, actor, commandID, projectID, head, node, data)
}

func (f *P2Flows) QueueRun(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, workspaceRef string, secretRefs []string, maxAttempts int) (AppendResult, error) {
	if workspaceRef == "" || maxAttempts < 1 {
		return AppendResult{}, invalid("queue run", "run", "workspace reference and positive max attempts are required")
	}
	for _, ref := range secretRefs {
		if !strings.HasPrefix(ref, "secret://") || strings.ContainsAny(ref, "?=#") {
			return AppendResult{}, invalid("queue run", "secret_ref", "must be an opaque secret reference")
		}
	}
	data := RunData{MaxAttempts: maxAttempts, WorkspaceRef: workspaceRef, SecretRefs: append([]string(nil), secretRefs...)}
	return f.upsert(ctx, actor, commandID, projectID, head, WorkNode{ID: id, Kind: "run", Revision: 1, State: "queued", CreatorID: actor.ID}, data)
}

func (f *P2Flows) ClaimRun(ctx context.Context, runner Actor, commandID, projectID string, head int64, id string, lease time.Duration) (AppendResult, error) {
	if runner.Kind != ActorAgent || lease <= 0 {
		return AppendResult{}, denied("claim run", "an Agent and positive lease are required")
	}
	projection, err := f.kernel.Replay(ctx, runner.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok || node.Kind != "run" {
		return AppendResult{}, notFound("claim run", id)
	}
	var data RunData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("claim run", "invalid run data")
	}
	if node.State == "running" && data.LeaseUntil.After(f.kernel.now()) {
		return AppendResult{}, conflict("claim run", "run has an active lease")
	}
	if data.Attempt >= data.MaxAttempts {
		return AppendResult{}, invariant("claim run", "retry limit reached")
	}
	data.Attempt++
	data.LeaseOwner = runner.ID
	data.LeaseUntil = f.kernel.now().Add(lease)
	node.Revision++
	node.State = "running"
	node.ExecutorIDs = []string{runner.ID}
	return f.upsert(ctx, runner, commandID, projectID, head, node, data)
}

func (f *P2Flows) CompleteRun(ctx context.Context, runner Actor, commandID, projectID string, head int64, id string) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, runner.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok {
		return AppendResult{}, notFound("complete run", id)
	}
	var data RunData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("complete run", "invalid run data")
	}
	if runner.Kind != ActorAgent || data.LeaseOwner != runner.ID || data.LeaseUntil.Before(f.kernel.now()) {
		return AppendResult{}, denied("complete run", "active lease owner is required")
	}
	var hasEvidence bool
	for _, evidence := range projection.Evidence {
		if evidence.SubjectID == id {
			hasEvidence = true
			break
		}
	}
	if !hasEvidence {
		return AppendResult{}, invariant("complete run", "runner must return evidence before validation")
	}
	node.Revision++
	node.State = "validation"
	return f.upsert(ctx, runner, commandID, projectID, head, node, data)
}

func (f *P2Flows) HeartbeatRun(ctx context.Context, runner Actor, commandID, projectID string, head int64, id string, lease time.Duration) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, runner.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok {
		return AppendResult{}, notFound("heartbeat run", id)
	}
	var data RunData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("heartbeat run", "invalid run data")
	}
	if runner.Kind != ActorAgent || node.State != "running" || data.LeaseOwner != runner.ID || data.LeaseUntil.Before(f.kernel.now()) || lease <= 0 {
		return AppendResult{}, denied("heartbeat run", "active lease owner is required")
	}
	data.LeaseUntil = f.kernel.now().Add(lease)
	node.Revision++
	return f.upsert(ctx, runner, commandID, projectID, head, node, data)
}

func (f *P2Flows) CancelRun(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok {
		return AppendResult{}, notFound("cancel run", id)
	}
	var data RunData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("cancel run", "invalid run data")
	}
	node.Revision++
	node.State = "cancelled"
	data.LeaseOwner = ""
	data.LeaseUntil = time.Time{}
	return f.upsert(ctx, actor, commandID, projectID, head, node, data)
}

func (f *P2Flows) RetryRun(ctx context.Context, actor Actor, commandID, projectID string, head int64, id string) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok {
		return AppendResult{}, notFound("retry run", id)
	}
	var data RunData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("retry run", "invalid run data")
	}
	if data.Attempt >= data.MaxAttempts || (node.State != "cancelled" && node.State != "failed") {
		return AppendResult{}, invariant("retry run", "run is not retryable")
	}
	node.Revision++
	node.State = "queued"
	data.LeaseOwner = ""
	data.LeaseUntil = time.Time{}
	return f.upsert(ctx, actor, commandID, projectID, head, node, data)
}

func (f *P2Flows) upsert(ctx context.Context, actor Actor, commandID, projectID string, head int64, node WorkNode, data any) (AppendResult, error) {
	encoded, err := canonicalKernelRequest(data)
	if err != nil {
		return AppendResult{}, err
	}
	node.Data = encoded
	return f.kernel.UpsertNode(ctx, actor, commandID, projectID, head, node)
}

func (f *P2Flows) updateRequirement(ctx context.Context, actor Actor, commandID, projectID string, head int64, id, state string, update func(*RequirementData) error) (AppendResult, error) {
	projection, err := f.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[id]
	if !ok || node.Kind != "requirement" {
		return AppendResult{}, notFound("update requirement", id)
	}
	var data RequirementData
	if err := json.Unmarshal(node.Data, &data); err != nil {
		return AppendResult{}, invariant("update requirement", "invalid data")
	}
	if err := update(&data); err != nil {
		return AppendResult{}, err
	}
	node.Revision++
	node.State = state
	return f.upsert(ctx, actor, commandID, projectID, head, node, data)
}
