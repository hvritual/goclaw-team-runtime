package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/multica-ai/multica/server/internal/knowledge"
	knowledgeSqlite "github.com/multica-ai/multica/server/internal/knowledge/adapter/sqlite"
	"github.com/multica-ai/multica/server/internal/knowledge/outbox"
	"github.com/multica-ai/multica/server/internal/util"
	db "github.com/multica-ai/multica/server/pkg/db/generated"
)

var errProductionKnowledgeDisabled = errors.New("knowledge capability disabled")

type knowledgeRuntime struct {
	enabled     bool
	store       *knowledgeSqlite.Store
	service     *knowledge.Service
	unavailable error
	dispatcher  *outbox.Dispatcher
	cancel      context.CancelFunc
	done        chan struct{}
	dispatchMu  sync.RWMutex
	dispatchErr string
	pool        *pgxpool.Pool
}

func openKnowledgeRuntime(pool *pgxpool.Pool, queries *db.Queries) *knowledgeRuntime {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("MULTICA_KNOWLEDGE_ENABLED")), "false") {
		return &knowledgeRuntime{unavailable: errProductionKnowledgeDisabled}
	}
	path := strings.TrimSpace(os.Getenv("MULTICA_KNOWLEDGE_SQLITE_PATH"))
	if path == "" {
		path = filepath.Join("data", "multica-knowledge.db")
	}
	store, err := knowledgeSqlite.Open(path)
	if err != nil {
		slog.Warn("knowledge store unavailable; core domains remain active", "error", err)
		return &knowledgeRuntime{
			enabled: true, unavailable: fmt.Errorf("open knowledge store: %w", err), pool: pool,
		}
	}
	runtime := &knowledgeRuntime{
		enabled: true,
		store:   store,
		service: knowledge.NewService(
			store,
			knowledge.DefaultPromotionPolicy{},
			postgresProjectValidator{queries: queries},
		),
		pool: pool,
	}
	if pool != nil {
		runtime.dispatcher = outbox.NewDispatcher(&postgresEvidenceOutbox{pool: pool}, runtime.service)
		runtime.startDispatcher()
	}
	return runtime
}

func (runtime *knowledgeRuntime) Close() error {
	if runtime == nil || runtime.store == nil {
		return nil
	}
	if runtime.cancel != nil {
		runtime.cancel()
		<-runtime.done
	}
	return runtime.store.Close()
}

func (runtime *knowledgeRuntime) startDispatcher() {
	ctx, cancel := context.WithCancel(context.Background())
	runtime.cancel = cancel
	runtime.done = make(chan struct{})
	go func() {
		defer close(runtime.done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			_, err := runtime.dispatcher.Drain(ctx, 50)
			runtime.dispatchMu.Lock()
			runtime.dispatchErr = ""
			if err != nil {
				runtime.dispatchErr = err.Error()
			}
			runtime.dispatchMu.Unlock()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (runtime *knowledgeRuntime) KnowledgeHealth(
	ctx context.Context,
	workspaceID string,
) (map[string]any, error) {
	if runtime.pool == nil {
		return map[string]any{}, nil
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return nil, err
	}
	var pending, failed int64
	var lastDelivered pgtype.Timestamptz
	if err := runtime.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE delivered_at IS NULL),
			COUNT(*) FILTER (WHERE delivered_at IS NULL AND attempts > 0),
			MAX(delivered_at)
		FROM knowledge_evidence_outbox
		WHERE workspace_id = $1`, workspaceUUID,
	).Scan(&pending, &failed, &lastDelivered); err != nil {
		return nil, err
	}
	runtime.dispatchMu.RLock()
	lastError := runtime.dispatchErr
	runtime.dispatchMu.RUnlock()
	return map[string]any{
		"pending": pending, "failed": failed,
		"last_delivered_at": util.TimestampToPtr(lastDelivered),
		"last_error":        nullableKnowledgeHealth(lastError),
	}, nil
}

func nullableKnowledgeHealth(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

type postgresProjectValidator struct {
	queries *db.Queries
}

func (validator postgresProjectValidator) ValidateProject(
	ctx context.Context,
	workspaceID string,
	projectID string,
) error {
	if validator.queries == nil {
		return knowledge.ErrProjectValidator
	}
	workspaceUUID, err := util.ParseUUID(workspaceID)
	if err != nil {
		return knowledge.ErrProjectScope
	}
	projectUUID, err := util.ParseUUID(projectID)
	if err != nil {
		return knowledge.ErrProjectScope
	}
	if _, err := validator.queries.GetProjectInWorkspace(ctx, db.GetProjectInWorkspaceParams{
		ID: projectUUID, WorkspaceID: workspaceUUID,
	}); err != nil {
		return knowledge.ErrProjectScope
	}
	return nil
}
