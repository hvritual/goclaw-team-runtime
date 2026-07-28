package catalog

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const maxMarkdownBytes = 2 * 1024 * 1024

var sourceSchemePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]*$`)

func (s *Service) IngestPath(path string, options IngestOptions) (IngestReport, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return IngestReport{}, err
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return IngestReport{}, err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return IngestReport{}, errors.New("catalog ingestion does not follow symbolic links")
	}
	if options.SourceRoot == "" {
		if info.IsDir() {
			options.SourceRoot = absolute
		} else {
			options.SourceRoot = filepath.Dir(absolute)
		}
	}
	options.SourceRoot, err = filepath.Abs(options.SourceRoot)
	if err != nil {
		return IngestReport{}, err
	}
	if options.SourceScheme == "" {
		options.SourceScheme = "markdown"
	}
	if !sourceSchemePattern.MatchString(options.SourceScheme) {
		return IngestReport{}, errors.New("catalog source scheme is invalid")
	}
	if options.SourceKind == "" {
		switch options.SourceScheme {
		case "markdown":
			options.SourceKind = "markdown"
		case "git+markdown":
			options.SourceKind = "git-markdown"
		default:
			options.SourceKind = options.SourceScheme + "-markdown"
		}
	}
	if options.SourceRevision == "" && options.SourceScheme == "git+markdown" {
		options.SourceRevision, err = gitSourceRevision(options.SourceRoot)
		if err != nil {
			return IngestReport{}, err
		}
	}
	report := IngestReport{}
	if !info.IsDir() {
		err := s.ingestMarkdownFile(absolute, options, &report)
		return report, err
	}
	err = filepath.WalkDir(absolute, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			report.Failed++
			report.Errors = append(report.Errors, walkErr.Error())
			return nil
		}
		if entry.IsDir() {
			if current != absolute && skipIngestDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			report.Failed++
			report.Errors = append(report.Errors, "skipped symlink: "+current)
			return nil
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".md") {
			if err := s.ingestMarkdownFile(current, options, &report); err != nil {
				report.Failed++
				report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", current, err))
			}
		}
		return nil
	})
	return report, err
}

func (s *Service) ingestMarkdownFile(
	path string,
	options IngestOptions,
	report *IngestReport,
) error {
	report.Scanned++
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.New("symbolic links are not allowed")
	}
	if info.Size() > maxMarkdownBytes {
		return fmt.Errorf("file exceeds %d-byte limit", maxMarkdownBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	metadata, content, err := parseMarkdown(data)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(options.SourceRoot, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return errors.New("source file is outside source root")
	}
	rel = filepath.ToSlash(rel)
	projectID := firstNonEmpty(
		stringValue(metadata["project_id"]),
		stringValue(nestedValue(metadata, "goclaw", "project_id")),
		options.ProjectID,
	)
	if projectID == "" {
		projectID = s.cfg.DefaultProject
	}
	kind := RecordKind(firstNonEmpty(
		stringValue(metadata["type"]),
		stringValue(nestedValue(metadata, "goclaw", "kind")),
		string(options.DefaultKind),
		string(kindFromPath(rel)),
	))
	if !validKind(kind) {
		kind = KindFact
	}
	title := firstNonEmpty(
		stringValue(metadata["title"]),
		firstTitle(content),
		strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)),
	)
	abstract := firstNonEmpty(
		stringValue(metadata["description"]),
		stringValue(metadata["abstract"]),
	)
	subjects := stringSliceValue(metadata["subject"])
	if len(subjects) == 0 {
		subjects = stringSliceValue(metadata["tags"])
	}
	facets := mapStringSliceValue(nestedValue(metadata, "goclaw", "facets"))
	authorities := stringSliceValue(nestedValue(metadata, "goclaw", "authorities"))
	evidence := stringSliceValue(nestedValue(metadata, "goclaw", "evidence_refs"))
	confidence := floatValue(nestedValue(metadata, "goclaw", "confidence"))
	if confidence == 0 {
		confidence = 0.7
	}
	sourceURI := firstNonEmpty(
		stringValue(metadata["source"]),
		fmt.Sprintf("%s://%s/%s", options.SourceScheme, projectID, rel),
	)
	modTime := info.ModTime().UTC()
	input := CreateInput{
		ProjectID:    projectID,
		Collection:   firstNonEmpty(options.Collection, "knowledge-markdown"),
		WorkID:       stringValue(nestedValue(metadata, "goclaw", "work_id")),
		ExpressionID: stringValue(nestedValue(metadata, "goclaw", "expression_id")),
		Title:        title,
		Abstract:     abstract,
		Content:      content,
		Kind:         kind,
		Language:     firstNonEmpty(stringValue(metadata["language"]), "und"),
		Subjects:     subjects,
		Facets:       facets,
		AuthorityIDs: authorities,
		EvidenceRefs: evidence,
		Confidence:   confidence,
		ValidFrom:    timeValue(nestedValue(metadata, "goclaw", "valid_from")),
		ValidUntil:   timeValue(nestedValue(metadata, "goclaw", "valid_until")),
		ReviewAt:     timeValue(nestedValue(metadata, "goclaw", "review_at")),
		ExpiresAt:    timeValue(nestedValue(metadata, "goclaw", "expires_at")),
		CreatedBy:    firstNonEmpty(options.Actor, "catalog-importer"),
		Provenance: Provenance{
			SourceURI:      sourceURI,
			SourceKind:     options.SourceKind,
			SourceRevision: firstNonEmpty(options.SourceRevision, modTime.Format(time.RFC3339Nano)),
			CapturedAt:     time.Now().UTC(),
			AgentID:        firstNonEmpty(options.Actor, "catalog-importer"),
			TraceID:        stringValue(nestedValue(metadata, "goclaw", "trace_id")),
		},
	}
	record, created, err := s.CreateCandidate(input)
	if err != nil {
		return err
	}
	report.Records = append(report.Records, record.ID)
	if created {
		report.Created++
	} else {
		report.Existing++
	}
	return nil
}

func gitSourceRevision(root string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(
		ctx,
		"git",
		"-C",
		root,
		"rev-parse",
		"--verify",
		"HEAD",
	).Output()
	if err != nil {
		return "", fmt.Errorf("resolve Git Markdown source revision: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func parseMarkdown(data []byte) (map[string]any, string, error) {
	raw := strings.ReplaceAll(string(data), "\r\n", "\n")
	metadata := make(map[string]any)
	if !strings.HasPrefix(raw, "---\n") {
		return metadata, strings.TrimSpace(raw), nil
	}
	remainder := raw[4:]
	index := strings.Index(remainder, "\n---\n")
	if index < 0 {
		return nil, "", errors.New("unterminated YAML frontmatter")
	}
	if err := yaml.Unmarshal([]byte(remainder[:index]), &metadata); err != nil {
		return nil, "", fmt.Errorf("parse YAML frontmatter: %w", err)
	}
	return normalizeYAMLMap(metadata), strings.TrimSpace(remainder[index+5:]), nil
}

func normalizeYAMLMap(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		switch typed := value.(type) {
		case map[string]any:
			result[key] = normalizeYAMLMap(typed)
		case map[any]any:
			converted := make(map[string]any, len(typed))
			for nestedKey, nestedValue := range typed {
				converted[fmt.Sprint(nestedKey)] = nestedValue
			}
			result[key] = normalizeYAMLMap(converted)
		default:
			result[key] = value
		}
	}
	return result
}

func nestedValue(input map[string]any, path ...string) any {
	var current any = input
	for _, key := range path {
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = asMap[key]
	}
	return current
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return strings.TrimSpace(typed.String())
	default:
		return ""
	}
}

func stringSliceValue(value any) []string {
	switch typed := value.(type) {
	case string:
		if strings.Contains(typed, ",") {
			return cleanStrings(strings.Split(typed, ","))
		}
		return cleanStrings([]string{typed})
	case []string:
		return cleanStrings(typed)
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			if value := stringValue(item); value != "" {
				result = append(result, value)
			}
		}
		return cleanStrings(result)
	default:
		return nil
	}
}

func mapStringSliceValue(value any) map[string][]string {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	result := make(map[string][]string)
	for key, item := range raw {
		if values := stringSliceValue(item); len(values) > 0 {
			result[key] = values
		}
	}
	return result
}

func floatValue(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case float32:
		return float64(typed)
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed
	default:
		return 0
	}
}

func timeValue(value any) *time.Time {
	raw := stringValue(value)
	if raw == "" {
		return nil
	}
	formats := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"}
	for _, format := range formats {
		if parsed, err := time.Parse(format, raw); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}

func kindFromPath(path string) RecordKind {
	path = filepath.ToSlash(path)
	switch {
	case strings.HasPrefix(path, "01-goals/"):
		return KindGoal
	case strings.HasPrefix(path, "02-decisions/"):
		return KindDecision
	case strings.HasPrefix(path, "03-constraints/"):
		return KindConstraint
	case strings.HasPrefix(path, "04-requirements/"):
		return KindRequirement
	case strings.HasPrefix(path, "05-knowledge/"):
		return KindFact
	case strings.Contains(filepath.Base(path), "MEMORY.md"):
		return KindFact
	default:
		name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		if _, err := time.Parse("2006-01-02", name); err == nil {
			return KindContext
		}
		return KindFact
	}
}

func skipIngestDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", ".obsidian", ".trash", "node_modules", "dist", "vendor":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
