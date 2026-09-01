package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/hvritual/workspace/internal/modules/auth"
	engineering "github.com/hvritual/workspace/internal/modules/engineering"
	engineeringcontract "github.com/hvritual/workspace/internal/modules/engineering/contract"
	engineeringmcp "github.com/hvritual/workspace/internal/modules/engineering/mcpserver"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	_ "modernc.org/sqlite"
)

const version = "p2-s08-v1"

type config struct {
	SQLitePath string
	UserID     string
}

func main() {
	cfg, err := parseConfig(os.Args[1:], os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	ctx := context.Background()
	db, err := openCanonicalSQLite(ctx, cfg.SQLitePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer db.Close()
	if err := auth.MigrateSqlite(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := engineering.MigrateSqlite(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	memberships, err := auth.NewSQLiteWorkspaceMembershipReader(db)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resolver := engineeringcontract.WorkspaceRoleResolverFunc(func(requestContext context.Context, userID, workspaceID string) (string, bool, error) {
		membership, found, readErr := memberships.FindForUserAndWorkspace(requestContext, userID, workspaceID)
		return membership.Role, found, readErr
	})
	module, err := engineering.NewWithSQLite(db, engineering.Dependencies{Memberships: resolver, Now: time.Now})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	server, err := engineeringmcp.New(engineeringmcp.Dependencies{
		Service: module.Service(), Compiler: module.AuthorizedContextCompiler(), UserID: cfg.UserID, Version: version,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := server.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func parseConfig(arguments []string, output io.Writer) (config, error) {
	flags := flag.NewFlagSet("engineering-mcp", flag.ContinueOnError)
	flags.SetOutput(output)
	var cfg config
	flags.StringVar(&cfg.SQLitePath, "sqlite-path", "data/multica-canonical.db", "Canonical SQLite database path")
	flags.StringVar(&cfg.UserID, "user-id", "", "fixed Auth user id for this stdio MCP process")
	if err := flags.Parse(arguments); err != nil {
		return config{}, err
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	cfg.SQLitePath = strings.TrimSpace(cfg.SQLitePath)
	cfg.UserID = strings.TrimSpace(cfg.UserID)
	if cfg.SQLitePath == "" || cfg.UserID == "" {
		return config{}, errors.New("--sqlite-path and --user-id are required")
	}
	return cfg, nil
}

func openCanonicalSQLite(ctx context.Context, path string) (*sql.DB, error) {
	dataSource := path
	if path != ":memory:" {
		separator := "?"
		if strings.Contains(path, "?") {
			separator = "&"
		}
		dataSource += separator + "_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dataSource)
	if err != nil {
		return nil, fmt.Errorf("open Canonical SQLite database: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(8)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA busy_timeout = 5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure Canonical SQLite database: %w", err)
	}
	return db, nil
}
