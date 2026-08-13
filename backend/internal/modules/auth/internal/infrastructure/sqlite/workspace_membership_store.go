package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
)

type WorkspaceMembershipStore struct{ db *sql.DB }

func NewWorkspaceMembershipStore(db *sql.DB) (*WorkspaceMembershipStore, error) {
	if db == nil {
		return nil, errors.New("auth sqlite database is required")
	}
	return &WorkspaceMembershipStore{db: db}, nil
}

func (s *WorkspaceMembershipStore) ListForUser(ctx context.Context, userID string) ([]contract.WorkspaceMembership, error) {
	if strings.TrimSpace(userID) == "" {
		return []contract.WorkspaceMembership{}, nil
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, workspace_id, role FROM auth_members WHERE user_id = ?`, strings.TrimSpace(userID))
	if err != nil {
		return nil, fmt.Errorf("list user workspace memberships: %w", err)
	}
	defer rows.Close()
	values := make([]contract.WorkspaceMembership, 0)
	for rows.Next() {
		var value contract.WorkspaceMembership
		if err := rows.Scan(&value.MemberID, &value.WorkspaceID, &value.Role); err != nil {
			return nil, fmt.Errorf("scan user workspace membership: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user workspace memberships: %w", err)
	}
	return values, nil
}

func (s *WorkspaceMembershipStore) FindForUserAndWorkspace(ctx context.Context, userID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	return s.find(ctx, `SELECT id, workspace_id, role FROM auth_members WHERE user_id = ? AND workspace_id = ?`, strings.TrimSpace(userID), strings.TrimSpace(workspaceID))
}

func (s *WorkspaceMembershipStore) FindByMemberAndWorkspace(ctx context.Context, memberID, workspaceID string) (contract.WorkspaceMembership, bool, error) {
	return s.find(ctx, `SELECT id, workspace_id, role FROM auth_members WHERE id = ? AND workspace_id = ?`, strings.TrimSpace(memberID), strings.TrimSpace(workspaceID))
}

func (s *WorkspaceMembershipStore) find(ctx context.Context, query string, arguments ...any) (contract.WorkspaceMembership, bool, error) {
	var value contract.WorkspaceMembership
	err := s.db.QueryRowContext(ctx, query, arguments...).Scan(&value.MemberID, &value.WorkspaceID, &value.Role)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.WorkspaceMembership{}, false, nil
	}
	if err != nil {
		return contract.WorkspaceMembership{}, false, fmt.Errorf("find workspace membership: %w", err)
	}
	return value, true, nil
}

var _ contract.WorkspaceMembershipReader = (*WorkspaceMembershipStore)(nil)
