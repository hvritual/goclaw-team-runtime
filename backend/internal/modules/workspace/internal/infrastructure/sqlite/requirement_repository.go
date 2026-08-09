package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	requirementDomain "github.com/hvritual/workspace/internal/modules/workspace/internal/domain/requirement"
)

type requirementRepository struct{ db *sql.DB }

func NewRequirementRepository(c Config) (application.RequirementRepository, error) {
	if c.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &requirementRepository{c.DB}, nil
}
func (r *requirementRepository) FindByID(ctx context.Context, w, id string) (requirementDomain.Requirement, requirementDomain.Version, error) {
	var req requirementDomain.Requirement
	var issues, created, updated string
	err := r.db.QueryRowContext(ctx, `SELECT id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at FROM workspace_requirements WHERE workspace_id=? AND id=?`, w, id).Scan(&req.ID, &req.WorkspaceID, &req.ProjectID, &req.Title, &req.CurrentVersion, &req.ApprovalStatus, &req.CoverageStatus, &issues, &created, &updated)
	if errors.Is(err, sql.ErrNoRows) {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, application.ErrRequirementRecordNotFound
	}
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, fmt.Errorf("select Requirement: %w", err)
	}
	if err = json.Unmarshal([]byte(issues), &req.IssueIDs); err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, err
	}
	req.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, err
	}
	req.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, err
	}
	req, err = requirementDomain.Rehydrate(req)
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, err
	}
	var version requirementDomain.Version
	var versionCreated string
	err = r.db.QueryRowContext(ctx, `SELECT id,requirement_id,version,content,created_at FROM workspace_requirement_versions WHERE requirement_id=? AND version=?`, req.ID, req.CurrentVersion).Scan(&version.ID, &version.RequirementID, &version.Version, &version.Content, &versionCreated)
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, fmt.Errorf("select Requirement version: %w", err)
	}
	version.CreatedAt, err = time.Parse(time.RFC3339Nano, versionCreated)
	if err != nil {
		return requirementDomain.Requirement{}, requirementDomain.Version{}, err
	}
	return req, version, nil
}
func (r *requirementRepository) SaveVersion(ctx context.Context, req requirementDomain.Requirement, v requirementDomain.Version, creating bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Requirement version: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	issues, _ := json.Marshal(req.IssueIDs)
	if creating {
		_, err = tx.ExecContext(ctx, `INSERT INTO workspace_requirements(id,workspace_id,project_id,title,current_version,approval_status,coverage_status,issue_ids,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, req.ID, req.WorkspaceID, req.ProjectID, req.Title, req.CurrentVersion, req.ApprovalStatus, req.CoverageStatus, string(issues), req.CreatedAt.Format(time.RFC3339Nano), req.UpdatedAt.Format(time.RFC3339Nano))
	} else {
		result, e := tx.ExecContext(ctx, `UPDATE workspace_requirements SET title=?,current_version=?,approval_status=?,coverage_status=?,issue_ids=?,updated_at=? WHERE workspace_id=? AND id=? AND current_version=?`, req.Title, req.CurrentVersion, req.ApprovalStatus, req.CoverageStatus, string(issues), req.UpdatedAt.Format(time.RFC3339Nano), req.WorkspaceID, req.ID, req.CurrentVersion-1)
		err = e
		if err == nil {
			n, _ := result.RowsAffected()
			if n == 0 {
				err = errors.New("requirement version conflict")
			}
		}
	}
	if err != nil {
		return fmt.Errorf("write Requirement: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO workspace_requirement_versions(id,requirement_id,version,content,created_at) VALUES(?,?,?,?,?)`, v.ID, v.RequirementID, v.Version, v.Content, v.CreatedAt.Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("insert Requirement version: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit Requirement version: %w", err)
	}
	return nil
}
