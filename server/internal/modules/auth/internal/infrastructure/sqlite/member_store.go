package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/member"
)

// MemberStore binds Auth member transactions to the native SQLite connection.
type MemberStore struct {
	db *sql.DB
}

func NewMemberStore(db *sql.DB) *MemberStore {
	return &MemberStore{db: db}
}

func (s *MemberStore) WithinTransaction(ctx context.Context, operation func(application.MemberRepository) error) error {
	return s.withinTransaction(ctx, "member", func(tx *sql.Tx) error {
		return operation(&memberRepository{tx: tx})
	})
}

func (s *MemberStore) withinTransaction(ctx context.Context, label string, operation func(*sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin %s transaction: %w", label, err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := operation(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s transaction: %w", label, err)
	}
	return nil
}

func (s *MemberStore) WithinInvitationTransaction(
	ctx context.Context,
	operation func(application.MemberRepository, application.InvitationRepository) error,
) error {
	return s.withinTransaction(ctx, "invitation", func(tx *sql.Tx) error {
		return operation(&memberRepository{tx: tx}, &invitationRepository{tx: tx})
	})
}

type memberRepository struct {
	tx *sql.Tx
}

type invitationRepository struct {
	tx *sql.Tx
}

func (r *invitationRepository) RevokePending(
	ctx context.Context,
	workspaceID string,
	invitationID string,
	updatedAt time.Time,
) error {
	result, err := r.tx.ExecContext(ctx, `UPDATE invitations
		SET status = 'revoked', updated_at = ?
		WHERE id = ? AND workspace_id = ? AND status = 'pending'`,
		updatedAt.UTC().Format(time.RFC3339Nano), invitationID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("revoke invitation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked invitation count: %w", err)
	}
	if affected == 0 {
		return application.ErrInvitationNotFound
	}
	return nil
}

func (r *memberRepository) FindByUserAndWorkspace(ctx context.Context, userID, workspaceID string) (member.Member, error) {
	return scanMember(r.tx.QueryRowContext(ctx, memberProjection+`
		WHERE m.user_id = ? AND m.workspace_id = ?`, userID, workspaceID))
}

func (r *memberRepository) FindByIDAndWorkspace(ctx context.Context, memberID, workspaceID string) (member.Member, error) {
	return scanMember(r.tx.QueryRowContext(ctx, memberProjection+`
		WHERE m.id = ? AND m.workspace_id = ?`, memberID, workspaceID))
}

func (r *memberRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]member.Member, error) {
	rows, err := r.tx.QueryContext(ctx, memberProjection+`
		WHERE m.workspace_id = ? ORDER BY m.created_at`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace members: %w", err)
	}
	defer func() { _ = rows.Close() }()

	values := make([]member.Member, 0)
	for rows.Next() {
		value, scanErr := scanMember(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace members: %w", err)
	}
	return values, nil
}

func (r *memberRepository) CountOwners(ctx context.Context, workspaceID string) (int, error) {
	var count int
	if err := r.tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM members WHERE workspace_id = ? AND role = 'owner'`,
		workspaceID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count workspace owners: %w", err)
	}
	return count, nil
}

func (r *memberRepository) UpdateRole(ctx context.Context, workspaceID, memberID string, role member.Role) (member.Member, error) {
	result, err := r.tx.ExecContext(ctx,
		`UPDATE members SET role = ? WHERE id = ? AND workspace_id = ?`,
		string(role), memberID, workspaceID,
	)
	if err != nil {
		return member.Member{}, fmt.Errorf("update member role: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return member.Member{}, fmt.Errorf("read updated member count: %w", err)
	}
	if affected == 0 {
		return member.Member{}, application.ErrMembershipNotFound
	}
	return scanMember(r.tx.QueryRowContext(ctx, memberProjection+`
		WHERE m.id = ? AND m.workspace_id = ?`, memberID, workspaceID))
}

func (r *memberRepository) DeleteByIDAndWorkspace(ctx context.Context, workspaceID, memberID string) error {
	return deleteMembership(r.tx.ExecContext(ctx,
		`DELETE FROM members WHERE id = ? AND workspace_id = ?`,
		memberID, workspaceID,
	))
}

func (r *memberRepository) DeleteByUserAndWorkspace(ctx context.Context, workspaceID, userID string) error {
	return deleteMembership(r.tx.ExecContext(ctx,
		`DELETE FROM members WHERE workspace_id = ? AND user_id = ?`,
		workspaceID, userID,
	))
}

func deleteMembership(result sql.Result, err error) error {
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted membership count: %w", err)
	}
	if affected == 0 {
		return application.ErrMembershipNotFound
	}
	return nil
}

const memberProjection = `SELECT m.id, m.workspace_id, m.user_id, m.role, m.created_at,
	u.name, u.email, u.avatar_url
	FROM members m
	JOIN users u ON u.id = m.user_id`

type memberScanner interface {
	Scan(...any) error
}

func scanMember(row memberScanner) (member.Member, error) {
	var (
		value     member.Member
		role      string
		avatarURL sql.NullString
	)
	if err := row.Scan(
		&value.ID,
		&value.WorkspaceID,
		&value.UserID,
		&role,
		&value.CreatedAt,
		&value.Name,
		&value.Email,
		&avatarURL,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return member.Member{}, application.ErrMembershipNotFound
		}
		return member.Member{}, fmt.Errorf("scan member: %w", err)
	}
	parsedRole, err := member.ParseRole(role)
	if err != nil {
		return member.Member{}, fmt.Errorf("scan member role: %w", err)
	}
	value.Role = parsedRole
	if avatarURL.Valid {
		value.AvatarURL = &avatarURL.String
	}
	return value, nil
}

var _ application.MemberUnitOfWork = (*MemberStore)(nil)
var _ application.InvitationUnitOfWork = (*MemberStore)(nil)
