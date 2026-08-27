package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

type issueSimilarityRepository struct{ db *sql.DB }

const issueSimilarityCandidatePoolLimit = 50

func NewIssueSimilarityRepository(config Config) (application.IssueSimilarityRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &issueSimilarityRepository{db: config.DB}, nil
}

func (r *issueSimilarityRepository) FindIssueSimilarityCandidates(ctx context.Context, query application.IssueSimilarityQuery) ([]application.IssueSimilarityCandidate, bool, error) {
	limit := query.Limit
	if limit <= 0 || limit > issueSimilarityCandidatePoolLimit {
		limit = issueSimilarityCandidatePoolLimit
	}
	titleTerms := strings.Fields(query.Title)
	descriptionTerms := strings.Fields(query.Description)
	titlePredicate := issueSimilarityTermPredicate("d.title", len(titleTerms))
	descriptionPredicate := issueSimilarityTermPredicate("d.description", len(descriptionTerms))
	matchPredicates := []string{`d.identifier=?`, `d.title=?`, `(` + titlePredicate + `)`}
	args := []any{query.WorkspaceID, query.Title, query.Title}
	for _, term := range titleTerms {
		args = append(args, term)
	}
	if descriptionPredicate != "" {
		matchPredicates = append(matchPredicates, `(`+descriptionPredicate+`)`)
		for _, term := range descriptionTerms {
			args = append(args, term)
		}
	}
	where := `d.workspace_id=? AND (` + strings.Join(matchPredicates, ` OR `) + `)`
	if !query.IncludeClosed {
		where += ` AND d.status NOT IN ('done','cancelled')`
	}
	if query.ExcludeIssueID != "" {
		where += ` AND i.id<>?`
		args = append(args, query.ExcludeIssueID)
	}
	args = append(args, query.Title, query.Title, query.ProjectID, query.ProjectID, limit)
	rows, err := r.db.QueryContext(ctx, `SELECT `+qualifiedIssueColumns+`
		FROM workspace_issue_search_documents d
		JOIN workspace_issues i ON i.id=d.issue_id AND i.workspace_id=d.workspace_id
		WHERE `+where+`
		ORDER BY CASE WHEN d.identifier=? THEN 0 WHEN d.title=? THEN 1 ELSE 2 END,
		CASE WHEN ?<>'' AND i.project_id=? THEN 0 ELSE 1 END,
		d.updated_at DESC, d.issue_id ASC
		LIMIT ?`, args...)
	if err != nil {
		return nil, false, fmt.Errorf("query Issue similarity candidates: %w", err)
	}
	defer rows.Close()
	values := make([]application.IssueSimilarityCandidate, 0)
	for rows.Next() {
		issue, err := scanIssue(rows)
		if err != nil {
			return nil, false, err
		}
		score, componentScores, sameProject := issueSimilarityScore(query, issue)
		values = append(values, application.IssueSimilarityCandidate{
			Issue: issue, Score: score, ComponentScores: componentScores, SameProject: sameProject,
			Closed: issue.Status == "done" || issue.Status == "cancelled",
		})
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("iterate Issue similarity candidates: %w", err)
	}
	sort.SliceStable(values, func(left, right int) bool {
		return values[left].Score > values[right].Score
	})
	return values, false, nil
}

func issueSimilarityScore(query application.IssueSimilarityQuery, issue issueDomain.Issue) (int, map[string]float64, bool) {
	components := map[string]float64{
		"identifier":             0,
		"exact_normalized_title": 0,
		"title_terms":            0,
		"description_overlap":    0,
		"same_project":           0,
	}
	score := 0
	if application.NormalizeIssueSearchText(issue.Identifier) == query.Title {
		components["identifier"] = 1
		score += 120
	}
	normalizedTitle := application.NormalizeIssueSearchText(issue.Title)
	if normalizedTitle == query.Title {
		components["exact_normalized_title"] = 1
		score += 100
	} else if issueSimilarityHasTerms(normalizedTitle, strings.Fields(query.Title)) {
		components["title_terms"] = 1
		score += 70
	}
	if issue.Description != nil && issueSimilarityHasTerms(application.NormalizeIssueSearchText(*issue.Description), strings.Fields(query.Description)) {
		components["description_overlap"] = 1
		score += 40
	}
	sameProject := query.ProjectID != "" && issue.ProjectID != nil && *issue.ProjectID == query.ProjectID
	if sameProject {
		components["same_project"] = 1
		score += 10
	}
	return score, components, sameProject
}

func issueSimilarityHasTerms(value string, terms []string) bool {
	if len(terms) == 0 {
		return false
	}
	for _, term := range terms {
		if !strings.Contains(value, term) {
			return false
		}
	}
	return true
}

func issueSimilarityTermPredicate(column string, count int) string {
	if count == 0 {
		return "0"
	}
	parts := make([]string, count)
	for index := range parts {
		parts[index] = `instr(` + column + `, ?) > 0`
	}
	return strings.Join(parts, ` AND `)
}

var _ application.IssueSimilarityRepository = (*issueSimilarityRepository)(nil)
