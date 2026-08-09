package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth/internal/application"
	memberDomain "github.com/hvritual/workspace/internal/modules/auth/internal/domain/member"
)

type MemberStore struct {
	db *sql.DB
}

func NewMemberStore(db *sql.DB) *MemberStore {
	return &MemberStore{db: db}
}

func (s *MemberStore) WithinTransaction(ctx context.Context, operation func(application.MemberRepository) error) error {
	connection, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire Auth member connection: %w", err)
	}
	defer connection.Close()
	if _, err := connection.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return fmt.Errorf("begin Auth member transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	if err := operation(&memberRepository{connection: connection}); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit Auth member transaction: %w", err)
	}
	committed = true
	return nil
}

type memberRepository struct {
	connection *sql.Conn
}

func (r *memberRepository) CreateWorkspaceRoot(ctx context.Context, value memberDomain.WorkspaceRoot) error {
	_, err := r.connection.ExecContext(ctx, `INSERT INTO auth_workspace_membership_roots(
		workspace_id, user_id, member_id, created_at
	) VALUES (?, ?, ?, ?)`,
		value.WorkspaceID(), value.UserID(), value.MemberID(), value.CreatedAt().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create workspace membership root: %w", err)
	}
	return nil
}

func (r *memberRepository) FindWorkspaceRoot(ctx context.Context, workspaceID string) (memberDomain.WorkspaceRoot, error) {
	var rowWorkspaceID, userID, memberID, rawCreatedAt string
	if err := r.connection.QueryRowContext(ctx, `SELECT workspace_id, user_id, member_id, created_at
		FROM auth_workspace_membership_roots WHERE workspace_id = ?`, workspaceID,
	).Scan(&rowWorkspaceID, &userID, &memberID, &rawCreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memberDomain.WorkspaceRoot{}, application.ErrWorkspaceRootNotFound
		}
		return memberDomain.WorkspaceRoot{}, fmt.Errorf("find workspace membership root: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rawCreatedAt)
	if err != nil {
		return memberDomain.WorkspaceRoot{}, fmt.Errorf("parse workspace membership root created_at: %w", err)
	}
	value, err := memberDomain.NewWorkspaceRoot(rowWorkspaceID, userID, memberID, createdAt)
	if err != nil {
		return memberDomain.WorkspaceRoot{}, fmt.Errorf("map workspace membership root: %w", err)
	}
	return value, nil
}

func (r *memberRepository) FindUserByID(ctx context.Context, userID string) (memberDomain.User, error) {
	var id, name, email string
	var avatarURL sql.NullString
	if err := r.connection.QueryRowContext(ctx, `SELECT id, name, email, avatar_url FROM auth_users WHERE id = ?`, userID).Scan(
		&id, &name, &email, &avatarURL,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memberDomain.User{}, application.ErrAuthUserRecordNotFound
		}
		return memberDomain.User{}, fmt.Errorf("find Auth user: %w", err)
	}
	var avatar *string
	if avatarURL.Valid {
		avatar = &avatarURL.String
	}
	value, err := memberDomain.RehydrateUser(id, name, email, avatar)
	if err != nil {
		return memberDomain.User{}, fmt.Errorf("map Auth user: %w", err)
	}
	return value, nil
}

func (r *memberRepository) Create(ctx context.Context, value memberDomain.Member) error {
	_, err := r.connection.ExecContext(ctx, `INSERT INTO auth_members(id, workspace_id, user_id, role, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		value.ID(), value.WorkspaceID(), value.UserID(), string(value.Role()), value.CreatedAt().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("%w: %v", application.ErrMemberRecordConflict, err)
	}
	return nil
}

func (r *memberRepository) FindByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (memberDomain.Member, error) {
	return scanMember(r.connection.QueryRowContext(ctx, memberProjection+`
		WHERE m.user_id = ? AND m.workspace_id = ?`, userID, workspaceID))
}

func (r *memberRepository) FindByIDAndWorkspace(ctx context.Context, memberID, workspaceID string) (memberDomain.Member, error) {
	return scanMember(r.connection.QueryRowContext(ctx, memberProjection+`
		WHERE m.id = ? AND m.workspace_id = ?`, memberID, workspaceID))
}

func (r *memberRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]memberDomain.Member, error) {
	rows, err := r.connection.QueryContext(ctx, memberProjection+`
		WHERE m.workspace_id = ? ORDER BY m.created_at ASC, m.id ASC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list Workspace members: %w", err)
	}
	defer rows.Close()
	values := make([]memberDomain.Member, 0)
	for rows.Next() {
		value, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Workspace members: %w", err)
	}
	return values, nil
}

func (r *memberRepository) CountOwners(ctx context.Context, workspaceID string) (int, error) {
	var count int
	if err := r.connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM auth_members
		WHERE workspace_id = ? AND role = 'owner'`, workspaceID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count Workspace owners: %w", err)
	}
	return count, nil
}

func (r *memberRepository) UpdateRole(ctx context.Context, value memberDomain.Member) error {
	result, err := r.connection.ExecContext(ctx, `UPDATE auth_members SET role = ?
		WHERE workspace_id = ? AND id = ?`, string(value.Role()), value.WorkspaceID(), value.ID())
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect member role update: %w", err)
	}
	if affected == 0 {
		return application.ErrMemberRecordNotFound
	}
	return nil
}

const memberProjection = `SELECT
	m.id, m.workspace_id, m.user_id, m.role, m.created_at,
	u.name, u.email, u.avatar_url
	FROM auth_members m
	JOIN auth_users u ON u.id = m.user_id `

type memberScanner interface {
	Scan(...any) error
}

func scanMember(scanner memberScanner) (memberDomain.Member, error) {
	var id, workspaceID, userID, role, rawCreatedAt, name, email string
	var avatarURL sql.NullString
	if err := scanner.Scan(&id, &workspaceID, &userID, &role, &rawCreatedAt, &name, &email, &avatarURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return memberDomain.Member{}, application.ErrMemberRecordNotFound
		}
		return memberDomain.Member{}, fmt.Errorf("scan Auth member: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, rawCreatedAt)
	if err != nil {
		return memberDomain.Member{}, fmt.Errorf("parse Auth member created_at: %w", err)
	}
	var avatar *string
	if avatarURL.Valid {
		avatar = &avatarURL.String
	}
	value, err := memberDomain.Rehydrate(id, workspaceID, userID, role, createdAt, name, email, avatar)
	if err != nil {
		return memberDomain.Member{}, fmt.Errorf("map Auth member: %w", err)
	}
	return value, nil
}

var _ application.MemberUnitOfWork = (*MemberStore)(nil)
