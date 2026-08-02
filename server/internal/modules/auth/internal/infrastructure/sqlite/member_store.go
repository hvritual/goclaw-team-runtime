package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/multica-ai/multica/server/internal/modules/auth/internal/application"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/domain/invitation"
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

func (r *invitationRepository) ExpirePendingByWorkspace(ctx context.Context, workspaceID string, expiredAt time.Time) error {
	timestamp := expiredAt.UTC().Format(time.RFC3339Nano)
	if _, err := r.tx.ExecContext(ctx, `UPDATE invitations
		SET status = 'expired', updated_at = ?
		WHERE workspace_id = ? AND status = 'pending' AND expires_at <= ?`,
		timestamp, workspaceID, timestamp,
	); err != nil {
		return fmt.Errorf("expire workspace invitations: %w", err)
	}
	return nil
}

func (r *invitationRepository) ExpirePendingByWorkspaceAndEmail(
	ctx context.Context,
	workspaceID string,
	email string,
	expiredAt time.Time,
) error {
	timestamp := expiredAt.UTC().Format(time.RFC3339Nano)
	if _, err := r.tx.ExecContext(ctx, `UPDATE invitations
		SET status = 'expired', updated_at = ?
		WHERE workspace_id = ? AND invitee_email = ? AND status = 'pending' AND expires_at <= ?`,
		timestamp, workspaceID, email, timestamp,
	); err != nil {
		return fmt.Errorf("expire invitee invitations: %w", err)
	}
	return nil
}

func (r *invitationRepository) PendingExistsByWorkspaceAndEmail(
	ctx context.Context,
	workspaceID string,
	email string,
) (bool, error) {
	var count int
	if err := r.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM invitations
		WHERE workspace_id = ? AND invitee_email = ? AND status = 'pending'`,
		workspaceID, email,
	).Scan(&count); err != nil {
		return false, fmt.Errorf("check pending invitation: %w", err)
	}
	return count > 0, nil
}

func (r *invitationRepository) Create(ctx context.Context, value invitation.Invitation) error {
	_, err := r.tx.ExecContext(ctx, `INSERT INTO invitations(
		id, workspace_id, inviter_id, invitee_email, invitee_user_id, role, status,
		created_at, updated_at, expires_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		value.ID, value.WorkspaceID, value.InviterID, value.InviteeEmail, value.InviteeUserID,
		string(value.Role), string(value.Status),
		value.CreatedAt.UTC().Format(time.RFC3339Nano),
		value.UpdatedAt.UTC().Format(time.RFC3339Nano),
		value.ExpiresAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		// The current public route reports every insert conflict as an existing
		// pending invitation; the application performs explicit checks first.
		return application.ErrPendingInvitationExists
	}
	return nil
}

func (r *invitationRepository) ListPendingByWorkspace(ctx context.Context, workspaceID string) ([]invitation.Invitation, error) {
	rows, err := r.tx.QueryContext(ctx, invitationProjection+`
		WHERE i.workspace_id = ? AND i.status = 'pending'
		ORDER BY i.created_at`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	values := make([]invitation.Invitation, 0)
	for rows.Next() {
		value, scanErr := scanInvitation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate workspace invitations: %w", err)
	}
	return values, nil
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

func (r *memberRepository) ExistsByEmail(ctx context.Context, workspaceID, email string) (bool, error) {
	var count int
	if err := r.tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM members m
		JOIN users u ON u.id = m.user_id
		WHERE m.workspace_id = ? AND lower(u.email) = ?`, workspaceID, email).Scan(&count); err != nil {
		return false, fmt.Errorf("check workspace membership by email: %w", err)
	}
	return count > 0, nil
}

func (r *memberRepository) FindUserIDByEmail(ctx context.Context, email string) (*string, error) {
	var userID string
	if err := r.tx.QueryRowContext(ctx, `SELECT id FROM users WHERE lower(email) = ?`, email).Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("resolve invitee user: %w", err)
	}
	return &userID, nil
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

const invitationProjection = `SELECT
	i.id, i.workspace_id, i.inviter_id, i.invitee_email, i.invitee_user_id,
	i.role, i.status, i.created_at, i.updated_at, i.expires_at,
	u.name, u.email
	FROM invitations i
	JOIN users u ON u.id = i.inviter_id`

func scanInvitation(row memberScanner) (invitation.Invitation, error) {
	var (
		value                           invitation.Invitation
		inviteeUserID                   sql.NullString
		role, status                    string
		createdAt, updatedAt, expiresAt string
	)
	if err := row.Scan(
		&value.ID, &value.WorkspaceID, &value.InviterID, &value.InviteeEmail, &inviteeUserID,
		&role, &status, &createdAt, &updatedAt, &expiresAt,
		&value.InviterName, &value.InviterEmail,
	); err != nil {
		return invitation.Invitation{}, fmt.Errorf("scan invitation: %w", err)
	}
	parsedRole, err := member.ParseRole(role)
	if err != nil {
		return invitation.Invitation{}, fmt.Errorf("scan invitation role: %w", err)
	}
	parsedStatus, err := invitation.ParseStatus(status)
	if err != nil {
		return invitation.Invitation{}, fmt.Errorf("scan invitation status: %w", err)
	}
	value.Role = parsedRole
	value.Status = parsedStatus
	if inviteeUserID.Valid {
		value.InviteeUserID = &inviteeUserID.String
	}
	if value.CreatedAt, err = parseInvitationTime("created_at", createdAt); err != nil {
		return invitation.Invitation{}, err
	}
	if value.UpdatedAt, err = parseInvitationTime("updated_at", updatedAt); err != nil {
		return invitation.Invitation{}, err
	}
	if value.ExpiresAt, err = parseInvitationTime("expires_at", expiresAt); err != nil {
		return invitation.Invitation{}, err
	}
	return value, nil
}

func parseInvitationTime(field, value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("scan invitation %s: %w", field, err)
	}
	return parsed, nil
}

var _ application.MemberUnitOfWork = (*MemberStore)(nil)
var _ application.InvitationUnitOfWork = (*MemberStore)(nil)
