package space

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/hvritual/workspace/internal/modules/space/contract"
	"github.com/hvritual/workspace/internal/modules/space/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/space/internal/infrastructure/sqlite"
)

//go:embed internal/infrastructure/sqlite/migrations/*.sql
var sqliteMigrationFiles embed.FS

func SqliteMigrationFS() fs.FS   { return sqliteMigrationFiles }
func SqliteMigrationDir() string { return "internal/infrastructure/sqlite/migrations" }

func MigrateSqlite(ctx context.Context, db *sql.DB) error {
	if db == nil {
		return errors.New("space sqlite database is required")
	}
	paths, err := fs.Glob(sqliteMigrationFiles, SqliteMigrationDir()+"/*.up.sql")
	if err != nil {
		return fmt.Errorf("list Space SQLite migrations: %w", err)
	}
	sort.Strings(paths)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Space SQLite migration: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS space_schema_migrations(version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create Space SQLite migration catalog: %w", err)
	}
	for _, path := range paths {
		version := path[len(SqliteMigrationDir())+1:]
		var applied string
		err := tx.QueryRowContext(ctx, `SELECT version FROM space_schema_migrations WHERE version=?`, version).Scan(&applied)
		if err == nil {
			continue
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("check Space SQLite migration %s: %w", version, err)
		}
		migration, err := fs.ReadFile(sqliteMigrationFiles, path)
		if err != nil {
			return fmt.Errorf("read Space SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(migration)); err != nil {
			return fmt.Errorf("apply Space SQLite migration %s: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO space_schema_migrations(version) VALUES(?)`, version); err != nil {
			return fmt.Errorf("record Space SQLite migration %s: %w", version, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Space SQLite migrations: %w", err)
	}
	return nil
}

type SQLiteAttachmentConfig struct {
	DB                     *sql.DB
	StorageRoot            string
	Relations              contract.AttachmentRelations
	HTTPIdentity           contract.HTTPIdentityResolver
	HTTPUserIdentity       contract.HTTPUserResolver
	HTTPMutationAuthorizer contract.HTTPMutationAuthorizer
	WorkspaceMemberships   contract.WorkspaceMembershipReader
	Events                 contract.WorkspaceEventPublisher
	NewID                  func() (string, error)
	Now                    func() time.Time
	HTTPEnabled            bool
}

func NewWithSQLiteAttachments(config SQLiteAttachmentConfig) (*Module, error) {
	repository, err := persistence.NewAttachmentRepository(config.DB)
	if err != nil {
		return nil, fmt.Errorf("configure Space attachment repository: %w", err)
	}
	objects, err := persistence.NewObjectStore(config.StorageRoot)
	if err != nil {
		return nil, fmt.Errorf("configure Space attachment object store: %w", err)
	}
	service, err := application.NewAttachmentService(repository, objects, config.Relations)
	if err != nil {
		return nil, fmt.Errorf("configure Space attachment application: %w", err)
	}
	if config.NewID != nil {
		service.SetIDGenerator(config.NewID)
	}
	if config.Now != nil {
		service.SetClock(config.Now)
	}
	if err := service.Reconcile(context.Background()); err != nil {
		return nil, fmt.Errorf("reconcile Space attachment objects: %w", err)
	}
	var attachmentService contract.AttachmentService = service
	if config.Events != nil {
		attachmentService = publishingAttachmentService{AttachmentService: service, events: config.Events}
	}
	module := New()
	if config.HTTPEnabled {
		module.extensions = append(module.extensions, newAttachmentExtension(attachmentService, config.HTTPIdentity, config.HTTPUserIdentity, config.HTTPMutationAuthorizer, config.WorkspaceMemberships))
	}
	module.attachments = attachmentService
	return module, nil
}

func (m *Module) Attachments() contract.AttachmentService { return m.attachments }
