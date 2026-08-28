package controlplane

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

var ErrRuntimeContextPackNotFound = errors.New("runtime context pack not found")

type FrozenRuntimeContextPack struct {
	ID               string
	WorkspaceID      string
	WorkItemKind     string
	WorkItemID       string
	WorkItemRevision string
	PolicyVersion    string
	Checksum         string
}

type RuntimeContextPackReader interface {
	ResolveFrozenContextPack(context.Context, string, string) (FrozenRuntimeContextPack, error)
}

type RuntimeContextPackReaderFunc func(context.Context, string, string) (FrozenRuntimeContextPack, error)

func (f RuntimeContextPackReaderFunc) ResolveFrozenContextPack(ctx context.Context, workspaceID, packID string) (FrozenRuntimeContextPack, error) {
	return f(ctx, workspaceID, packID)
}

type SkillVersionPin struct {
	SkillID   string `json:"skill_id"`
	VersionID string `json:"version_id"`
}

type RunExecutionContextRequest struct {
	ContextPackID       string            `json:"context_pack_id"`
	ContextPackChecksum string            `json:"context_pack_checksum"`
	AgentReleaseID      string            `json:"agent_release_id"`
	SkillVersions       []SkillVersionPin `json:"skill_versions,omitempty"`
}

type RunExecutionContextData struct {
	ContextPackID       string            `json:"context_pack_id"`
	ContextPackChecksum string            `json:"context_pack_checksum"`
	WorkItemKind        string            `json:"work_item_kind"`
	WorkItemID          string            `json:"work_item_id"`
	WorkItemRevision    string            `json:"work_item_revision"`
	ContextPolicy       string            `json:"context_policy"`
	AgentReleaseID      string            `json:"agent_release_id"`
	SkillVersions       []SkillVersionPin `json:"skill_versions,omitempty"`
}

type RunContextBinder struct {
	kernel   *DeliveryKernel
	contexts RuntimeContextPackReader
}

func NewRunContextBinder(kernel *DeliveryKernel, contexts RuntimeContextPackReader) (*RunContextBinder, error) {
	if kernel == nil || contexts == nil {
		return nil, invalid("new run context binder", "dependency", "delivery kernel and ContextPack reader are required")
	}
	return &RunContextBinder{kernel: kernel, contexts: contexts}, nil
}

func (b *RunContextBinder) QueueRun(ctx context.Context, actor Actor, commandID, projectID string, expectedHead int64, runID, workspaceRef string, secretRefs []string, maxAttempts int, request RunExecutionContextRequest) (AppendResult, error) {
	const op = "queue contextual run"
	if err := b.kernel.allow(ctx, actor, PermissionWrite); err != nil {
		return AppendResult{}, err
	}
	if err := validateIdentifier(op, "run_id", runID); err != nil {
		return AppendResult{}, err
	}
	if err := validateIdentifier(op, "project_id", projectID); err != nil {
		return AppendResult{}, err
	}
	if !validWorkspaceReference(workspaceRef) || maxAttempts < 1 {
		return AppendResult{}, invalid(op, "run", "workspace reference and positive max attempts are required")
	}
	for _, ref := range secretRefs {
		if !validSecretReference(ref) {
			return AppendResult{}, invalid(op, "secret_ref", "must be a canonical secret://provider/uuid reference")
		}
	}

	request.ContextPackID = strings.TrimSpace(request.ContextPackID)
	request.ContextPackChecksum = strings.ToLower(strings.TrimSpace(request.ContextPackChecksum))
	request.AgentReleaseID = strings.TrimSpace(request.AgentReleaseID)
	if !safeImmutableAuditRef(request.ContextPackID, 128) || !validSHA256(request.ContextPackChecksum) {
		return AppendResult{}, invalid(op, "context_pack", "id and canonical SHA-256 checksum are required")
	}
	if err := validateIdentifier(op, "agent_release_id", request.AgentReleaseID); err != nil {
		return AppendResult{}, err
	}
	skills, err := normalizeSkillVersionPins(request.SkillVersions)
	if err != nil {
		return AppendResult{}, err
	}
	request.SkillVersions = skills

	commandRequest, _ := canonicalKernelRequest(struct {
		RunID        string                     `json:"run_id"`
		WorkspaceRef string                     `json:"workspace_ref"`
		SecretRefs   []string                   `json:"secret_refs,omitempty"`
		MaxAttempts  int                        `json:"max_attempts"`
		Context      RunExecutionContextRequest `json:"context"`
	}{runID, workspaceRef, append([]string(nil), secretRefs...), maxAttempts, request})
	command := CommandEnvelope{
		WorkspaceID:  actor.WorkspaceID,
		ProjectID:    projectID,
		CommandID:    commandID,
		Name:         "run.queue.contextual",
		Actor:        actor,
		ExpectedHead: expectedHead,
		Request:      commandRequest,
	}
	if receipts, ok := b.kernel.store.(kernelCommandReceiptReader); ok {
		found, lookupErr := receipts.KernelCommandExists(ctx, actor.WorkspaceID, projectID, commandID)
		if lookupErr != nil {
			return AppendResult{}, unavailable(op, "read command receipt")
		}
		if found {
			return b.kernel.store.AppendCommand(ctx, command, nil)
		}
	}

	pack, err := b.contexts.ResolveFrozenContextPack(ctx, actor.WorkspaceID, request.ContextPackID)
	if err != nil {
		if errors.Is(err, ErrRuntimeContextPackNotFound) {
			return AppendResult{}, notFound(op, request.ContextPackID)
		}
		return AppendResult{}, unavailable(op, "resolve frozen ContextPack")
	}
	pack = normalizeFrozenRuntimeContextPack(pack)
	if pack.WorkspaceID != actor.WorkspaceID || pack.ID != request.ContextPackID {
		return AppendResult{}, invariant(op, "resolved ContextPack identity does not match the run workspace/request")
	}
	if !validSHA256(pack.Checksum) || pack.Checksum != request.ContextPackChecksum {
		return AppendResult{}, conflict(op, "ContextPack checksum changed")
	}
	if !validRuntimeWorkPin(pack.WorkItemKind, pack.WorkItemID, pack.WorkItemRevision) || !safeImmutableAuditRef(pack.PolicyVersion, 128) {
		return AppendResult{}, invariant(op, "resolved ContextPack is missing immutable work/policy identity")
	}

	contextData := RunExecutionContextData{
		ContextPackID:       pack.ID,
		ContextPackChecksum: pack.Checksum,
		WorkItemKind:        pack.WorkItemKind,
		WorkItemID:          pack.WorkItemID,
		WorkItemRevision:    pack.WorkItemRevision,
		ContextPolicy:       pack.PolicyVersion,
		AgentReleaseID:      request.AgentReleaseID,
		SkillVersions:       append([]SkillVersionPin(nil), skills...),
	}
	if err := validateRunExecutionContext(contextData); err != nil {
		return AppendResult{}, err
	}

	runData, _ := canonicalKernelRequest(RunData{MaxAttempts: maxAttempts, WorkspaceRef: workspaceRef, SecretRefs: append([]string(nil), secretRefs...)})
	contextDataJSON, _ := canonicalKernelRequest(contextData)
	runNode := WorkNode{ID: runID, Kind: "run", Revision: 1, State: "queued", CreatorID: actor.ID, Data: runData}
	contextNodeID := runContextNodeID(actor.WorkspaceID, projectID, runID)
	contextNode := WorkNode{ID: contextNodeID, Kind: "run_context", Revision: 1, State: "frozen", CreatorID: actor.ID, Data: contextDataJSON}
	edge := WorkEdge{ID: runContextEdgeID(actor.WorkspaceID, projectID, runID), From: runID, To: contextNodeID, Kind: "trace"}
	if err := validateWorkNode(op, runNode); err != nil {
		return AppendResult{}, err
	}
	if err := validateWorkNode(op, contextNode); err != nil {
		return AppendResult{}, err
	}
	for field, value := range map[string]string{"edge_id": edge.ID, "edge_from": edge.From, "edge_to": edge.To} {
		if err := validateIdentifier(op, field, value); err != nil {
			return AppendResult{}, err
		}
	}

	projection, err := b.kernel.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	if _, exists := projection.Nodes[runNode.ID]; exists {
		return AppendResult{}, conflict(op, "run id already exists")
	}
	if _, exists := projection.Nodes[contextNode.ID]; exists {
		return AppendResult{}, conflict(op, "run context binding already exists")
	}
	if _, exists := projection.Edges[edge.ID]; exists {
		return AppendResult{}, conflict(op, "run context trace already exists")
	}

	now := b.kernel.now().UTC()
	runPayload, _ := canonicalKernelRequest(runNode)
	contextPayload, _ := canonicalKernelRequest(contextNode)
	edgePayload, _ := canonicalKernelRequest(edge)
	return b.kernel.store.AppendCommand(ctx, command, []ProposedEvent{
		{Type: EventWorkNodeUpserted, Payload: runPayload, OccurredAt: now},
		{Type: EventWorkNodeUpserted, Payload: contextPayload, OccurredAt: now},
		{Type: EventWorkEdgeAdded, Payload: edgePayload, OccurredAt: now},
	})
}

func ResolveRunExecutionContext(projection ProjectProjection, runID string) (RunExecutionContextData, bool, error) {
	const op = "resolve run execution context"
	runID = strings.TrimSpace(runID)
	run, exists := projection.Nodes[runID]
	if !exists {
		return RunExecutionContextData{}, false, notFound(op, runID)
	}
	if run.Kind != "run" {
		return RunExecutionContextData{}, false, invariant(op, "subject is not a run")
	}
	var bindingID string
	for _, edge := range projection.Edges {
		if edge.Kind != "trace" || edge.From != runID {
			continue
		}
		node, ok := projection.Nodes[edge.To]
		if !ok || node.Kind != "run_context" {
			continue
		}
		if bindingID != "" && bindingID != edge.To {
			return RunExecutionContextData{}, false, invariant(op, "run has multiple frozen context bindings")
		}
		bindingID = edge.To
	}
	if bindingID == "" {
		return RunExecutionContextData{}, false, nil
	}
	node := projection.Nodes[bindingID]
	if node.Revision != 1 || node.State != "frozen" {
		return RunExecutionContextData{}, false, invariant(op, "run context binding is mutable")
	}
	var value RunExecutionContextData
	if err := json.Unmarshal(node.Data, &value); err != nil {
		return RunExecutionContextData{}, false, invariant(op, "run context binding payload is invalid")
	}
	if err := validateRunExecutionContext(value); err != nil {
		return RunExecutionContextData{}, false, invariant(op, "run context binding identity is invalid")
	}
	return value, true, nil
}

func normalizeFrozenRuntimeContextPack(value FrozenRuntimeContextPack) FrozenRuntimeContextPack {
	value.ID = strings.TrimSpace(value.ID)
	value.WorkspaceID = strings.TrimSpace(value.WorkspaceID)
	value.WorkItemKind = strings.TrimSpace(value.WorkItemKind)
	value.WorkItemID = strings.TrimSpace(value.WorkItemID)
	value.WorkItemRevision = strings.TrimSpace(value.WorkItemRevision)
	value.PolicyVersion = strings.TrimSpace(value.PolicyVersion)
	value.Checksum = strings.ToLower(strings.TrimSpace(value.Checksum))
	return value
}

func validRuntimeWorkPin(kind, id, revision string) bool {
	switch strings.TrimSpace(kind) {
	case "project", "requirement", "issue", "todo", "task":
	default:
		return false
	}
	return safeImmutableAuditRef(id, 128) && safeImmutableAuditRef(revision, 128)
}

func validateRunExecutionContext(value RunExecutionContextData) error {
	const op = "validate run execution context"
	if !safeImmutableAuditRef(value.ContextPackID, 128) || !validSHA256(strings.ToLower(strings.TrimSpace(value.ContextPackChecksum))) {
		return invalid(op, "context_pack", "immutable ContextPack identity is required")
	}
	if !validRuntimeWorkPin(value.WorkItemKind, value.WorkItemID, value.WorkItemRevision) || !safeImmutableAuditRef(value.ContextPolicy, 128) {
		return invalid(op, "work_item", "kind, id, revision, and context policy are required")
	}
	if err := validateIdentifier(op, "agent_release_id", strings.TrimSpace(value.AgentReleaseID)); err != nil {
		return err
	}
	_, err := normalizeSkillVersionPins(value.SkillVersions)
	return err
}

func normalizeSkillVersionPins(values []SkillVersionPin) ([]SkillVersionPin, error) {
	const op = "normalize Skill version pins"
	seen := make(map[string]struct{}, len(values))
	result := make([]SkillVersionPin, 0, len(values))
	for _, raw := range values {
		value := SkillVersionPin{SkillID: strings.TrimSpace(raw.SkillID), VersionID: strings.TrimSpace(raw.VersionID)}
		if err := validateIdentifier(op, "skill_id", value.SkillID); err != nil {
			return nil, err
		}
		if !safeImmutableAuditRef(value.VersionID, 128) {
			return nil, invalid(op, "version_id", "must be a non-empty immutable version reference")
		}
		key := value.SkillID + "\x00" + value.VersionID
		if _, exists := seen[key]; exists {
			return nil, conflict(op, "duplicate Skill version pin")
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SkillID != result[j].SkillID {
			return result[i].SkillID < result[j].SkillID
		}
		return result[i].VersionID < result[j].VersionID
	})
	return result, nil
}

func safeImmutableAuditRef(value string, max int) bool {
	value = strings.TrimSpace(value)
	return value != "" && len(value) <= max && !strings.ContainsAny(value, "\r\n\t")
}

func runContextNodeID(workspaceID, projectID, runID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{workspaceID, projectID, runID, "context"}, "\x00")))
	return "runctx-" + hex.EncodeToString(sum[:16])
}

func runContextEdgeID(workspaceID, projectID, runID string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{workspaceID, projectID, runID, "trace"}, "\x00")))
	return "runctx-edge-" + hex.EncodeToString(sum[:16])
}

type kernelCommandReceiptReader interface {
	KernelCommandExists(context.Context, string, string, string) (bool, error)
}

func (r *sqlRepository) KernelCommandExists(ctx context.Context, workspaceID, projectID, commandID string) (bool, error) {
	var marker int
	err := r.db.QueryRowContext(ctx, r.bind(`SELECT 1 FROM cp_kernel_commands WHERE workspace_id = ? AND project_id = ? AND command_id = ?`), workspaceID, projectID, commandID).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read kernel command receipt: %w", err)
	}
	return marker == 1, nil
}
