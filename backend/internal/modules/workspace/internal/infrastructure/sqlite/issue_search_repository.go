package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type issueSearchRepository struct{ db *sql.DB }

func NewIssueSearchRepository(config Config) (application.IssueSearchRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	repository := &issueSearchRepository{db: config.DB}
	if err := repository.rebuild(context.Background()); err != nil {
		return nil, err
	}
	return repository, nil
}

func (r *issueSearchRepository) rebuild(ctx context.Context) (err error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Issue search projection rebuild: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err = tx.ExecContext(ctx, `DELETE FROM workspace_issue_search_documents`); err != nil {
		return fmt.Errorf("clear Issue search projection: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_issue_search_documents(
		issue_id,workspace_id,identifier,number,title,description,status,updated_at
	) SELECT id,workspace_id,goclaw_issue_search_normalize(identifier),number,
		goclaw_issue_search_normalize(title),goclaw_issue_search_normalize(COALESCE(description,'')),status,updated_at
		FROM workspace_issues`); err != nil {
		return fmt.Errorf("rebuild Issue search projection: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit Issue search projection rebuild: %w", err)
	}
	return nil
}

const qualifiedIssueColumns = `i.id, i.workspace_id, i.number, i.identifier, i.title, i.description, i.status, i.priority,
	i.assignee_type, i.assignee_id, i.creator_type, i.creator_id, i.parent_issue_id, i.project_id, i.position,
	i.stage, i.start_date, i.due_date, i.created_at, i.updated_at, i.metadata, i.properties, i.asset_ids`

func (r *issueSearchRepository) SearchIssues(ctx context.Context, query application.IssueSearchQuery) ([]application.IssueSearchResult, int, error) {
	matchSQL, rankSQL, sourceSQL, matchArgs, rankArgs := issueSearchPredicates(query)
	closedSQL := ""
	if !query.IncludeClosed {
		closedSQL = ` AND d.status NOT IN ('done','cancelled')`
	}
	var total int
	countArgs := append([]any{query.WorkspaceID}, matchArgs...)
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_issue_search_documents d
		WHERE d.workspace_id=?`+closedSQL+` AND (`+matchSQL+`)`, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count Issue search results: %w", err)
	}
	selectArgs := append([]any{}, rankArgs...)
	selectArgs = append(selectArgs, query.WorkspaceID)
	selectArgs = append(selectArgs, matchArgs...)
	selectArgs = append(selectArgs, rankArgs...)
	selectArgs = append(selectArgs, query.Limit, query.Offset)
	rows, err := r.db.QueryContext(ctx, `SELECT `+qualifiedIssueColumns+`, `+sourceSQL+`
		FROM workspace_issue_search_documents d
		JOIN workspace_issues i ON i.id=d.issue_id AND i.workspace_id=d.workspace_id
		WHERE d.workspace_id=?`+closedSQL+` AND (`+matchSQL+`)
		ORDER BY `+rankSQL+` ASC, d.updated_at DESC, d.issue_id ASC
		LIMIT ? OFFSET ?`, selectArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("query Issue search results: %w", err)
	}
	defer rows.Close()
	results := make([]application.IssueSearchResult, 0)
	for rows.Next() {
		value, err := scanIssueWithSource(rows)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, value)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate Issue search results: %w", err)
	}
	return results, total, nil
}

func issueSearchPredicates(query application.IssueSearchQuery) (matchSQL, rankSQL, sourceSQL string, matchArgs, rankArgs []any) {
	titleTerms, descriptionTerms := termPredicate("d.title", query.Terms), termPredicate("d.description", query.Terms)
	matchSQL = `d.identifier=? OR CAST(d.number AS TEXT)=? OR d.title=? OR (` + titleTerms + `) OR (` + descriptionTerms + `)`
	rankSQL = `CASE WHEN d.identifier=? OR CAST(d.number AS TEXT)=? THEN 0 WHEN d.title=? THEN 1 WHEN (` + titleTerms + `) THEN 2 ELSE 3 END`
	sourceSQL = `CASE WHEN d.identifier=? OR CAST(d.number AS TEXT)=? THEN 'identifier' WHEN d.title=? OR (` + titleTerms + `) THEN 'title' ELSE 'description' END`
	rankArgs = []any{query.Phrase, query.Phrase, query.Phrase}
	titleArgs := make([]any, 0, len(query.Terms))
	for _, term := range query.Terms {
		titleArgs = append(titleArgs, term)
	}
	rankArgs = append(rankArgs, titleArgs...)
	matchArgs = append(matchArgs, rankArgs...)
	for _, term := range query.Terms {
		matchArgs = append(matchArgs, term)
	}
	return matchSQL, rankSQL, sourceSQL, matchArgs, rankArgs
}

func termPredicate(column string, terms []string) string {
	parts := make([]string, len(terms))
	for index := range terms {
		parts[index] = `instr(` + column + `, ?) > 0`
	}
	return strings.Join(parts, " AND ")
}

type issueSearchScanner interface{ Scan(...any) error }

func scanIssueWithSource(scanner issueSearchScanner) (application.IssueSearchResult, error) {
	var source string
	row := &issueAndSourceScanner{scanner: scanner, source: &source}
	value, err := scanIssue(row)
	if err != nil {
		return application.IssueSearchResult{}, err
	}
	result := application.IssueSearchResult{Issue: value, MatchSource: source}
	if source == "description" && value.Description != nil {
		result.DescriptionSnippet = boundedIssueSearchSnippet(*value.Description, 240)
		result.MatchedSnippet = result.DescriptionSnippet
	}
	return result, nil
}

type issueAndSourceScanner struct {
	scanner issueSearchScanner
	source  *string
}

func (s *issueAndSourceScanner) Scan(dest ...any) error {
	return s.scanner.Scan(append(dest, s.source)...)
}

func boundedIssueSearchSnippet(value string, limit int) string {
	if utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	return string(runes[:limit]) + "…"
}

var _ application.IssueSearchRepository = (*issueSearchRepository)(nil)
