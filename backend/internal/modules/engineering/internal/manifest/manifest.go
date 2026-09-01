package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
	"gopkg.in/yaml.v3"
)

const (
	SchemaVersionV1  = "v1"
	MaxManifestBytes = 256 << 10
)

var (
	ErrInvalidManifest             = errors.New("invalid engineering manifest")
	ErrUnsupportedSchemaVersion    = errors.New("unsupported engineering manifest schema version")
	ErrUnsupportedSource           = errors.New("unsupported engineering manifest source")
	ErrInvalidSourceLocator        = errors.New("invalid engineering manifest source locator")
	ErrDuplicateValue              = errors.New("duplicate engineering manifest value")
	ErrUnsupportedManifestRelation = errors.New("unsupported engineering manifest relation")
	ErrInvalidInterface            = errors.New("invalid engineering manifest interface")
	ErrUnsafeKnowledgePath         = errors.New("unsafe engineering manifest knowledge path")
)

type Manifest struct {
	SchemaVersion string      `yaml:"schema_version" json:"schema_version"`
	Entity        Entity      `yaml:"entity" json:"entity"`
	Source        Source      `yaml:"source" json:"source"`
	Domains       []string    `yaml:"domains,omitempty" json:"domains,omitempty"`
	Dependencies  []string    `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	Relations     []Relation  `yaml:"relations,omitempty" json:"relations,omitempty"`
	Interfaces    []Interface `yaml:"interfaces,omitempty" json:"interfaces,omitempty"`
	Knowledge     Knowledge   `yaml:"knowledge,omitempty" json:"knowledge"`
}

type Entity struct {
	ID       string `yaml:"id" json:"id"`
	Type     string `yaml:"type" json:"type"`
	Name     string `yaml:"name" json:"name"`
	Status   string `yaml:"status" json:"status"`
	OwnerRef string `yaml:"owner_ref,omitempty" json:"owner_ref,omitempty"`
}

type Source struct {
	Type    string `yaml:"type" json:"type"`
	Locator string `yaml:"locator" json:"locator"`
}

type Relation struct {
	Relation string `yaml:"relation" json:"relation"`
	Target   string `yaml:"target" json:"target"`
}

type Interface struct {
	ID        string `yaml:"id" json:"id"`
	Type      string `yaml:"type" json:"type"`
	Direction string `yaml:"direction" json:"direction"`
}

type Knowledge struct {
	Architecture    []string `yaml:"architecture,omitempty" json:"architecture,omitempty"`
	ADR             []string `yaml:"adr,omitempty" json:"adr,omitempty"`
	Standards       []string `yaml:"standards,omitempty" json:"standards,omitempty"`
	Runbooks        []string `yaml:"runbooks,omitempty" json:"runbooks,omitempty"`
	Troubleshooting []string `yaml:"troubleshooting,omitempty" json:"troubleshooting,omitempty"`
}

var manifestRelations = map[domain.RelationType]struct{}{
	domain.RelationPartOf:     {},
	domain.RelationImplements: {},
	domain.RelationConstrains: {},
	domain.RelationGoverns:    {},
	domain.RelationOperates:   {},
}

func Parse(data []byte) (Manifest, error) {
	if len(data) == 0 || len(data) > MaxManifestBytes {
		return Manifest{}, fmt.Errorf("%w: document size %d", ErrInvalidManifest, len(data))
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var value Manifest
	if err := decoder.Decode(&value); err != nil {
		if errors.Is(err, io.EOF) {
			return Manifest{}, fmt.Errorf("%w: empty document", ErrInvalidManifest)
		}
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Manifest{}, fmt.Errorf("%w: multiple YAML documents", ErrInvalidManifest)
		}
		return Manifest{}, fmt.Errorf("%w: %v", ErrInvalidManifest, err)
	}
	return Normalize(value)
}

func Normalize(value Manifest) (Manifest, error) {
	value.SchemaVersion = strings.TrimSpace(value.SchemaVersion)
	if value.SchemaVersion != SchemaVersionV1 {
		return Manifest{}, fmt.Errorf("%w: %q", ErrUnsupportedSchemaVersion, value.SchemaVersion)
	}

	value.Entity.ID = strings.TrimSpace(value.Entity.ID)
	value.Entity.Type = strings.TrimSpace(value.Entity.Type)
	value.Entity.Name = strings.TrimSpace(value.Entity.Name)
	value.Entity.Status = strings.TrimSpace(value.Entity.Status)
	value.Entity.OwnerRef = strings.TrimSpace(value.Entity.OwnerRef)
	if value.Entity.Status == "" {
		return Manifest{}, fmt.Errorf("%w: entity status is required", ErrInvalidManifest)
	}
	if _, err := domain.NewEngineeringEntity(
		value.Entity.ID,
		"manifest-validation",
		domain.EntityType(value.Entity.Type),
		value.Entity.Name,
		domain.EntityStatus(value.Entity.Status),
		value.Entity.OwnerRef,
	); err != nil {
		return Manifest{}, fmt.Errorf("%w: entity: %v", ErrInvalidManifest, err)
	}

	value.Source.Type = strings.ToLower(strings.TrimSpace(value.Source.Type))
	if value.Source.Type != "github" {
		return Manifest{}, fmt.Errorf("%w: %q", ErrUnsupportedSource, value.Source.Type)
	}
	locator, err := normalizeGitHubLocator(value.Source.Locator)
	if err != nil {
		return Manifest{}, err
	}
	value.Source.Locator = locator

	value.Domains, err = normalizeUniqueStrings(value.Domains, "domain", true)
	if err != nil {
		return Manifest{}, err
	}
	value.Dependencies, err = normalizeUniqueStrings(value.Dependencies, "dependency", false)
	if err != nil {
		return Manifest{}, err
	}
	for _, dependency := range value.Dependencies {
		if dependency == value.Entity.ID {
			return Manifest{}, fmt.Errorf("%w: dependency %q points to the manifest entity", ErrInvalidManifest, dependency)
		}
	}
	value.Relations, err = normalizeRelations(value.Entity.ID, value.Relations)
	if err != nil {
		return Manifest{}, err
	}
	value.Interfaces, err = normalizeInterfaces(value.Interfaces)
	if err != nil {
		return Manifest{}, err
	}
	if value.Knowledge, err = normalizeKnowledge(value.Knowledge); err != nil {
		return Manifest{}, err
	}
	return value, nil
}

func (value Manifest) Checksum() string {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeGitHubLocator(raw string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(parsed.Scheme, "github") || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidSourceLocator, raw)
	}
	owner := strings.TrimSpace(parsed.Host)
	repository := strings.Trim(strings.TrimSpace(parsed.Path), "/")
	if owner == "" || repository == "" || strings.Contains(repository, "/") || strings.ContainsAny(owner+repository, " \t\r\n") {
		return "", fmt.Errorf("%w: %q", ErrInvalidSourceLocator, raw)
	}
	repository = strings.TrimSuffix(repository, ".git")
	if repository == "" {
		return "", fmt.Errorf("%w: %q", ErrInvalidSourceLocator, raw)
	}
	return "github://" + strings.ToLower(owner) + "/" + strings.ToLower(repository), nil
}

func normalizeUniqueStrings(values []string, label string, lower bool) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if lower {
			value = strings.ToLower(value)
		}
		if value == "" {
			return nil, fmt.Errorf("%w: empty %s", ErrInvalidManifest, label)
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("%w: %s %q", ErrDuplicateValue, label, value)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeRelations(entityID string, values []Relation) ([]Relation, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]Relation, 0, len(values))
	for _, raw := range values {
		relation := domain.RelationType(strings.TrimSpace(raw.Relation))
		target := strings.TrimSpace(raw.Target)
		if !relation.Valid() {
			return nil, fmt.Errorf("%w: relation %q", ErrInvalidManifest, raw.Relation)
		}
		if _, allowed := manifestRelations[relation]; !allowed {
			return nil, fmt.Errorf("%w: %s", ErrUnsupportedManifestRelation, relation)
		}
		if target == "" || target == entityID {
			return nil, fmt.Errorf("%w: relation target %q", ErrInvalidManifest, target)
		}
		key := string(relation) + "\x00" + target
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: relation %s -> %s", ErrDuplicateValue, relation, target)
		}
		seen[key] = struct{}{}
		result = append(result, Relation{Relation: string(relation), Target: target})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Relation == result[j].Relation {
			return result[i].Target < result[j].Target
		}
		return result[i].Relation < result[j].Relation
	})
	return result, nil
}

func normalizeInterfaces(values []Interface) ([]Interface, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]Interface, 0, len(values))
	for _, raw := range values {
		value := Interface{
			ID:        strings.TrimSpace(raw.ID),
			Type:      strings.TrimSpace(raw.Type),
			Direction: strings.TrimSpace(raw.Direction),
		}
		if value.ID == "" || (value.Type != string(domain.EntityTypeAPI) && value.Type != string(domain.EntityTypeThingModel)) || (value.Direction != "provides" && value.Direction != "uses") {
			return nil, fmt.Errorf("%w: %+v", ErrInvalidInterface, raw)
		}
		if _, exists := seen[value.ID]; exists {
			return nil, fmt.Errorf("%w: interface id %s", ErrDuplicateValue, value.ID)
		}
		seen[value.ID] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Direction != result[j].Direction {
			return result[i].Direction < result[j].Direction
		}
		if result[i].Type != result[j].Type {
			return result[i].Type < result[j].Type
		}
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func normalizeKnowledge(value Knowledge) (Knowledge, error) {
	var err error
	if value.Architecture, err = normalizeKnowledgePaths(value.Architecture, "architecture"); err != nil {
		return Knowledge{}, err
	}
	if value.ADR, err = normalizeKnowledgePaths(value.ADR, "adr"); err != nil {
		return Knowledge{}, err
	}
	if value.Standards, err = normalizeKnowledgePaths(value.Standards, "standards"); err != nil {
		return Knowledge{}, err
	}
	if value.Runbooks, err = normalizeKnowledgePaths(value.Runbooks, "runbooks"); err != nil {
		return Knowledge{}, err
	}
	if value.Troubleshooting, err = normalizeKnowledgePaths(value.Troubleshooting, "troubleshooting"); err != nil {
		return Knowledge{}, err
	}
	return value, nil
}

func normalizeKnowledgePaths(values []string, label string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, raw := range values {
		value := strings.TrimSpace(raw)
		if value == "" || strings.Contains(value, "\\") || strings.Contains(value, "://") || path.IsAbs(value) {
			return nil, fmt.Errorf("%w: %s %q", ErrUnsafeKnowledgePath, label, raw)
		}
		for _, segment := range strings.Split(value, "/") {
			if segment == "" || segment == "." || segment == ".." {
				return nil, fmt.Errorf("%w: %s %q", ErrUnsafeKnowledgePath, label, raw)
			}
		}
		clean := path.Clean(value)
		if clean != value || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
			return nil, fmt.Errorf("%w: %s %q", ErrUnsafeKnowledgePath, label, raw)
		}
		if _, exists := seen[clean]; exists {
			return nil, fmt.Errorf("%w: %s %q", ErrDuplicateValue, label, clean)
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}
	sort.Strings(result)
	return result, nil
}
