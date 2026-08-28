package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"
)

type ContextKind string

const (
	ContextKindEngineeringEntity ContextKind = "engineering_entity"
	ContextKindRequirement       ContextKind = "requirement"
	ContextKindArchitecture      ContextKind = "architecture"
	ContextKindADR               ContextKind = "adr"
	ContextKindStandard          ContextKind = "standard"
	ContextKindRunbook           ContextKind = "runbook"
	ContextKindChange            ContextKind = "change"
	ContextKindIncident          ContextKind = "incident"
	ContextKindFact              ContextKind = "fact"
	ContextKindKnowledge         ContextKind = "knowledge"
)

var validContextKinds = map[ContextKind]struct{}{
	ContextKindEngineeringEntity: {}, ContextKindRequirement: {}, ContextKindArchitecture: {},
	ContextKindADR: {}, ContextKindStandard: {}, ContextKindRunbook: {}, ContextKindChange: {},
	ContextKindIncident: {}, ContextKindFact: {}, ContextKindKnowledge: {},
}

func (value ContextKind) Valid() bool {
	_, ok := validContextKinds[ContextKind(strings.TrimSpace(string(value)))]
	return ok
}

var (
	ErrContextPackIDRequired            = errors.New("context pack id is required")
	ErrWorkItemKindInvalid              = errors.New("context pack work item kind is invalid")
	ErrWorkItemRevisionRequired         = errors.New("work item revision is required")
	ErrTargetEntityRequired             = errors.New("context pack requires at least one target engineering entity")
	ErrContextKindInvalid               = errors.New("invalid context reference kind")
	ErrContextReferenceIDRequired       = errors.New("context reference id is required")
	ErrContextReferenceRevisionRequired = errors.New("context reference revision is required")
	ErrContextReferenceChecksumRequired = errors.New("context reference checksum is required")
	ErrContextPolicyVersionRequired     = errors.New("context policy version is required")
	ErrContextPackChecksumMismatch      = errors.New("context pack checksum mismatch")
)

type ContextReference struct {
	kind     ContextKind
	id       string
	revision string
	checksum string
}

func NewContextReference(kind ContextKind, id, revision, checksum string) (ContextReference, error) {
	id = strings.TrimSpace(id)
	revision = strings.TrimSpace(revision)
	checksum = strings.TrimSpace(checksum)
	if !kind.Valid() {
		return ContextReference{}, ErrContextKindInvalid
	}
	if id == "" {
		return ContextReference{}, ErrContextReferenceIDRequired
	}
	if revision == "" {
		return ContextReference{}, ErrContextReferenceRevisionRequired
	}
	if checksum == "" {
		return ContextReference{}, ErrContextReferenceChecksumRequired
	}
	return ContextReference{kind: kind, id: id, revision: revision, checksum: checksum}, nil
}

func (value ContextReference) Kind() ContextKind { return value.kind }
func (value ContextReference) ID() string        { return value.id }
func (value ContextReference) Revision() string  { return value.revision }
func (value ContextReference) Checksum() string  { return value.checksum }

type ContextPack struct {
	id               string
	workspaceID      string
	workItem         NodeRef
	workItemRevision string
	targetEntityIDs  []string
	references       []ContextReference
	policyVersion    string
	checksum         string
	createdAt        time.Time
}

func NewContextPack(id, workspaceID string, workItem NodeRef, workItemRevision string, targetEntityIDs []string, references []ContextReference, policyVersion string, createdAt time.Time) (ContextPack, error) {
	id = strings.TrimSpace(id)
	workspaceID = strings.TrimSpace(workspaceID)
	workItemRevision = strings.TrimSpace(workItemRevision)
	policyVersion = strings.TrimSpace(policyVersion)
	if id == "" {
		return ContextPack{}, ErrContextPackIDRequired
	}
	if workspaceID == "" {
		return ContextPack{}, ErrWorkspaceIDRequired
	}
	if !isWorkItemKind(workItem.Kind()) || workItem.ID() == "" {
		return ContextPack{}, ErrWorkItemKindInvalid
	}
	if workItemRevision == "" {
		return ContextPack{}, ErrWorkItemRevisionRequired
	}
	targets, err := normalizeContextTargets(targetEntityIDs)
	if err != nil {
		return ContextPack{}, err
	}
	refs, err := normalizeContextReferences(references)
	if err != nil {
		return ContextPack{}, err
	}
	if policyVersion == "" {
		return ContextPack{}, ErrContextPolicyVersionRequired
	}
	if createdAt.IsZero() {
		return ContextPack{}, ErrTimestampRequired
	}
	return ContextPack{
		id: id, workspaceID: workspaceID, workItem: workItem, workItemRevision: workItemRevision,
		targetEntityIDs: targets, references: refs, policyVersion: policyVersion,
		checksum:  contextPackChecksum(workspaceID, workItem, workItemRevision, targets, refs, policyVersion),
		createdAt: createdAt.UTC(),
	}, nil
}

func RehydrateContextPack(id, workspaceID string, workItem NodeRef, workItemRevision string, targetEntityIDs []string, references []ContextReference, policyVersion, checksum string, createdAt time.Time) (ContextPack, error) {
	value, err := NewContextPack(id, workspaceID, workItem, workItemRevision, targetEntityIDs, references, policyVersion, createdAt)
	if err != nil {
		return ContextPack{}, err
	}
	if value.checksum != strings.TrimSpace(checksum) {
		return ContextPack{}, ErrContextPackChecksumMismatch
	}
	return value, nil
}

func normalizeContextTargets(values []string) ([]string, error) {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, ErrTargetEntityRequired
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil, ErrTargetEntityRequired
	}
	sort.Strings(result)
	return result, nil
}

func normalizeContextReferences(values []ContextReference) ([]ContextReference, error) {
	result := append([]ContextReference(nil), values...)
	for _, value := range result {
		if !value.kind.Valid() {
			return nil, ErrContextKindInvalid
		}
		if value.id == "" {
			return nil, ErrContextReferenceIDRequired
		}
		if value.revision == "" {
			return nil, ErrContextReferenceRevisionRequired
		}
		if value.checksum == "" {
			return nil, ErrContextReferenceChecksumRequired
		}
	}
	sort.Slice(result, func(i, j int) bool {
		left := string(result[i].kind) + "\x00" + result[i].id + "\x00" + result[i].revision + "\x00" + result[i].checksum
		right := string(result[j].kind) + "\x00" + result[j].id + "\x00" + result[j].revision + "\x00" + result[j].checksum
		return left < right
	})
	return result, nil
}

func contextPackChecksum(workspaceID string, workItem NodeRef, workItemRevision string, targets []string, references []ContextReference, policyVersion string) string {
	type canonicalReference struct {
		Kind     ContextKind `json:"kind"`
		ID       string      `json:"id"`
		Revision string      `json:"revision"`
		Checksum string      `json:"checksum"`
	}
	payload := struct {
		WorkspaceID      string               `json:"workspace_id"`
		WorkItemKind     NodeKind             `json:"work_item_kind"`
		WorkItemID       string               `json:"work_item_id"`
		WorkItemRevision string               `json:"work_item_revision"`
		TargetEntityIDs  []string             `json:"target_entity_ids"`
		References       []canonicalReference `json:"references"`
		PolicyVersion    string               `json:"policy_version"`
	}{
		WorkspaceID: workspaceID, WorkItemKind: workItem.Kind(), WorkItemID: workItem.ID(), WorkItemRevision: workItemRevision,
		TargetEntityIDs: targets, PolicyVersion: policyVersion,
	}
	for _, reference := range references {
		payload.References = append(payload.References, canonicalReference{
			Kind: reference.Kind(), ID: reference.ID(), Revision: reference.Revision(), Checksum: reference.Checksum(),
		})
	}
	encoded, _ := json.Marshal(payload)
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

func (value ContextPack) ID() string               { return value.id }
func (value ContextPack) WorkspaceID() string      { return value.workspaceID }
func (value ContextPack) WorkItem() NodeRef        { return value.workItem }
func (value ContextPack) WorkItemRevision() string { return value.workItemRevision }
func (value ContextPack) TargetEntityIDs() []string {
	return append([]string(nil), value.targetEntityIDs...)
}
func (value ContextPack) References() []ContextReference {
	return append([]ContextReference(nil), value.references...)
}
func (value ContextPack) PolicyVersion() string { return value.policyVersion }
func (value ContextPack) Checksum() string      { return value.checksum }
func (value ContextPack) CreatedAt() time.Time  { return value.createdAt }
