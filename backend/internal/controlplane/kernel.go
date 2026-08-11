package controlplane

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"time"
)

type DeliveryKernel struct {
	store     KernelStore
	now       Clock
	authorize AuthorizeFunc
}

type AuthorizeFunc func(context.Context, Actor, string) error

func NewDeliveryKernel(store KernelStore, clock Clock, authorize AuthorizeFunc) (*DeliveryKernel, error) {
	if store == nil {
		return nil, invalid("new delivery kernel", "store", "is required")
	}
	if clock == nil {
		clock = func() time.Time { return time.Now().UTC() }
	}
	return &DeliveryKernel{store: store, now: clock, authorize: authorize}, nil
}

func KernelStoreFrom(repository Repository) (KernelStore, error) {
	store, ok := repository.(KernelStore)
	if !ok {
		return nil, invalid("resolve kernel store", "repository", "does not support the delivery kernel")
	}
	return store, nil
}

func (k *DeliveryKernel) UpsertNode(ctx context.Context, actor Actor, commandID, projectID string, expectedHead int64, node WorkNode) (AppendResult, error) {
	const op = "upsert work node"
	if err := k.allow(ctx, actor, PermissionWrite); err != nil {
		return AppendResult{}, err
	}
	if err := validateWorkNode(op, node); err != nil {
		return AppendResult{}, err
	}
	projection, err := k.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	if current, ok := projection.Nodes[node.ID]; ok {
		if node.Revision == current.Revision && reflect.DeepEqual(node, current) {
			// Permit the store to replay the original command result.
		} else if node.Revision != current.Revision+1 {
			return AppendResult{}, conflict(op, "node revision must advance by one")
		}
	} else if node.Revision != 1 {
		return AppendResult{}, conflict(op, "new node revision must be one")
	}
	request, _ := canonicalKernelRequest(node)
	payload, _ := canonicalKernelRequest(node)
	return k.append(ctx, actor, commandID, projectID, "work.node.upsert", expectedHead, request, EventWorkNodeUpserted, payload)
}

func (k *DeliveryKernel) AddEdge(ctx context.Context, actor Actor, commandID, projectID string, expectedHead int64, edge WorkEdge) (AppendResult, error) {
	const op = "add work edge"
	if err := k.allow(ctx, actor, PermissionWrite); err != nil {
		return AppendResult{}, err
	}
	for field, value := range map[string]string{"edge_id": edge.ID, "from": edge.From, "to": edge.To} {
		if err := validateIdentifier(op, field, value); err != nil {
			return AppendResult{}, err
		}
	}
	if edge.From == edge.To || (edge.Kind != "depends_on" && edge.Kind != "blocks" && edge.Kind != "trace") {
		return AppendResult{}, invalid(op, "edge", "requires distinct nodes and a supported kind")
	}
	projection, err := k.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	if _, ok := projection.Nodes[edge.From]; !ok {
		return AppendResult{}, notFound(op, edge.From)
	}
	if _, ok := projection.Nodes[edge.To]; !ok {
		return AppendResult{}, notFound(op, edge.To)
	}
	if existing, ok := projection.Edges[edge.ID]; ok {
		if existing != edge {
			return AppendResult{}, conflict(op, "edge id already exists")
		}
	}
	projection.Edges[edge.ID] = edge
	if (edge.Kind == "depends_on" || edge.Kind == "blocks") && graphHasCycle(projection) {
		return AppendResult{}, invariant(op, "dependency edge creates a cycle")
	}
	request, _ := canonicalKernelRequest(edge)
	payload, _ := canonicalKernelRequest(edge)
	return k.append(ctx, actor, commandID, projectID, "work.edge.add", expectedHead, request, EventWorkEdgeAdded, payload)
}

func (k *DeliveryKernel) AttachEvidence(ctx context.Context, actor Actor, commandID, projectID string, expectedHead int64, evidence EvidenceRef) (AppendResult, error) {
	const op = "attach evidence"
	if err := k.allow(ctx, actor, PermissionRun); err != nil {
		return AppendResult{}, err
	}
	if evidence.ProducedBy != actor.ID {
		return AppendResult{}, denied(op, "producer does not match actor")
	}
	if err := validateEvidence(op, evidence); err != nil {
		return AppendResult{}, err
	}
	projection, err := k.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	if _, ok := projection.Nodes[evidence.SubjectID]; !ok {
		return AppendResult{}, notFound(op, evidence.SubjectID)
	}
	if existing, ok := projection.Evidence[evidence.ID]; ok && existing.SHA256 != evidence.SHA256 {
		return AppendResult{}, invariant(op, "evidence id cannot change digest")
	}
	request, _ := canonicalKernelRequest(evidence)
	evidence.CapturedAt = k.now()
	payload, _ := canonicalKernelRequest(evidence)
	return k.append(ctx, actor, commandID, projectID, "evidence.attach", expectedHead, request, EventEvidenceAttached, payload)
}

func (k *DeliveryKernel) RecordCheck(ctx context.Context, actor Actor, commandID, projectID string, expectedHead int64, check CheckResult) (AppendResult, error) {
	const op = "record deterministic check"
	if actor.Kind != ActorHuman {
		return AppendResult{}, denied(op, "checker authority requires a human reviewer identity")
	}
	if err := k.allow(ctx, actor, PermissionReview); err != nil {
		return AppendResult{}, err
	}
	if !check.Deterministic || check.CheckerID != actor.ID || (check.Outcome != CheckPassed && check.Outcome != CheckFailed) {
		return AppendResult{}, invalid(op, "check", "deterministic human checker and supported outcome are required")
	}
	for field, value := range map[string]string{"check_id": check.ID, "policy_id": check.PolicyID, "subject_id": check.SubjectID} {
		if err := validateIdentifier(op, field, value); err != nil {
			return AppendResult{}, err
		}
	}
	projection, err := k.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[check.SubjectID]
	if !ok {
		return AppendResult{}, notFound(op, check.SubjectID)
	}
	if node.Revision != check.Revision || len(check.EvidenceIDs) == 0 {
		return AppendResult{}, conflict(op, "check revision or evidence is stale")
	}
	for _, evidenceID := range check.EvidenceIDs {
		if _, ok := projection.Evidence[evidenceID]; !ok {
			return AppendResult{}, notFound(op, evidenceID)
		}
	}
	request, _ := canonicalKernelRequest(check)
	check.RecordedAt = k.now()
	payload, _ := canonicalKernelRequest(check)
	return k.append(ctx, actor, commandID, projectID, "check.record", expectedHead, request, EventCheckRecorded, payload)
}

func (k *DeliveryKernel) AcceptDone(ctx context.Context, actor Actor, commandID, projectID, subjectID string, revision, expectedHead int64, requiredPolicies []string) (AppendResult, error) {
	const op = "accept done gate"
	if actor.Kind != ActorHuman {
		return AppendResult{}, denied(op, "only a human can accept")
	}
	if err := k.allow(ctx, actor, PermissionAccept); err != nil {
		return AppendResult{}, err
	}
	projection, err := k.Replay(ctx, actor.WorkspaceID, projectID)
	if err != nil {
		return AppendResult{}, err
	}
	node, ok := projection.Nodes[subjectID]
	if !ok {
		return AppendResult{}, notFound(op, subjectID)
	}
	if node.Revision != revision {
		return AppendResult{}, conflict(op, "subject revision changed")
	}
	if actor.ID == node.CreatorID || contains(node.AssigneeIDs, actor.ID) || contains(node.ExecutorIDs, actor.ID) {
		return AppendResult{}, denied(op, "acceptor must be independent")
	}
	if len(requiredPolicies) == 0 {
		return AppendResult{}, invariant(op, "at least one required policy is required")
	}
	for _, edge := range projection.Edges {
		if edge.Kind == "blocks" && edge.To == subjectID {
			if blocker, exists := projection.Nodes[edge.From]; exists && blocker.State != "done" {
				return AppendResult{}, invariant(op, "work graph contains an unresolved blocker")
			}
		}
	}
	for _, policyID := range requiredPolicies {
		checks := projection.Checks[subjectID]
		var latest *CheckResult
		for index := range checks {
			candidate := &checks[index]
			if candidate.PolicyID == policyID && candidate.Revision == revision && (latest == nil || candidate.RecordedAt.After(latest.RecordedAt)) {
				latest = candidate
			}
		}
		if latest == nil || !latest.Deterministic || latest.Outcome != CheckPassed {
			return AppendResult{}, invariant(op, "required deterministic policy has not passed")
		}
		for _, evidenceID := range latest.EvidenceIDs {
			if evidence, ok := projection.Evidence[evidenceID]; !ok || !validSHA256(evidence.SHA256) || !evidence.Sanitized {
				return AppendResult{}, invariant(op, "required evidence is missing or invalid")
			}
		}
	}
	policies := append([]string(nil), requiredPolicies...)
	sort.Strings(policies)
	request, _ := canonicalKernelRequest(struct {
		SubjectID string   `json:"subject_id"`
		Revision  int64    `json:"revision"`
		Policies  []string `json:"policies"`
	}{subjectID, revision, policies})
	acceptance := Acceptance{SubjectID: subjectID, Revision: revision, AcceptorID: actor.ID, PolicyIDs: policies, AcceptedAt: k.now()}
	payload, _ := canonicalKernelRequest(acceptance)
	return k.append(ctx, actor, commandID, projectID, "done.accept", expectedHead, request, EventAcceptanceRecorded, payload)
}

func (k *DeliveryKernel) Replay(ctx context.Context, workspaceID, projectID string) (ProjectProjection, error) {
	events, err := k.store.ListSessionEvents(ctx, workspaceID, projectID)
	if err != nil {
		return ProjectProjection{}, err
	}
	projection := ProjectProjection{SchemaVersion: kernelSchemaVersion, WorkspaceID: workspaceID, ProjectID: projectID, Nodes: map[string]WorkNode{}, Edges: map[string]WorkEdge{}, Evidence: map[string]EvidenceRef{}, Checks: map[string][]CheckResult{}, Acceptances: map[string]Acceptance{}}
	previousHash := ""
	for index, event := range events {
		if event.SchemaVersion != kernelSchemaVersion || event.WorkspaceID != workspaceID || event.ProjectID != projectID || event.Sequence != int64(index+1) || event.PreviousHash != previousHash || hashSessionEvent(event) != event.Hash {
			return ProjectProjection{}, invariant("replay kernel", "event chain integrity failed")
		}
		switch event.Type {
		case EventWorkNodeUpserted:
			var value WorkNode
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProjectProjection{}, invariant("replay kernel", "invalid node event")
			}
			projection.Nodes[value.ID] = value
		case EventWorkEdgeAdded:
			var value WorkEdge
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProjectProjection{}, invariant("replay kernel", "invalid edge event")
			}
			projection.Edges[value.ID] = value
		case EventEvidenceAttached:
			var value EvidenceRef
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProjectProjection{}, invariant("replay kernel", "invalid evidence event")
			}
			if previous, ok := projection.Evidence[value.ID]; ok && previous.SHA256 != value.SHA256 {
				return ProjectProjection{}, invariant("replay kernel", "evidence digest changed")
			}
			projection.Evidence[value.ID] = value
		case EventCheckRecorded:
			var value CheckResult
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProjectProjection{}, invariant("replay kernel", "invalid check event")
			}
			projection.Checks[value.SubjectID] = append(projection.Checks[value.SubjectID], value)
		case EventAcceptanceRecorded:
			var value Acceptance
			if err := json.Unmarshal(event.Payload, &value); err != nil {
				return ProjectProjection{}, invariant("replay kernel", "invalid acceptance event")
			}
			projection.Acceptances[value.SubjectID] = value
			node := projection.Nodes[value.SubjectID]
			node.State = "done"
			projection.Nodes[value.SubjectID] = node
		default:
			return ProjectProjection{}, invariant("replay kernel", "unknown event type")
		}
		projection.Head, projection.HeadHash, previousHash = event.Sequence, event.Hash, event.Hash
	}
	return projection, nil
}

func (k *DeliveryKernel) append(ctx context.Context, actor Actor, commandID, projectID, name string, expectedHead int64, request json.RawMessage, eventType string, payload json.RawMessage) (AppendResult, error) {
	command := CommandEnvelope{WorkspaceID: actor.WorkspaceID, ProjectID: projectID, CommandID: commandID, Name: name, Actor: actor, ExpectedHead: expectedHead, Request: request}
	return k.store.AppendCommand(ctx, command, []ProposedEvent{{Type: eventType, Payload: payload, OccurredAt: k.now()}})
}

func (k *DeliveryKernel) allow(ctx context.Context, actor Actor, permission string) error {
	if k.authorize != nil {
		return k.authorize(ctx, actor, permission)
	}
	return validateActor(actor, true)
}

func validateWorkNode(op string, node WorkNode) error {
	if err := validateIdentifier(op, "node_id", node.ID); err != nil {
		return err
	}
	if node.Kind == "" || node.State == "" || node.Revision < 1 || node.CreatorID == "" {
		return invalid(op, "node", "kind, state, revision, and creator are required")
	}
	return nil
}

func validateEvidence(op string, evidence EvidenceRef) error {
	for field, value := range map[string]string{"evidence_id": evidence.ID, "subject_id": evidence.SubjectID, "producer_id": evidence.ProducedBy} {
		if err := validateIdentifier(op, field, value); err != nil {
			return err
		}
	}
	parsed, err := url.Parse(evidence.URI)
	if err != nil || parsed.Scheme != "artifact" || parsed.RawQuery != "" || parsed.User != nil {
		return invalid(op, "uri", "must be an artifact URI without credentials or query parameters")
	}
	if evidence.Kind == "" || evidence.MediaType == "" || evidence.Size < 0 || !validSHA256(evidence.SHA256) || !evidence.Sanitized {
		return invalid(op, "evidence", "immutable digest, media type, size, and sanitized marker are required")
	}
	return nil
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func graphHasCycle(projection ProjectProjection) bool {
	adjacent := map[string][]string{}
	for _, edge := range projection.Edges {
		if edge.Kind == "depends_on" || edge.Kind == "blocks" {
			adjacent[edge.From] = append(adjacent[edge.From], edge.To)
		}
	}
	visiting, visited := map[string]bool{}, map[string]bool{}
	var visit func(string) bool
	visit = func(node string) bool {
		if visiting[node] {
			return true
		}
		if visited[node] {
			return false
		}
		visiting[node] = true
		for _, next := range adjacent[node] {
			if visit(next) {
				return true
			}
		}
		visiting[node] = false
		visited[node] = true
		return false
	}
	for node := range projection.Nodes {
		if visit(node) {
			return true
		}
	}
	return false
}

func stableProjectionDigest(projection ProjectProjection) string {
	encoded, _ := json.Marshal(projection)
	digest := sha256.Sum256(encoded)
	return fmt.Sprintf("%x", digest[:])
}
