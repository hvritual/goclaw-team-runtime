package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type issueDeletionRepository struct{ db *sql.DB }

func NewIssueDeletionRepository(config Config) (application.IssueDeletionRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &issueDeletionRepository{db: config.DB}, nil
}

func (r *issueDeletionRepository) Delete(ctx context.Context, workspaceID, issueID string) (resolvedID string, err error) {
	var resolvedIdentifier string
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return "", fmt.Errorf("acquire Issue deletion connection: %w", err)
	}
	defer connection.Close()
	if _, err = connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		return "", fmt.Errorf("configure Issue deletion lock wait: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return "", fmt.Errorf("begin Issue deletion: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	if err = connection.QueryRowContext(ctx, `SELECT id,identifier FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&resolvedID, &resolvedIdentifier); errors.Is(err, sql.ErrNoRows) {
		return "", application.ErrIssueRecordNotFound
	} else if err != nil {
		return "", fmt.Errorf("resolve Issue deletion target: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_todos SET issue_id=NULL WHERE workspace_id=? AND issue_id IN (?,?)`, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear Workspace Todo Issue references: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `UPDATE workspace_issues SET parent_issue_id=NULL WHERE workspace_id=? AND parent_issue_id IN (?,?)`, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear child Issue parent references: %w", err)
	}
	if err = clearRequirementIssueReferences(ctx, connection, workspaceID, resolvedID, resolvedIdentifier); err != nil {
		return "", fmt.Errorf("clear Workspace Requirement Issue references: %w", err)
	}
	if _, err = connection.ExecContext(ctx, `DELETE FROM workspace_pins WHERE workspace_id=? AND item_type='issue' AND item_id=?`, workspaceID, resolvedID); err != nil {
		return "", fmt.Errorf("clear Workspace Issue pins: %w", err)
	}
	result, err := connection.ExecContext(ctx, `DELETE FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, resolvedID)
	if err != nil {
		return "", fmt.Errorf("delete Workspace Issue: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return "", fmt.Errorf("inspect Workspace Issue deletion: %w", err)
	}
	if rows != 1 {
		return "", fmt.Errorf("inspect Workspace Issue deletion: expected 1 row, got %d", rows)
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return "", fmt.Errorf("commit Workspace Issue deletion: %w", err)
	}
	committed = true
	return resolvedID, nil
}

type requirementIssueReference struct {
	id, content, rawIssueIDs string
	currentVersion           int
}

func clearRequirementIssueReferences(ctx context.Context, connection *sql.Conn, workspaceID, issueID, identifier string) error {
	return clearRequirementIssueReferencesMany(ctx, connection, workspaceID, []string{issueID, identifier}, time.Now().UTC().Format(time.RFC3339Nano))
}

func clearRequirementIssueReferencesMany(ctx context.Context, connection *sql.Conn, workspaceID string, removed []string, now string) error {
	if len(removed) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(removed)), ",")
	arguments := make([]any, 0, len(removed)+1)
	arguments = append(arguments, workspaceID)
	removedSet := make(map[string]struct{}, len(removed))
	for _, value := range removed {
		arguments = append(arguments, value)
		removedSet[value] = struct{}{}
	}
	rows, err := connection.QueryContext(ctx, `SELECT r.id,r.current_version,r.issue_ids,v.content
		FROM workspace_requirements r
		JOIN workspace_requirement_versions v ON v.requirement_id=r.id AND v.version=r.current_version
		WHERE r.workspace_id=? AND EXISTS (SELECT 1 FROM json_each(r.issue_ids) WHERE value IN (`+placeholders+`))`, arguments...)
	if err != nil {
		return err
	}
	var affected []requirementIssueReference
	for rows.Next() {
		var reference requirementIssueReference
		if err := rows.Scan(&reference.id, &reference.currentVersion, &reference.rawIssueIDs, &reference.content); err != nil {
			_ = rows.Close()
			return err
		}
		affected = append(affected, reference)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, reference := range affected {
		var current []string
		if err := json.Unmarshal([]byte(reference.rawIssueIDs), &current); err != nil {
			return fmt.Errorf("decode Requirement Issue references: %w", err)
		}
		retained := make([]string, 0, len(current))
		for _, value := range current {
			if _, remove := removedSet[value]; !remove {
				retained = append(retained, value)
			}
		}
		encoded, err := json.Marshal(retained)
		if err != nil {
			return err
		}
		nextVersion := reference.currentVersion + 1
		coverage := "covered"
		if len(retained) == 0 {
			coverage = "uncovered"
		}
		result, err := connection.ExecContext(ctx, `UPDATE workspace_requirements
			SET current_version=?,approval_status='draft',coverage_status=?,issue_ids=?,updated_at=?
			WHERE workspace_id=? AND id=? AND current_version=?`, nextVersion, coverage, string(encoded), now, workspaceID, reference.id, reference.currentVersion)
		if err != nil {
			return err
		}
		if count, err := result.RowsAffected(); err != nil || count != 1 {
			if err != nil {
				return err
			}
			return errors.New("requirement version conflict")
		}
		if _, err := connection.ExecContext(ctx, `INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES(?,?,?,?,?)`, uuid.NewString(), reference.id, nextVersion, reference.content, now); err != nil {
			return err
		}
	}
	return nil
}
