package teamcontrol

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/url"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrNotFound          = errors.New("not found")
	ErrForbidden         = errors.New("forbidden")
	ErrAuthentication    = errors.New("authentication failed")
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid transition")
)

var idPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:-]{0,127}$`)
var keyPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)
var gitCommitPattern = regexp.MustCompile(`^[0-9a-fA-F]{7,64}$`)
var windowsAbsolutePathPattern = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

func newID(prefix string) string {
	return prefix + "-" + uuid.NewString()
}

func normalizeID(value, prefix string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = newID(prefix)
	}
	if !idPattern.MatchString(value) {
		return "", fmt.Errorf("invalid id %q", value)
	}
	return value, nil
}

func requireID(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if !idPattern.MatchString(value) {
		return "", fmt.Errorf("%s must be a valid id", field)
	}
	return value, nil
}

func requireKey(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if !keyPattern.MatchString(value) {
		return "", fmt.Errorf("%s must match %s", field, keyPattern)
	}
	return value, nil
}

func requireText(value, field string, maximum int) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if maximum > 0 && utf8.RuneCountInString(value) > maximum {
		return "", fmt.Errorf("%s exceeds %d characters", field, maximum)
	}
	return value, nil
}

func optionalText(value, field string, maximum int) (string, error) {
	value = strings.TrimSpace(value)
	if maximum > 0 && utf8.RuneCountInString(value) > maximum {
		return "", fmt.Errorf("%s exceeds %d characters", field, maximum)
	}
	return value, nil
}

func validateEmail(value string) (string, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "", nil
	}
	address, err := mail.ParseAddress(value)
	if err != nil || !strings.EqualFold(address.Address, value) {
		return "", fmt.Errorf("invalid email address")
	}
	return value, nil
}

func validateURI(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%s is invalid: %w", field, err)
	}
	if parsed.Scheme == "" && !strings.HasPrefix(value, "/") && !strings.HasPrefix(value, ".") {
		return "", fmt.Errorf("%s must be an absolute path or URI with a scheme", field)
	}
	return value, nil
}

// validateRegistryURI accepts only locations that can be fetched without
// embedding authority material. Query strings and fragments are intentionally
// rejected because signed URLs and access tokens must never enter durable
// control-plane state.
func validateRegistryURI(value, field string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	if strings.ContainsAny(value, "?#") {
		return "", fmt.Errorf("%s must not contain query or fragment data", field)
	}
	if filepath.IsAbs(value) || windowsAbsolutePathPattern.MatchString(value) {
		return value, nil
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", fmt.Errorf("%s is invalid: %w", field, err)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		parsed.Opaque != "" {
		return "", fmt.Errorf("%s must not contain credentials, query, fragment, or opaque data", field)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "file":
		if parsed.Host != "" && !strings.EqualFold(parsed.Host, "localhost") {
			return "", fmt.Errorf("%s file URI must be local", field)
		}
		if !filepath.IsAbs(parsed.Path) {
			return "", fmt.Errorf("%s file URI must contain an absolute path", field)
		}
	case "https", "git+https":
		if parsed.Host == "" {
			return "", fmt.Errorf("%s must contain a host", field)
		}
	default:
		return "", fmt.Errorf("%s uses unsupported scheme %q", field, parsed.Scheme)
	}
	return value, nil
}

func cleanUsageMetadata(values map[string]string) (map[string]string, error) {
	allowed := map[string]bool{
		"model": true, "provider": true, "operation": true,
	}
	return cleanTypedMetadata(values, allowed, func(key, value string) (string, error) {
		return requireKey(value, "usage metadata "+key)
	})
}

func cleanRegistryMetadata(values map[string]string) (map[string]string, error) {
	allowed := map[string]bool{
		"content_type":  true,
		"license":       true,
		"source_kind":   true,
		"repository_id": true,
		"owner_id":      true,
		"secret_ref":    true,
		"visibility":    true,
		"language":      true,
		"category":      true,
	}
	return cleanTypedMetadata(values, allowed, func(key, value string) (string, error) {
		switch key {
		case "repository_id", "owner_id", "secret_ref":
			return requireID(value, "registry metadata "+key)
		case "visibility":
			value = strings.ToLower(strings.TrimSpace(value))
			if value != "project" && value != "team" {
				return "", fmt.Errorf("registry metadata visibility must be project or team")
			}
			return value, nil
		case "source_kind":
			value = strings.ToLower(strings.TrimSpace(value))
			switch value {
			case "obsidian", "git", "file", "package", "documentation":
				return value, nil
			default:
				return "", fmt.Errorf("unsupported registry metadata source_kind %q", value)
			}
		case "content_type":
			value, err := requireText(value, "registry metadata content_type", 100)
			if err != nil {
				return "", err
			}
			if strings.ContainsAny(value, "\r\n") {
				return "", fmt.Errorf("registry metadata content_type contains a line break")
			}
			return value, nil
		default:
			return requireKey(value, "registry metadata "+key)
		}
	})
}

func cleanTypedMetadata(
	values map[string]string,
	allowed map[string]bool,
	validate func(key, value string) (string, error),
) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make(map[string]string, len(values))
	for rawKey, rawValue := range values {
		key := strings.ToLower(strings.TrimSpace(rawKey))
		if !allowed[key] {
			return nil, fmt.Errorf("unsupported metadata key %q", rawKey)
		}
		value, err := validate(key, rawValue)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func validateOptionalSHA256(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	decoded, err := hex.DecodeString(value)
	if err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("sha256 must be 64 lowercase hexadecimal characters")
	}
	return value, nil
}

func validateRelativePath(value, field string) (string, error) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	clean := filepath.ToSlash(filepath.Clean(value))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || filepath.IsAbs(value) {
		return "", fmt.Errorf("%s must remain inside its repository", field)
	}
	return clean, nil
}

func uniqueIDs(values []string, field string) ([]string, error) {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		id, err := requireID(value, field)
		if err != nil {
			return nil, err
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	slices.Sort(result)
	return result, nil
}

func cleanLabels(values []string) ([]string, error) {
	if len(values) > 100 {
		return nil, fmt.Errorf("labels exceed 100 values")
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		label, err := requireID(strings.ToLower(value), "label")
		if err != nil {
			return nil, err
		}
		if _, exists := seen[label]; exists {
			continue
		}
		seen[label] = struct{}{}
		result = append(result, label)
	}
	slices.Sort(result)
	return result, nil
}

func normalizeBusinessDomain(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	return requireText(value, "business_domain", 200)
}

func cleanBusinessDomains(values []string) ([]string, error) {
	if len(values) > 100 {
		return nil, fmt.Errorf("business_domains exceed 100 values")
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		domain, err := normalizeBusinessDomain(value)
		if err != nil {
			return nil, err
		}
		if domain == "" {
			return nil, fmt.Errorf("business_domain is required")
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		result = append(result, domain)
	}
	slices.Sort(result)
	return result, nil
}

func validateGitCommit(value, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if !gitCommitPattern.MatchString(value) {
		return "", fmt.Errorf("%s must be a 7-64 character hexadecimal commit id", field)
	}
	return value, nil
}

func cleanStringMap(values map[string]string) (map[string]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	result := make(map[string]string, len(values))
	for key, value := range values {
		key, err := requireKey(key, "metadata key")
		if err != nil {
			return nil, err
		}
		value, err = optionalText(value, "metadata value", 4096)
		if err != nil {
			return nil, err
		}
		result[key] = value
	}
	return result, nil
}

func validUserStatus(value UserStatus) bool {
	return value == UserActive || value == UserDisabled
}

func validTeamRole(value TeamRole) bool {
	return value == TeamOwner || value == TeamAdmin || value == TeamRegularMember
}

func validProjectRole(value ProjectRole) bool {
	switch value {
	case ProjectOwner, ProjectMaintainer, ProjectDeveloper, ProjectReviewer, ProjectViewer:
		return true
	default:
		return false
	}
}

func validIssueType(value IssueType) bool {
	switch value {
	case IssueBug, IssueTask, IssueImprovement, IssueRisk:
		return true
	default:
		return false
	}
}

func validSeverity(value IssueSeverity) bool {
	switch value {
	case SeverityCritical, SeverityHigh, SeverityMedium, SeverityLow:
		return true
	default:
		return false
	}
}

func validPriority(value IssuePriority) bool {
	switch value {
	case PriorityP0, PriorityP1, PriorityP2, PriorityP3, PriorityP4:
		return true
	default:
		return false
	}
}

func validArtifactKind(value ArtifactKind) bool {
	switch value {
	case ArtifactDiff, ArtifactEvidence, ArtifactBuild, ArtifactLog, ArtifactReport,
		ArtifactTrace, ArtifactCommit, ArtifactPR, ArtifactPackage, ArtifactOther:
		return true
	default:
		return false
	}
}

func validResourceType(value ResourceType) bool {
	switch value {
	case ResourceIssue, ResourceTask, ResourceWorkItem, ResourceRun, ResourceTrace,
		ResourceCommit, ResourcePullRequest, ResourceCI, ResourceRelease,
		ResourceRegressionCase, ResourceSpec, ResourceArtifact, ResourceDocument,
		ResourceComponent, ResourceRepository, ResourcePolicy:
		return true
	default:
		return false
	}
}

func validDocumentKind(value DocumentKind) bool {
	switch value {
	case DocumentPRD, DocumentADR, DocumentDesign, DocumentRunbook, DocumentAPI,
		DocumentTestPlan, DocumentReport, DocumentKnowledge, DocumentOther:
		return true
	default:
		return false
	}
}

func validComponentKind(value ComponentKind) bool {
	switch value {
	case ComponentService, ComponentLibrary, ComponentApp, ComponentModule,
		ComponentDevice, ComponentOther:
		return true
	default:
		return false
	}
}

func validPolicyScope(value PolicyScope) bool {
	switch value {
	case PolicyTeam, PolicyProject, PolicyRepository, PolicyComponent:
		return true
	default:
		return false
	}
}

func validRegistryStatus(value RegistryStatus) bool {
	switch value {
	case RegistryDraft, RegistryApproved, RegistryDisabled:
		return true
	default:
		return false
	}
}

func entityNotFound(kind, id string) error {
	return fmt.Errorf("%w: %s %q", ErrNotFound, kind, id)
}

func conflict(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, args...))
}
