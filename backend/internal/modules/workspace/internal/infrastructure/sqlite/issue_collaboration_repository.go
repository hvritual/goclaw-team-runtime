package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
)

type IssueCollaborationRepository struct{ db *sql.DB }

func NewIssueCollaborationRepository(config Config) (*IssueCollaborationRepository, error) {
	if config.DB == nil {
		return nil, errors.New("workspace sqlite database is required")
	}
	return &IssueCollaborationRepository{db: config.DB}, nil
}

const issueCommentColumns = `id,workspace_id,issue_id,author_type,author_id,content,type,parent_id,created_at,updated_at,resolved_at,resolved_by_type,resolved_by_id`

type collaborationQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (r *IssueCollaborationRepository) ResolveIssue(ctx context.Context, workspaceID, issueID string) (string, error) {
	var resolved string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND (id=? OR identifier=?)`, workspaceID, issueID, issueID).Scan(&resolved)
	if errors.Is(err, sql.ErrNoRows) {
		return "", application.ErrIssueRecordNotFound
	}
	if err != nil {
		return "", fmt.Errorf("resolve collaboration Issue: %w", err)
	}
	return resolved, nil
}

func (r *IssueCollaborationRepository) GetComment(ctx context.Context, workspaceID, commentID string) (contract.IssueComment, error) {
	return getIssueComment(ctx, r.db, workspaceID, commentID)
}

func getIssueComment(ctx context.Context, queryer collaborationQueryer, workspaceID, commentID string) (contract.IssueComment, error) {
	value, err := scanIssueComment(queryer.QueryRowContext(ctx, `SELECT `+issueCommentColumns+` FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, workspaceID, commentID))
	if errors.Is(err, sql.ErrNoRows) {
		return contract.IssueComment{}, application.ErrIssueCommentNotFound
	}
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("get Issue comment: %w", err)
	}
	reactions, err := listCommentReactionsWith(ctx, queryer, workspaceID, []string{value.ID})
	if err != nil {
		return contract.IssueComment{}, err
	}
	value.Reactions = reactions[value.ID]
	if value.Reactions == nil {
		value.Reactions = []contract.CommentReaction{}
	}
	value.Attachments = []map[string]any{}
	return value, nil
}

func (r *IssueCollaborationRepository) ListComments(ctx context.Context, workspaceID, issueID string) ([]contract.IssueComment, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+issueCommentColumns+` FROM workspace_issue_comments WHERE workspace_id=? AND issue_id=? ORDER BY created_at,id LIMIT 2000`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Issue comments: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueComment, 0)
	ids := make([]string, 0)
	for rows.Next() {
		value, scanErr := scanIssueComment(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan Issue comment: %w", scanErr)
		}
		values, ids = append(values, value), append(ids, value.ID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Issue comments: %w", err)
	}
	reactions, err := r.listCommentReactions(ctx, workspaceID, ids)
	if err != nil {
		return nil, err
	}
	for index := range values {
		values[index].Reactions = reactions[values[index].ID]
		if values[index].Reactions == nil {
			values[index].Reactions = []contract.CommentReaction{}
		}
		values[index].Attachments = []map[string]any{}
	}
	return values, nil
}

func (r *IssueCollaborationRepository) ListActivities(ctx context.Context, workspaceID, issueID string) ([]contract.IssueActivity, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,issue_id,actor_type,actor_id,action,details,created_at FROM workspace_issue_activities WHERE workspace_id=? AND issue_id=? ORDER BY created_at,id LIMIT 2000`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Issue activities: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueActivity, 0)
	for rows.Next() {
		var value contract.IssueActivity
		var raw string
		if err := rows.Scan(&value.ID, &value.IssueID, &value.ActorType, &value.ActorID, &value.Action, &raw, &value.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan Issue activity: %w", err)
		}
		value.Details = map[string]any{}
		if err := json.Unmarshal([]byte(raw), &value.Details); err != nil {
			return nil, fmt.Errorf("decode Issue activity details: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate Issue activities: %w", err)
	}
	return values, nil
}

func (r *IssueCollaborationRepository) CreateComment(ctx context.Context, value contract.IssueComment) (created contract.IssueComment, err error) {
	connection, err := r.writeConnection(ctx, "comment creation")
	if err != nil {
		return contract.IssueComment{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, value.WorkspaceID, value.IssueID); err != nil {
		return contract.IssueComment{}, err
	}
	if value.ParentID != nil {
		var parentIssue string
		if err := connection.QueryRowContext(ctx, `SELECT issue_id FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, value.WorkspaceID, *value.ParentID).Scan(&parentIssue); errors.Is(err, sql.ErrNoRows) || parentIssue != value.IssueID {
			return contract.IssueComment{}, application.ErrIssueCollaborationInvalid
		} else if err != nil {
			return contract.IssueComment{}, fmt.Errorf("validate parent Issue comment: %w", err)
		}
	}
	_, err = connection.ExecContext(ctx, `INSERT INTO workspace_issue_comments(id,workspace_id,issue_id,author_type,author_id,content,type,parent_id,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?)`, value.ID, value.WorkspaceID, value.IssueID, value.AuthorType, value.AuthorID, value.Content, value.Type, nullableString(value.ParentID), value.CreatedAt, value.UpdatedAt)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("insert Issue comment: %w", err)
	}
	created, err = getIssueComment(ctx, connection, value.WorkspaceID, value.ID)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("read created Issue comment: %v", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.IssueComment{}, fmt.Errorf("commit Issue comment creation: %w", err)
	}
	committed = true
	return created, nil
}

func (r *IssueCollaborationRepository) UpdateComment(ctx context.Context, workspaceID, commentID, content, now string) (updated contract.IssueComment, err error) {
	connection, err := r.writeConnection(ctx, "comment update")
	if err != nil {
		return contract.IssueComment{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	result, err := connection.ExecContext(ctx, `UPDATE workspace_issue_comments SET content=?,updated_at=? WHERE workspace_id=? AND id=?`, content, now, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("update Issue comment: %w", err)
	}
	if err := requireOneRow(result, application.ErrIssueCommentNotFound); err != nil {
		return contract.IssueComment{}, err
	}
	updated, err = getIssueComment(ctx, connection, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("read updated Issue comment: %v", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.IssueComment{}, fmt.Errorf("commit Issue comment update: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *IssueCollaborationRepository) DeleteComment(ctx context.Context, workspaceID, commentID string) (err error) {
	connection, err := r.writeConnection(ctx, "comment deletion")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	var exists string
	if err := connection.QueryRowContext(ctx, `SELECT id FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, workspaceID, commentID).Scan(&exists); errors.Is(err, sql.ErrNoRows) {
		return application.ErrIssueCommentNotFound
	} else if err != nil {
		return fmt.Errorf("resolve Issue comment deletion: %w", err)
	}
	thread := `WITH RECURSIVE descendants(id) AS (SELECT id FROM workspace_issue_comments WHERE workspace_id=? AND id=? UNION SELECT c.id FROM workspace_issue_comments c JOIN descendants d ON c.parent_id=d.id WHERE c.workspace_id=?) SELECT id FROM descendants`
	for _, statement := range []string{
		`DELETE FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id IN (` + thread + `)`,
		`DELETE FROM workspace_comment_knowledge_proposals WHERE workspace_id=? AND comment_id IN (` + thread + `)`,
		`DELETE FROM workspace_issue_comments WHERE workspace_id=? AND id IN (` + thread + `)`,
	} {
		arguments := []any{workspaceID, workspaceID, commentID, workspaceID}
		if _, err := connection.ExecContext(ctx, statement, arguments...); err != nil {
			return fmt.Errorf("delete Issue comment dependents: %w", err)
		}
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Issue comment deletion: %w", err)
	}
	committed = true
	return nil
}

func (r *IssueCollaborationRepository) ResolveComment(ctx context.Context, workspaceID, commentID, actorType, actorID, now string, resolved bool) (contract.IssueComment, error) {
	connection, err := r.writeConnection(ctx, "comment resolution")
	if err != nil {
		return contract.IssueComment{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	root, err := commentThreadRoot(ctx, connection, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, err
	}
	if resolved {
		_, err = connection.ExecContext(ctx, `WITH RECURSIVE thread(id) AS (SELECT id FROM workspace_issue_comments WHERE workspace_id=? AND id=? UNION SELECT c.id FROM workspace_issue_comments c JOIN thread t ON c.parent_id=t.id WHERE c.workspace_id=?) UPDATE workspace_issue_comments SET resolved_at=NULL,resolved_by_type=NULL,resolved_by_id=NULL WHERE workspace_id=? AND id IN (SELECT id FROM thread)`, workspaceID, root, workspaceID, workspaceID)
		if err == nil {
			_, err = connection.ExecContext(ctx, `UPDATE workspace_issue_comments SET resolved_at=?,resolved_by_type=?,resolved_by_id=?,updated_at=? WHERE workspace_id=? AND id=?`, now, actorType, actorID, now, workspaceID, commentID)
		}
	} else {
		_, err = connection.ExecContext(ctx, `UPDATE workspace_issue_comments SET resolved_at=NULL,resolved_by_type=NULL,resolved_by_id=NULL,updated_at=? WHERE workspace_id=? AND id=?`, now, workspaceID, commentID)
	}
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("update Issue comment resolution: %w", err)
	}
	updated, err := getIssueComment(ctx, connection, workspaceID, commentID)
	if err != nil {
		return contract.IssueComment{}, fmt.Errorf("read resolved Issue comment: %v", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.IssueComment{}, fmt.Errorf("commit Issue comment resolution: %w", err)
	}
	committed = true
	return updated, nil
}

func (r *IssueCollaborationRepository) ProposeCommentKnowledge(ctx context.Context, workspaceID, commentID, evidenceID, revision, content, actorID, createdAt string) (bool, error) {
	connection, err := r.writeConnection(ctx, "comment knowledge proposal")
	if err != nil {
		return false, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if _, err := getIssueComment(ctx, connection, workspaceID, commentID); err != nil {
		return false, err
	}
	result, err := connection.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_comment_knowledge_proposals(id,workspace_id,comment_id,source_revision,content,actor_id,created_at) VALUES(?,?,?,?,?,?,?)`, evidenceID, workspaceID, commentID, revision, content, actorID, createdAt)
	if err != nil {
		return false, fmt.Errorf("store comment knowledge proposal: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return false, fmt.Errorf("commit comment knowledge proposal: %w", err)
	}
	committed = true
	return rows == 1, nil
}

func (r *IssueCollaborationRepository) AddCommentReaction(ctx context.Context, workspaceID, commentID string, value contract.CommentReaction) (contract.CommentReaction, error) {
	connection, err := r.writeConnection(ctx, "comment reaction")
	if err != nil {
		return contract.CommentReaction{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if _, err := getIssueComment(ctx, connection, workspaceID, commentID); err != nil {
		return contract.CommentReaction{}, err
	}
	_, err = connection.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_comment_reactions(id,workspace_id,comment_id,actor_type,actor_id,emoji,created_at) VALUES(?,?,?,?,?,?,?)`, value.ID, workspaceID, commentID, value.ActorType, value.ActorID, value.Emoji, value.CreatedAt)
	if err != nil {
		return contract.CommentReaction{}, fmt.Errorf("add comment reaction: %w", err)
	}
	err = connection.QueryRowContext(ctx, `SELECT id,comment_id,actor_type,actor_id,emoji,created_at FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id=? AND actor_type=? AND actor_id=? AND emoji=?`, workspaceID, commentID, value.ActorType, value.ActorID, value.Emoji).Scan(&value.ID, &value.CommentID, &value.ActorType, &value.ActorID, &value.Emoji, &value.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.CommentReaction{}, application.ErrIssueCommentNotFound
	}
	if err != nil {
		return contract.CommentReaction{}, fmt.Errorf("read comment reaction: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.CommentReaction{}, fmt.Errorf("commit comment reaction: %w", err)
	}
	committed = true
	return value, nil
}

func (r *IssueCollaborationRepository) RemoveCommentReaction(ctx context.Context, workspaceID, commentID, actorType, actorID, emoji string) error {
	connection, err := r.writeConnection(ctx, "comment reaction removal")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if _, err := getIssueComment(ctx, connection, workspaceID, commentID); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id=? AND actor_type=? AND actor_id=? AND emoji=?`, workspaceID, commentID, actorType, actorID, emoji); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit comment reaction removal: %w", err)
	}
	committed = true
	return nil
}

func (r *IssueCollaborationRepository) ListIssueReactions(ctx context.Context, workspaceID, issueID string) ([]contract.IssueReaction, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id,issue_id,actor_type,actor_id,emoji,created_at FROM workspace_issue_reactions WHERE workspace_id=? AND issue_id=? ORDER BY created_at,id`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Issue reactions: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueReaction, 0)
	for rows.Next() {
		var value contract.IssueReaction
		if err := rows.Scan(&value.ID, &value.IssueID, &value.ActorType, &value.ActorID, &value.Emoji, &value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *IssueCollaborationRepository) AddIssueReaction(ctx context.Context, workspaceID, issueID string, value contract.IssueReaction) (contract.IssueReaction, error) {
	connection, err := r.writeConnection(ctx, "Issue reaction")
	if err != nil {
		return contract.IssueReaction{}, err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return contract.IssueReaction{}, err
	}
	_, err = connection.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_issue_reactions(id,workspace_id,issue_id,actor_type,actor_id,emoji,created_at) VALUES(?,?,?,?,?,?,?)`, value.ID, workspaceID, issueID, value.ActorType, value.ActorID, value.Emoji, value.CreatedAt)
	if err != nil {
		return contract.IssueReaction{}, fmt.Errorf("add Issue reaction: %w", err)
	}
	err = connection.QueryRowContext(ctx, `SELECT id,issue_id,actor_type,actor_id,emoji,created_at FROM workspace_issue_reactions WHERE workspace_id=? AND issue_id=? AND actor_type=? AND actor_id=? AND emoji=?`, workspaceID, issueID, value.ActorType, value.ActorID, value.Emoji).Scan(&value.ID, &value.IssueID, &value.ActorType, &value.ActorID, &value.Emoji, &value.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return contract.IssueReaction{}, application.ErrIssueRecordNotFound
	}
	if err != nil {
		return contract.IssueReaction{}, fmt.Errorf("read Issue reaction: %w", err)
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return contract.IssueReaction{}, fmt.Errorf("commit Issue reaction: %w", err)
	}
	committed = true
	return value, nil
}

func (r *IssueCollaborationRepository) RemoveIssueReaction(ctx context.Context, workspaceID, issueID, actorType, actorID, emoji string) error {
	connection, err := r.writeConnection(ctx, "Issue reaction removal")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `DELETE FROM workspace_issue_reactions WHERE workspace_id=? AND issue_id=? AND actor_type=? AND actor_id=? AND emoji=?`, workspaceID, issueID, actorType, actorID, emoji); err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Issue reaction removal: %w", err)
	}
	committed = true
	return nil
}

func (r *IssueCollaborationRepository) ListIssueSubscribers(ctx context.Context, workspaceID, issueID string) ([]contract.IssueSubscriber, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT issue_id,user_type,user_id,reason,created_at FROM workspace_issue_subscribers WHERE workspace_id=? AND issue_id=? ORDER BY created_at,user_type,user_id`, workspaceID, issueID)
	if err != nil {
		return nil, fmt.Errorf("list Issue subscribers: %w", err)
	}
	defer rows.Close()
	values := make([]contract.IssueSubscriber, 0)
	for rows.Next() {
		var value contract.IssueSubscriber
		if err := rows.Scan(&value.IssueID, &value.UserType, &value.UserID, &value.Reason, &value.CreatedAt); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (r *IssueCollaborationRepository) SetIssueSubscriber(ctx context.Context, workspaceID, issueID string, value contract.IssueSubscriber, subscribed bool) error {
	connection, err := r.writeConnection(ctx, "Issue subscription")
	if err != nil {
		return err
	}
	defer connection.Close()
	committed := false
	defer rollbackConnection(ctx, connection, &committed)
	if err := requireIssueOnConnection(ctx, connection, workspaceID, issueID); err != nil {
		return err
	}
	if subscribed {
		_, err = connection.ExecContext(ctx, `INSERT OR IGNORE INTO workspace_issue_subscribers(workspace_id,issue_id,user_type,user_id,reason,created_at) VALUES(?,?,?,?,?,?)`, workspaceID, issueID, value.UserType, value.UserID, value.Reason, value.CreatedAt)
	} else {
		_, err = connection.ExecContext(ctx, `DELETE FROM workspace_issue_subscribers WHERE workspace_id=? AND issue_id=? AND user_type=? AND user_id=?`, workspaceID, issueID, value.UserType, value.UserID)
	}
	if err != nil {
		return err
	}
	if _, err := connection.ExecContext(ctx, `COMMIT`); err != nil {
		return fmt.Errorf("commit Issue subscription: %w", err)
	}
	committed = true
	return nil
}

func (r *IssueCollaborationRepository) RecordActivity(ctx context.Context, workspaceID string, value contract.IssueActivity) (contract.IssueActivity, error) {
	details, err := json.Marshal(value.Details)
	if err != nil {
		return contract.IssueActivity{}, err
	}
	result, err := r.db.ExecContext(ctx, `INSERT INTO workspace_issue_activities(id,workspace_id,issue_id,actor_type,actor_id,action,details,created_at) SELECT ?,?,?,?,?,?,?,? WHERE EXISTS (SELECT 1 FROM workspace_issues WHERE workspace_id=? AND id=?)`, value.ID, workspaceID, value.IssueID, value.ActorType, value.ActorID, value.Action, string(details), value.CreatedAt, workspaceID, value.IssueID)
	if err != nil {
		return contract.IssueActivity{}, fmt.Errorf("record Issue activity: %w", err)
	}
	if err := requireOneRow(result, application.ErrIssueRecordNotFound); err != nil {
		return contract.IssueActivity{}, err
	}
	return value, nil
}

func (r *IssueCollaborationRepository) listCommentReactions(ctx context.Context, workspaceID string, commentIDs []string) (map[string][]contract.CommentReaction, error) {
	return listCommentReactionsWith(ctx, r.db, workspaceID, commentIDs)
}

func listCommentReactionsWith(ctx context.Context, queryer collaborationQueryer, workspaceID string, commentIDs []string) (map[string][]contract.CommentReaction, error) {
	result := make(map[string][]contract.CommentReaction, len(commentIDs))
	if len(commentIDs) == 0 {
		return result, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(commentIDs)), ",")
	arguments := make([]any, 0, len(commentIDs)+1)
	arguments = append(arguments, workspaceID)
	for _, id := range commentIDs {
		arguments = append(arguments, id)
	}
	rows, err := queryer.QueryContext(ctx, `SELECT id,comment_id,actor_type,actor_id,emoji,created_at FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id IN (`+placeholders+`) ORDER BY created_at,id`, arguments...)
	if err != nil {
		return nil, fmt.Errorf("list comment reactions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var value contract.CommentReaction
		if err := rows.Scan(&value.ID, &value.CommentID, &value.ActorType, &value.ActorID, &value.Emoji, &value.CreatedAt); err != nil {
			return nil, err
		}
		result[value.CommentID] = append(result[value.CommentID], value)
	}
	return result, rows.Err()
}

func scanIssueComment(scanner interface{ Scan(...any) error }) (contract.IssueComment, error) {
	var value contract.IssueComment
	var parentID, resolvedAt, resolvedByType, resolvedByID sql.NullString
	err := scanner.Scan(&value.ID, &value.WorkspaceID, &value.IssueID, &value.AuthorType, &value.AuthorID, &value.Content, &value.Type, &parentID, &value.CreatedAt, &value.UpdatedAt, &resolvedAt, &resolvedByType, &resolvedByID)
	value.ParentID, value.ResolvedAt = nullStringPointer(parentID), nullStringPointer(resolvedAt)
	value.ResolvedByType, value.ResolvedByID = nullStringPointer(resolvedByType), nullStringPointer(resolvedByID)
	return value, err
}

func (r *IssueCollaborationRepository) writeConnection(ctx context.Context, operation string) (*sql.Conn, error) {
	connection, err := r.db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("acquire %s connection: %w", operation, err)
	}
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		_ = connection.Close()
		return nil, err
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return connection, nil
}

func rollbackConnection(ctx context.Context, connection *sql.Conn, committed *bool) {
	if !*committed {
		_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
	}
}

func requireIssueOnConnection(ctx context.Context, connection *sql.Conn, workspaceID, issueID string) error {
	var found string
	if err := connection.QueryRowContext(ctx, `SELECT id FROM workspace_issues WHERE workspace_id=? AND id=?`, workspaceID, issueID).Scan(&found); errors.Is(err, sql.ErrNoRows) {
		return application.ErrIssueRecordNotFound
	} else if err != nil {
		return err
	}
	return nil
}

func commentThreadRoot(ctx context.Context, connection *sql.Conn, workspaceID, commentID string) (string, error) {
	current := commentID
	seen := map[string]struct{}{}
	for index := 0; index < 2000; index++ {
		if _, duplicate := seen[current]; duplicate {
			return "", application.ErrIssueCollaborationInvalid
		}
		seen[current] = struct{}{}
		var parent sql.NullString
		if err := connection.QueryRowContext(ctx, `SELECT parent_id FROM workspace_issue_comments WHERE workspace_id=? AND id=?`, workspaceID, current).Scan(&parent); errors.Is(err, sql.ErrNoRows) {
			return "", application.ErrIssueCommentNotFound
		} else if err != nil {
			return "", err
		}
		if !parent.Valid || parent.String == "" {
			return current, nil
		}
		current = parent.String
	}
	return "", application.ErrIssueCollaborationInvalid
}

func requireOneRow(result sql.Result, notFound error) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows != 1 {
		return notFound
	}
	return nil
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}

func clearIssueCollaborationDependents(ctx context.Context, connection *sql.Conn, workspaceID string, issueIDs []string) error {
	if len(issueIDs) == 0 {
		return nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(issueIDs)), ",")
	arguments := make([]any, 0, len(issueIDs)+1)
	arguments = append(arguments, workspaceID)
	for _, issueID := range issueIDs {
		arguments = append(arguments, issueID)
	}
	commentSubquery := `SELECT id FROM workspace_issue_comments WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`
	for _, statement := range []string{
		`DELETE FROM workspace_comment_reactions WHERE workspace_id=? AND comment_id IN (` + commentSubquery + `)`,
		`DELETE FROM workspace_comment_knowledge_proposals WHERE workspace_id=? AND comment_id IN (` + commentSubquery + `)`,
		`DELETE FROM workspace_issue_comments WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`,
		`DELETE FROM workspace_issue_reactions WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`,
		`DELETE FROM workspace_issue_subscribers WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`,
		`DELETE FROM workspace_issue_activities WHERE workspace_id=? AND issue_id IN (` + placeholders + `)`,
	} {
		statementArguments := append([]any(nil), arguments...)
		if strings.Contains(statement, commentSubquery) {
			statementArguments = append([]any{workspaceID}, arguments...)
		}
		if _, err := connection.ExecContext(ctx, statement, statementArguments...); err != nil {
			return fmt.Errorf("clear Issue collaboration dependents: %w", err)
		}
	}
	return nil
}

var _ application.IssueCollaborationRepository = (*IssueCollaborationRepository)(nil)
