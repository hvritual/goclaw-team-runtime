package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/hvritual/workspace/internal/modules/workspace"
	_ "modernc.org/sqlite"
)

const (
	fixtureUserID      = "01990000-0000-7000-8000-000000000001"
	fixtureMemberID    = "01990000-0000-7000-8000-000000000002"
	fixtureWorkspaceID = "01990000-0000-7000-8000-000000000003"
	fixtureIssueID     = "01990000-0000-7000-8000-000000000004"
	fixtureEmail       = "canonical-fixture@multica.local"
	fixtureSlug        = "canonical-fixture"
	fixtureIdentifier  = "CAN-1"
)

func main() {
	path := flag.String("sqlite-path", "data/multica-canonical.db", "existing migrated Canonical SQLite database")
	flag.Parse()
	if flag.NArg() != 0 || strings.TrimSpace(*path) == "" {
		fmt.Fprintln(os.Stderr, "usage: canonical-fixture [-sqlite-path path]")
		os.Exit(2)
	}
	if info, err := os.Stat(*path); err != nil || info.IsDir() {
		fmt.Fprintln(os.Stderr, "Canonical fixture requires an existing migrated database")
		os.Exit(1)
	}
	db, err := sql.Open("sqlite", *path+"?mode=rw&_pragma=busy_timeout(5000)")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	result, err := seedFixture(context.Background(), db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(map[string]any{
		"result": result, "email": fixtureEmail, "verification_code": "888888",
		"workspace_slug": fixtureSlug, "issue_identifier": fixtureIdentifier,
	})
}

func seedFixture(ctx context.Context, db *sql.DB) (result string, err error) {
	connection, err := db.Conn(ctx)
	if err != nil {
		return "", err
	}
	defer connection.Close()
	for _, table := range []string{"auth_users", "auth_members", "auth_workspace_membership_roots", "workspaces", "workspace_issues"} {
		var count int
		if err := connection.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil || count != 1 {
			return "", errors.New("Canonical fixture requires an existing migrated database")
		}
	}
	if _, err = connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		return "", fmt.Errorf("begin Canonical fixture: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = connection.ExecContext(context.WithoutCancel(ctx), `ROLLBACK`)
		}
	}()
	var collisions int
	queries := []struct {
		query string
		args  []any
	}{
		{`SELECT COUNT(*) FROM auth_users WHERE id=? OR email=?`, []any{fixtureUserID, fixtureEmail}},
		{`SELECT COUNT(*) FROM workspaces WHERE id=? OR slug=?`, []any{fixtureWorkspaceID, fixtureSlug}},
		{`SELECT COUNT(*) FROM auth_members WHERE id=? OR (workspace_id=? AND user_id=?)`, []any{fixtureMemberID, fixtureWorkspaceID, fixtureUserID}},
		{`SELECT COUNT(*) FROM auth_workspace_membership_roots WHERE workspace_id=? OR user_id=? OR member_id=?`, []any{fixtureWorkspaceID, fixtureUserID, fixtureMemberID}},
		{`SELECT COUNT(*) FROM workspace_issues WHERE id=? OR (workspace_id=? AND identifier=?)`, []any{fixtureIssueID, fixtureWorkspaceID, fixtureIdentifier}},
	}
	for _, item := range queries {
		var count int
		if err := connection.QueryRowContext(ctx, item.query, item.args...).Scan(&count); err != nil {
			return "", fmt.Errorf("inspect Canonical fixture footprint: %w", err)
		}
		collisions += count
	}
	if collisions > 0 {
		if collisions != len(queries) {
			return "", errors.New("Canonical fixture footprint is partial or conflicts; refusing to overwrite")
		}
		var exact int
		exactQueries := []struct {
			query string
			args  []any
		}{
			{`SELECT COUNT(*) FROM auth_users WHERE id=? AND email=? AND onboarded_at IS NOT NULL`, []any{fixtureUserID, fixtureEmail}},
			{`SELECT COUNT(*) FROM workspaces WHERE id=? AND slug=? AND issue_prefix='CAN'`, []any{fixtureWorkspaceID, fixtureSlug}},
			{`SELECT COUNT(*) FROM auth_members WHERE id=? AND workspace_id=? AND user_id=? AND role='owner'`, []any{fixtureMemberID, fixtureWorkspaceID, fixtureUserID}},
			{`SELECT COUNT(*) FROM auth_workspace_membership_roots WHERE workspace_id=? AND user_id=? AND member_id=?`, []any{fixtureWorkspaceID, fixtureUserID, fixtureMemberID}},
			{`SELECT COUNT(*) FROM workspace_issues WHERE id=? AND workspace_id=? AND identifier=? AND creator_id=?`, []any{fixtureIssueID, fixtureWorkspaceID, fixtureIdentifier, fixtureMemberID}},
		}
		for _, item := range exactQueries {
			var count int
			if err := connection.QueryRowContext(ctx, item.query, item.args...).Scan(&count); err != nil {
				return "", err
			}
			exact += count
		}
		if exact != len(exactQueries) {
			return "", errors.New("Canonical fixture footprint is partial or conflicts; refusing to overwrite")
		}
		if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
			return "", err
		}
		committed = true
		return "already_present", nil
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO auth_users(id,name,email,onboarded_at,created_at,updated_at) VALUES(?,?,?,?,?,?)`, []any{fixtureUserID, "Canonical Fixture", fixtureEmail, now, now, now}},
		{`INSERT INTO workspaces(id,name,slug,settings,repos,issue_prefix,next_issue_number,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?)`, []any{fixtureWorkspaceID, "Canonical Fixture", fixtureSlug, `{}`, `[]`, "CAN", 2, now, now}},
		{`INSERT INTO auth_members(id,workspace_id,user_id,role,created_at) VALUES(?,?,?,?,?)`, []any{fixtureMemberID, fixtureWorkspaceID, fixtureUserID, "owner", now}},
		{`INSERT INTO auth_workspace_membership_roots(workspace_id,user_id,member_id,created_at) VALUES(?,?,?,?)`, []any{fixtureWorkspaceID, fixtureUserID, fixtureMemberID, now}},
		{`INSERT INTO workspace_issues(id,workspace_id,number,identifier,title,status,priority,creator_type,creator_id,metadata,properties,asset_ids,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)`, []any{fixtureIssueID, fixtureWorkspaceID, 1, fixtureIdentifier, "Canonical runtime acceptance", "todo", "none", "member", fixtureMemberID, `{"fixture":"canonical"}`, `{}`, `[]`, now, now}},
	}
	for _, item := range statements {
		if _, err := connection.ExecContext(ctx, item.query, item.args...); err != nil {
			return "", fmt.Errorf("write Canonical fixture: %w", err)
		}
	}
	if _, err = connection.ExecContext(ctx, `COMMIT`); err != nil {
		return "", fmt.Errorf("commit Canonical fixture: %w", err)
	}
	committed = true
	return "created", nil
}
