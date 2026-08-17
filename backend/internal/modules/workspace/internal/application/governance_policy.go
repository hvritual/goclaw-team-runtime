package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
)

type SafeFieldKind string

const (
	SafeIdentifier         SafeFieldKind = "identifier"
	SafeEnum               SafeFieldKind = "enum"
	SafeBoolean            SafeFieldKind = "boolean"
	SafeNonNegativeInteger SafeFieldKind = "non_negative_integer"
	SafeSHA256             SafeFieldKind = "sha256"
)

type SafeFieldRule struct {
	Kind       SafeFieldKind
	MaxLength  int
	EnumValues []string
	Required   bool
}

type EnvelopeSchema map[string]SafeFieldRule

// GovernanceActionPolicy is installed by a capability provider. Naming an
// action without an exact resource binding never installs it.
type GovernanceActionPolicy struct {
	Action        string
	ResourceKind  string
	RequestSchema EnvelopeSchema
	ReplaySchema  EnvelopeSchema
	AuditSchema   EnvelopeSchema
	EventSchemas  map[string]EnvelopeSchema
}

func (p GovernanceActionPolicy) validate() error {
	if strings.TrimSpace(p.Action) == "" || strings.TrimSpace(p.ResourceKind) == "" || p.RequestSchema == nil || p.ReplaySchema == nil || p.AuditSchema == nil || p.EventSchemas == nil {
		return contract.ErrGovernanceUnavailable
	}
	return nil
}

// GovernancePolicyProvider resolves both action policy and actor identity from
// trusted server context. Request actor fields are not an authority source.
type GovernancePolicyProvider interface {
	ResolveGovernancePolicy(context.Context, string, string, string, string) (contract.MutationIdentity, GovernanceActionPolicy, error)
}

type GovernanceEventPolicy struct {
	EventType     string
	AggregateKind string
	Schema        EnvelopeSchema
}

func (p GovernanceEventPolicy) validate() error {
	if strings.TrimSpace(p.EventType) == "" || strings.TrimSpace(p.AggregateKind) == "" || p.Schema == nil {
		return contract.ErrGovernanceUnavailable
	}
	return nil
}

// GovernanceEventPolicyProvider is the installed server-side catalog used to
// revalidate persisted event envelopes immediately before publication.
type GovernanceEventPolicyProvider interface {
	ResolveGovernanceEventPolicy(context.Context, string, string) (GovernanceEventPolicy, error)
}

func validateGovernanceEventPolicy(ctx context.Context, provider GovernanceEventPolicyProvider, event contract.OutboxEvent) error {
	if provider == nil {
		return contract.ErrGovernanceUnavailable
	}
	policy, err := provider.ResolveGovernanceEventPolicy(ctx, event.EventType, event.AggregateKind)
	if err != nil {
		return fmt.Errorf("resolve governance event policy: %w", err)
	}
	if err := policy.validate(); err != nil || policy.EventType != event.EventType || policy.AggregateKind != event.AggregateKind {
		return contract.ErrGovernanceUnavailable
	}
	return validateGovernanceEnvelopeFields("governance-outbox-v1", policy.Schema, event.Payload)
}

func resolveGovernancePolicy(ctx context.Context, provider GovernancePolicyProvider, request GovernanceRequest) (contract.MutationIdentity, GovernanceActionPolicy, error) {
	workspaceID := strings.TrimSpace(request.Identity.WorkspaceID)
	requestID := strings.TrimSpace(request.Identity.RequestID)
	action := strings.TrimSpace(request.Command.Action)
	resourceKind := strings.TrimSpace(request.Command.ResourceKind)
	if provider == nil || workspaceID == "" || requestID == "" || action == "" || resourceKind == "" {
		return contract.MutationIdentity{}, GovernanceActionPolicy{}, contract.ErrGovernanceUnavailable
	}
	identity, policy, err := provider.ResolveGovernancePolicy(ctx, workspaceID, requestID, action, resourceKind)
	if err != nil {
		return contract.MutationIdentity{}, GovernanceActionPolicy{}, fmt.Errorf("resolve governance policy: %w", err)
	}
	if err := policy.validate(); err != nil || policy.Action != action || policy.ResourceKind != resourceKind {
		return contract.MutationIdentity{}, GovernanceActionPolicy{}, contract.ErrGovernanceUnavailable
	}
	if err := identity.Validate(); err != nil || identity.WorkspaceID != workspaceID || identity.RequestID != requestID {
		return contract.MutationIdentity{}, GovernanceActionPolicy{}, contract.ErrGovernanceUnavailable
	}
	return identity, policy, nil
}

var safeIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/-]*$`)

func canonicalGovernanceFields(schema EnvelopeSchema, fields map[string]any) (map[string]any, error) {
	if schema == nil || fields == nil {
		return nil, contract.ErrInvalidGovernanceMutation
	}
	canonical := make(map[string]any, len(fields))
	for key, value := range fields {
		if contract.ContainsForbiddenGovernanceMaterial(key) {
			return nil, fmt.Errorf("%w: forbidden governance field", contract.ErrInvalidGovernanceMutation)
		}
		rule, ok := schema[key]
		if !ok {
			return nil, fmt.Errorf("%w: field %q is not allowed", contract.ErrInvalidGovernanceMutation, key)
		}
		normalized, err := normalizeGovernanceValue(rule, value)
		if err != nil {
			return nil, fmt.Errorf("%w: field %q", err, key)
		}
		canonical[key] = normalized
	}
	for key, rule := range schema {
		if _, ok := canonical[key]; rule.Required && !ok {
			return nil, fmt.Errorf("%w: field %q is required", contract.ErrInvalidGovernanceMutation, key)
		}
	}
	return canonical, nil
}

func normalizeGovernanceValue(rule SafeFieldRule, value any) (any, error) {
	switch rule.Kind {
	case SafeIdentifier:
		text, ok := value.(string)
		text = strings.TrimSpace(text)
		maxLength := rule.MaxLength
		if maxLength <= 0 {
			maxLength = 200
		}
		if !ok || len(text) == 0 || len(text) > maxLength || !safeIdentifierPattern.MatchString(text) || contract.ContainsForbiddenGovernanceMaterial(text) {
			return nil, contract.ErrInvalidGovernanceMutation
		}
		return text, nil
	case SafeEnum:
		text, ok := value.(string)
		if !ok || !containsExact(rule.EnumValues, text) || contract.ContainsForbiddenGovernanceMaterial(text) {
			return nil, contract.ErrInvalidGovernanceMutation
		}
		return text, nil
	case SafeBoolean:
		boolean, ok := value.(bool)
		if !ok {
			return nil, contract.ErrInvalidGovernanceMutation
		}
		return boolean, nil
	case SafeNonNegativeInteger:
		integer, ok := governanceInteger(value)
		if !ok || integer < 0 {
			return nil, contract.ErrInvalidGovernanceMutation
		}
		return integer, nil
	case SafeSHA256:
		text, ok := value.(string)
		decoded, err := hex.DecodeString(text)
		if !ok || err != nil || len(decoded) != sha256.Size || text != strings.ToLower(text) {
			return nil, contract.ErrInvalidGovernanceMutation
		}
		return text, nil
	default:
		return nil, contract.ErrInvalidGovernanceMutation
	}
}

func governanceInteger(value any) (int64, bool) {
	switch typed := value.(type) {
	case int:
		return int64(typed), true
	case int32:
		return int64(typed), true
	case int64:
		return typed, true
	case uint:
		if uint64(typed) > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(typed), true
	case json.Number:
		integer, err := strconv.ParseInt(string(typed), 10, 64)
		return integer, err == nil
	default:
		return 0, false
	}
}

func containsExact(values []string, candidate string) bool {
	for _, value := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func encodeGovernanceEnvelope(version string, schema EnvelopeSchema, fields map[string]any) (json.RawMessage, error) {
	canonical, err := canonicalGovernanceFields(schema, fields)
	if err != nil {
		return nil, err
	}
	envelope := struct {
		Version string         `json:"version"`
		Data    map[string]any `json:"data"`
	}{Version: version, Data: canonical}
	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("encode governance envelope: %w", err)
	}
	return encoded, nil
}

func validateGovernanceEnvelopeFields(version string, schema EnvelopeSchema, raw json.RawMessage) error {
	var envelope struct {
		Version string         `json:"version"`
		Data    map[string]any `json:"data"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return fmt.Errorf("%w: invalid governance envelope", contract.ErrInvalidGovernanceMutation)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF || envelope.Version != version || envelope.Data == nil {
		return fmt.Errorf("%w: invalid governance envelope", contract.ErrInvalidGovernanceMutation)
	}
	_, err := canonicalGovernanceFields(schema, envelope.Data)
	return err
}

func governanceRequestHash(action string, schema EnvelopeSchema, fields map[string]any) (string, error) {
	canonical, err := canonicalGovernanceFields(schema, fields)
	if err != nil {
		return "", err
	}
	preimage := struct {
		Version string         `json:"version"`
		Action  string         `json:"action"`
		Request map[string]any `json:"request"`
	}{Version: "governance-request-v1", Action: action, Request: canonical}
	encoded, err := json.Marshal(preimage)
	if err != nil {
		return "", fmt.Errorf("encode canonical governance request: %w", err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}
