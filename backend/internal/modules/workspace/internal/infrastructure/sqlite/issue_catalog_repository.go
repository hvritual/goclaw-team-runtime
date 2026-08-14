package sqlite

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	issueDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/issue"
)

type IssueCatalogRepository struct{ db *sql.DB }

func NewIssueCatalogRepository(config Config) (*IssueCatalogRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &IssueCatalogRepository{db: config.DB}, nil
}

func (r *IssueCatalogRepository) ResolveIssue(ctx context.Context, workspaceID, issueID string) (string, error) {
	var resolved string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&resolved)
	if errors.Is(err, sql.ErrNoRows) {
		return "", application.ErrIssueRecordNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve Issue catalog target: %w", err)
	}
	return resolved, nil
}

func (r *IssueCatalogRepository) ListLabels(ctx context.Context, workspaceID string) ([]contract.IssueLabel, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT label.id,label.workspace_id,label.resource_type,label.name,label.description,label.color,
		(SELECT COUNT(*) FROM workspace_issue_label_assignments assignment WHERE assignment.workspace_id=label.workspace_id AND assignment.label_id=label.id),
		label.created_at,label.updated_at FROM workspace_issue_labels label WHERE label.workspace_id=? AND label.resource_type='issue'
		ORDER BY label.name COLLATE NOCASE,label.id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Issue labels: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueLabel, 0)
	for rows.Next() {
		value, err := scanCatalogLabel(rows)
		if err != nil {
			return nil, fmt.Errorf("scan Issue label: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Issue labels: %w", err)
	}
	return values, nil
}

func (r *IssueCatalogRepository) GetLabel(ctx context.Context, workspaceID, labelID string) (contract.IssueLabel, error) {
	return getCatalogLabel(ctx, r.db, workspaceID, labelID)
}

func getCatalogLabel(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, labelID string) (contract.IssueLabel, error) {
	value, err := scanCatalogLabel(queryer.QueryRowContext(ctx, `SELECT label.id,label.workspace_id,label.resource_type,label.name,label.description,label.color,
		(SELECT COUNT(*) FROM workspace_issue_label_assignments assignment WHERE assignment.workspace_id=label.workspace_id AND assignment.label_id=label.id),
		label.created_at,label.updated_at FROM workspace_issue_labels label WHERE label.workspace_id=? AND label.id=? AND label.resource_type='issue'`, workspaceID, labelID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.IssueLabel{}, application.ErrIssueLabelNotFound
	}
	if err != nil {
		return contract.IssueLabel{}, fmt.Errorf("get Issue label: %w", err)
	}
	return value, nil
}

func (r *IssueCatalogRepository) CreateLabel(ctx context.Context, value contract.IssueLabel) (contract.IssueLabel, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issue_labels(id,workspace_id,resource_type,name,description,color,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, value.WorkspaceID, value.ResourceType, value.Name, value.Description, value.Color, value.CreatedAt, value.UpdatedAt)
	if catalogLabelNameConflict(err) {
		return contract.IssueLabel{}, application.ErrIssueCatalogConflict
	}
	if err != nil {
		return contract.IssueLabel{}, fmt.Errorf("insert Issue label: %w", err)
	}
	created, err := getCatalogLabel(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return contract.IssueLabel{}, err
	}
	committed = true
	return created, nil
}

func (r *IssueCatalogRepository) UpdateLabel(ctx context.Context, value contract.IssueLabel) (contract.IssueLabel, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issue_labels SET name=?,description=?,color=?,updated_at=? WHERE workspace_id=? AND id=? AND resource_type='issue'`, value.Name, value.Description, value.Color, value.UpdatedAt, value.WorkspaceID, value.ID)
	if catalogLabelNameConflict(err) {
		return contract.IssueLabel{}, application.ErrIssueCatalogConflict
	}
	if err != nil {
		return contract.IssueLabel{}, fmt.Errorf("update Issue label: %w", err)
	}
	if err := requireCatalogRow(result, application.ErrIssueLabelNotFound); err != nil {
		return contract.IssueLabel{}, err
	}
	updated, err := getCatalogLabel(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssueLabel{}, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return contract.IssueLabel{}, err
	}
	committed = true
	return updated, nil
}

func (r *IssueCatalogRepository) DeleteLabel(ctx context.Context, workspaceID, labelID string) error {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if _, err := getCatalogLabel(ctx, connection, workspaceID, labelID); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM workspace_issue_label_assignments WHERE workspace_id=? AND label_id=?`, workspaceID, labelID); err != nil {
		return fmt.Errorf("delete Issue label assignments: %w", err)
	}
	result, err := connection.ExecContext(ctx, `DELETE FROM workspace_issue_labels WHERE workspace_id=? AND id=?`, workspaceID, labelID)
	if err != nil {
		return fmt.Errorf("delete Issue label: %w", err)
	}
	if err := requireCatalogRow(result, application.ErrIssueLabelNotFound); err != nil {
		return err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return err
	}
	committed = true
	return nil
}

func (r *IssueCatalogRepository) ListIssueLabels(ctx context.Context, workspaceID, issueID string) ([]contract.IssueLabel, error) {
	return listIssueLabelsWith(ctx, r.db, workspaceID, issueID)
}

func listIssueLabelsWith(ctx context.Context, queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, workspaceID, issueID string) ([]contract.IssueLabel, error) {
	rows, err := queryer.QueryContext(ctx, `SELECT label.id,label.workspace_id,label.resource_type,label.name,label.description,label.color,
		(SELECT COUNT(*) FROM workspace_issue_label_assignments usage WHERE usage.workspace_id=label.workspace_id AND usage.label_id=label.id),
		label.created_at,label.updated_at FROM workspace_issue_labels label JOIN workspace_issue_label_assignments assignment
		ON assignment.workspace_id=label.workspace_id AND assignment.label_id=label.id
		WHERE assignment.workspace_id=? AND assignment.issue_id=? ORDER BY label.name COLLATE NOCASE,label.id`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list labels for Issue: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueLabel, 0)
	for rows.Next() {
		value, err := scanCatalogLabel(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *IssueCatalogRepository) AttachIssueLabel(ctx context.Context, workspaceID, issueID, labelID, now string) ([]contract.IssueLabel, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return nil, err
	}
	if _, err := getCatalogLabel(ctx, connection, workspaceID, labelID); err != nil {
		return nil, err
	}
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_issue_label_assignments(workspace_id,issue_id,label_id,created_at) VALUES(?,?,?,?) ON CONFLICT(workspace_id,issue_id,label_id) DO NOTHING`, workspaceID, issueID, labelID, now); err != nil {
		return nil, fmt.Errorf("attach Issue label: %w", err)
	}
	values, err := listIssueLabelsWith(ctx, connection, workspaceID, issueID)
	if err != nil {
		return nil, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return nil, err
	}
	committed = true
	return values, nil
}

func (r *IssueCatalogRepository) DetachIssueLabel(ctx context.Context, workspaceID, issueID, labelID string) ([]contract.IssueLabel, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return nil, err
	}
	if _, err := getCatalogLabel(ctx, connection, workspaceID, labelID); err != nil {
		return nil, err
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM workspace_issue_label_assignments WHERE workspace_id=? AND issue_id=? AND label_id=?`, workspaceID, issueID, labelID); err != nil {
		return nil, fmt.Errorf("detach Issue label: %w", err)
	}
	values, err := listIssueLabelsWith(ctx, connection, workspaceID, issueID)
	if err != nil {
		return nil, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return nil, err
	}
	committed = true
	return values, nil
}

func (r *IssueCatalogRepository) ListProperties(ctx context.Context, workspaceID string, includeArchived bool) ([]contract.IssuePropertyDefinition, error) {
	statement := `SELECT id,workspace_id,name,type,description,icon,config,position,archived_at,created_at,updated_at FROM workspace_issue_property_definitions WHERE workspace_id=?`
	if !includeArchived {
		statement += ` AND archived_at IS NULL`
	}
	statement += ` ORDER BY position,name COLLATE NOCASE,id`
	rows, err := r.db.QueryContext(ctx, statement, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Issue properties: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssuePropertyDefinition, 0)
	for rows.Next() {
		value, err := scanCatalogProperty(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for index := range values {
		values[index].UsageCount, err = propertyUsageCount(ctx, r.db, workspaceID, values[index].ID)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}

func (r *IssueCatalogRepository) GetProperty(ctx context.Context, workspaceID, propertyID string) (contract.IssuePropertyDefinition, error) {
	value, err := getCatalogProperty(ctx, r.db, workspaceID, propertyID)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	value.UsageCount, err = propertyUsageCount(ctx, r.db, workspaceID, propertyID)
	return value, err
}

func getCatalogProperty(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, propertyID string) (contract.IssuePropertyDefinition, error) {
	value, err := scanCatalogProperty(queryer.QueryRowContext(ctx, `SELECT id,workspace_id,name,type,description,icon,config,position,archived_at,created_at,updated_at FROM workspace_issue_property_definitions WHERE workspace_id=? AND id=?`, workspaceID, propertyID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.IssuePropertyDefinition{}, application.ErrIssuePropertyNotFound
	}
	if err != nil {
		return contract.IssuePropertyDefinition{}, fmt.Errorf("get Issue property: %w", err)
	}
	return value, nil
}

func (r *IssueCatalogRepository) CreateProperty(ctx context.Context, value contract.IssuePropertyDefinition) (contract.IssuePropertyDefinition, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	var active int
	if err := connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_issue_property_definitions WHERE workspace_id=? AND archived_at IS NULL`, value.WorkspaceID).Scan(&active); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if active >= 20 {
		return contract.IssuePropertyDefinition{}, application.ErrIssuePropertyLimit
	}
	if err := connection.QueryRowContext(ctx, `SELECT COALESCE(MAX(position),-1)+1 FROM workspace_issue_property_definitions WHERE workspace_id=?`, value.WorkspaceID).Scan(&value.Position); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	config, _ := json.Marshal(value.Config)
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issue_property_definitions(id,workspace_id,name,type,description,icon,config,position,archived_at,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, value.ID, value.WorkspaceID, value.Name, value.Type, value.Description, value.Icon, string(config), value.Position, nil, value.CreatedAt, value.UpdatedAt)
	if catalogPropertyNameConflict(err) {
		return contract.IssuePropertyDefinition{}, application.ErrIssueCatalogConflict
	}
	if err != nil {
		return contract.IssuePropertyDefinition{}, fmt.Errorf("insert Issue property: %w", err)
	}
	created, err := getCatalogProperty(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	committed = true
	return created, nil
}

func (r *IssueCatalogRepository) UpdateProperty(ctx context.Context, value contract.IssuePropertyDefinition, removed []string) (contract.IssuePropertyDefinition, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	existing, err := getCatalogProperty(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if existing.Archived && !value.Archived {
		var active int
		if err := connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_issue_property_definitions WHERE workspace_id=? AND archived_at IS NULL`, value.WorkspaceID).Scan(&active); err != nil {
			return contract.IssuePropertyDefinition{}, err
		}
		if active >= 20 {
			return contract.IssuePropertyDefinition{}, application.ErrIssuePropertyLimit
		}
	}
	if len(removed) != 0 {
		rows, err := connection.QueryContext(ctx, `SELECT properties FROM workspace_issues WHERE workspace_id=?`, value.WorkspaceID)
		if err != nil {
			return contract.IssuePropertyDefinition{}, err
		}
		usage := make(map[string]int, len(removed))
		for rows.Next() {
			var raw string
			if err := rows.Scan(&raw); err != nil {
				rows.Close()
				return contract.IssuePropertyDefinition{}, err
			}
			bag := decodePropertyBag(raw)
			for _, optionID := range propertyOptionUsage(bag[value.ID], removed) {
				usage[optionID]++
			}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return contract.IssuePropertyDefinition{}, err
		}
		if err := rows.Close(); err != nil {
			return contract.IssuePropertyDefinition{}, err
		}
		if len(usage) != 0 {
			options := make([]application.IssuePropertyOptionUsage, 0, len(usage))
			for _, option := range existing.Config.Options {
				if count := usage[option.ID]; count != 0 {
					options = append(options, application.IssuePropertyOptionUsage{Name: option.Name, Count: count})
				}
			}
			return contract.IssuePropertyDefinition{}, &application.IssuePropertyOptionsInUseError{Options: options}
		}
	}
	config, _ := json.Marshal(value.Config)
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issue_property_definitions SET name=?,description=?,icon=?,config=?,archived_at=?,updated_at=? WHERE workspace_id=? AND id=?`, value.Name, value.Description, value.Icon, string(config), nullableString(value.ArchivedAt), value.UpdatedAt, value.WorkspaceID, value.ID)
	if catalogPropertyNameConflict(err) {
		return contract.IssuePropertyDefinition{}, application.ErrIssueCatalogConflict
	}
	if err != nil {
		return contract.IssuePropertyDefinition{}, fmt.Errorf("update Issue property: %w", err)
	}
	if err := requireCatalogRow(result, application.ErrIssuePropertyNotFound); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	updated, err := getCatalogProperty(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	updated.UsageCount, err = propertyUsageCount(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	committed = true
	return updated, nil
}

func (r *IssueCatalogRepository) SetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID string, raw json.RawMessage, now string) (string, map[string]any, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return "", nil, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return "", nil, err
	}
	definition, err := getCatalogProperty(ctx, connection, workspaceID, propertyID)
	if err != nil {
		return "", nil, err
	}
	if definition.Archived {
		return "", nil, application.ErrIssueCatalogInvalid
	}
	value, err := application.ValidateIssuePropertyValue(definition, raw)
	if err != nil {
		return "", nil, err
	}
	bag, err := readIssuePropertyBag(ctx, connection, workspaceID, issueID)
	if err != nil {
		return "", nil, err
	}
	bag[propertyID] = value
	encoded, err := json.Marshal(bag)
	if err != nil || len(encoded) > 16*1024 {
		return "", nil, application.ErrIssueCatalogInvalid
	}
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET properties=?,updated_at=? WHERE workspace_id=? AND id=?`, string(encoded), now, workspaceID, issueID)
	if err != nil {
		return "", nil, fmt.Errorf("set Issue property: %w", err)
	}
	if err := requireCatalogRow(result, application.ErrIssueRecordNotFound); err != nil {
		return "", nil, err
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return "", nil, err
	}
	committed = true
	return issueID, bag, nil
}

func (r *IssueCatalogRepository) UnsetIssueProperty(ctx context.Context, workspaceID, issueID, propertyID, now string) (string, map[string]any, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return "", nil, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return "", nil, err
	}
	if _, err := getCatalogProperty(ctx, connection, workspaceID, propertyID); err != nil {
		return "", nil, err
	}
	bag, err := readIssuePropertyBag(ctx, connection, workspaceID, issueID)
	if err != nil {
		return "", nil, err
	}
	delete(bag, propertyID)
	encoded, _ := json.Marshal(bag)
	if _, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET properties=?,updated_at=? WHERE workspace_id=? AND id=?`, string(encoded), now, workspaceID, issueID); err != nil {
		return "", nil, fmt.Errorf("unset Issue property: %w", err)
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return "", nil, err
	}
	committed = true
	return issueID, bag, nil
}

func (r *IssueCatalogRepository) ListAcceptanceConclusions(ctx context.Context, workspaceID, issueID string) ([]contract.AcceptanceConclusion, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,workspace_id,issue_id,result,rationale,evidence_refs,actor_id,created_at,updated_at FROM workspace_issue_acceptance_conclusions WHERE workspace_id=? AND issue_id=? ORDER BY created_at DESC,id DESC`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list acceptance conclusions: %w", err)
	}
	defer rows.Close()
	values := make([]contract.AcceptanceConclusion, 0)
	for rows.Next() {
		value, err := scanAcceptanceConclusion(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return values, nil
}

func (r *IssueCatalogRepository) CreateAcceptanceConclusion(ctx context.Context, workspaceID, issueID string, value contract.AcceptanceConclusion, complete bool, now string) (issueDomain.Issue, contract.AcceptanceConclusion, error) {
	connection, err := catalogWriteConnection(ctx, r.db)
	if err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	issue, err := scanIssue(connection.QueryRowContext(ctx, `SELECT `+issueColumns+` FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID))
	if errors.Is(err, sql.ErrNoRows) {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, application.ErrIssueRecordNotFound
	}
	if err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, err
	}
	if complete {
		result, err := connection.ExecContext(ctx, `UPDATE workspace_issues SET status='done',updated_at=? WHERE workspace_id=? AND id=?`, now, workspaceID, issueID)
		if err != nil {
			return issueDomain.Issue{}, contract.AcceptanceConclusion{}, fmt.Errorf("complete accepted Issue: %w", err)
		}
		if err := requireCatalogRow(result, application.ErrIssueRecordNotFound); err != nil {
			return issueDomain.Issue{}, contract.AcceptanceConclusion{}, err
		}
	} else if issue.Status != "done" {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, application.ErrIssueAcceptanceConflict
	}
	refs, _ := json.Marshal(value.EvidenceRefs)
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issue_acceptance_conclusions(id,workspace_id,issue_id,result,rationale,evidence_refs,actor_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, value.ID, workspaceID, issueID, value.Result, value.Rationale, string(refs), value.ActorID, value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, fmt.Errorf("insert acceptance conclusion: %w", err)
	}
	issue, err = scanIssue(connection.QueryRowContext(ctx, `SELECT `+issueColumns+` FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID))
	if err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, err
	}
	captureContent := acceptanceCaptureContent(value)
	captureSum := sha256.Sum256([]byte(captureContent))
	sourceRevision := issue.UpdatedAt.UTC().Format(time.RFC3339Nano) + "@sha256:" + hex.EncodeToString(captureSum[:])
	if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_acceptance_knowledge_proposals(id,workspace_id,issue_id,conclusion_id,source_revision,content,actor_id,created_at) VALUES(?,?,?,?,?,?,?,?)`, value.ID, workspaceID, issueID, value.ID, sourceRevision, captureContent, value.ActorID, value.CreatedAt); err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, fmt.Errorf("capture acceptance knowledge: %w", err)
	}
	if err := commitCatalog(ctx, connection); err != nil {
		return issueDomain.Issue{}, contract.AcceptanceConclusion{}, err
	}
	committed = true
	return issue, value, nil
}

func acceptanceCaptureContent(value contract.AcceptanceConclusion) string {
	content := "Result: " + value.Result + "\n\n" + value.Rationale
	if len(value.EvidenceRefs) != 0 {
		content += "\n\nEvidence:\n- " + strings.Join(value.EvidenceRefs, "\n- ")
	}
	return content
}

func catalogWriteConnection(ctx context.Context, db *sql.DB) (*sql.Conn, error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire Issue catalog connection: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		connection.Close()
		return nil, err
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		connection.Close()
		return nil, err
	}
	return connection, nil
}

func commitCatalog(ctx context.Context, connection *sql.Conn) error {
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Issue catalog transaction: %w", err)
	}
	return nil
}

func catalogLabelNameConflict(err error) bool {
	message := strings.ToLower(errorMessage(err))
	return strings.Contains(message, "unique constraint failed") &&
		strings.Contains(message, "workspace_issue_labels.workspace_id") &&
		strings.Contains(message, "workspace_issue_labels.resource_type") &&
		strings.Contains(message, "workspace_issue_labels.name")
}

func catalogPropertyNameConflict(err error) bool {
	message := strings.ToLower(errorMessage(err))
	return strings.Contains(message, "unique constraint failed") &&
		strings.Contains(message, "workspace_issue_property_definitions.workspace_id") &&
		strings.Contains(message, "workspace_issue_property_definitions.name")
}

func errorMessage(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func requireCatalogRow(result sql.Result, missing error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return missing
	}
	return nil
}

func scanCatalogLabel(scanner interface{ Scan(...any) error }) (contract.IssueLabel, error) {
	var value contract.IssueLabel
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.ResourceType, &value.Name, &value.Description, &value.Color, &value.UsageCount, &value.CreatedAt, &value.UpdatedAt)
	return value, err
}

func scanCatalogProperty(scanner interface{ Scan(...any) error }) (contract.IssuePropertyDefinition, error) {
	var value contract.IssuePropertyDefinition
	var config string
	var archived sql.NullString
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.Name, &value.Type, &value.Description, &value.Icon, &config, &value.Position, &archived, &value.CreatedAt, &value.UpdatedAt)
	if err != nil {
		return contract.IssuePropertyDefinition{}, err
	}
	if json.Unmarshal([]byte(config), &value.Config) != nil {
		return contract.IssuePropertyDefinition{}, fmt.Errorf("decode Issue property config")
	}
	value.Archived = archived.Valid
	value.ArchivedAt = nullStringPointer(archived)
	return value, nil
}

func propertyUsageCount(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, propertyID string) (int64, error) {
	var count int64
	path := `$."` + strings.ReplaceAll(propertyID, `"`, `\"`) + `"`
	err := queryer.QueryRowContext(ctx, `SELECT COUNT(*) FROM workspace_issues WHERE workspace_id=? AND json_type(properties,?) IS NOT NULL`, workspaceID, path).Scan(&count)
	return count, err
}

func propertyOptionUsage(raw any, removed []string) []string {
	set := make(map[string]struct{}, len(removed))
	for _, id := range removed {
		set[id] = struct{}{}
	}
	switch value := raw.(type) {
	case string:
		if _, ok := set[value]; ok {
			return []string{value}
		}
	case []any:
		used := make([]string, 0, len(value))
		seen := map[string]struct{}{}
		for _, item := range value {
			if text, ok := item.(string); ok {
				if _, found := set[text]; found {
					if _, duplicate := seen[text]; !duplicate {
						seen[text] = struct{}{}
						used = append(used, text)
					}
				}
			}
		}
		return used
	}
	return nil
}

func readIssuePropertyBag(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, workspaceID, issueID string) (map[string]any, error) {
	var raw string
	if err := queryer.QueryRowContext(ctx, `SELECT properties FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID).Scan(&raw); errors.Is(err, sql.ErrNoRows) {
		return nil, application.ErrIssueRecordNotFound
	} else if err != nil {
		return nil, err
	}
	return decodePropertyBag(raw), nil
}

func decodePropertyBag(raw string) map[string]any {
	decoder := json.NewDecoder(bytes.NewBufferString(raw))
	decoder.UseNumber()
	decoded := map[string]any{}
	if decoder.Decode(&decoded) != nil || decoded == nil {
		return map[string]any{}
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return map[string]any{}
	}
	value := make(map[string]any, len(decoded))
	for key, candidate := range decoded {
		switch candidate := candidate.(type) {
		case string, json.Number, bool:
			value[key] = candidate
		case []any:
			items := make([]string, 0, len(candidate))
			valid := true
			for _, item := range candidate {
				text, ok := item.(string)
				if !ok {
					valid = false
					break
				}
				items = append(items, text)
			}
			if valid {
				value[key] = items
			}
		}
	}
	return value
}

func scanAcceptanceConclusion(scanner interface{ Scan(...any) error }) (contract.AcceptanceConclusion, error) {
	var value contract.AcceptanceConclusion
	var refs string
	if err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.IssueID, &value.Result, &value.Rationale, &refs, &value.ActorID, &value.CreatedAt, &value.UpdatedAt); err != nil {
		return contract.AcceptanceConclusion{}, err
	}
	value.EvidenceRefs = []string{}
	if err := json.Unmarshal([]byte(refs), &value.EvidenceRefs); err != nil || value.EvidenceRefs == nil {
		value.EvidenceRefs = []string{}
	}
	return value, nil
}

var _ application.IssueCatalogRepository = (*IssueCatalogRepository)(nil)
