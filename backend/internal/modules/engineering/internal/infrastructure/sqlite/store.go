package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

var ErrDatabaseRequired = errors.New("engineering sqlite database is required")

type Store struct {
	db      *sql.DB
	writeMu sync.Mutex
}

func New(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, ErrDatabaseRequired
	}
	return &Store{db: db}, nil
}

func (s *Store) PutEntity(ctx context.Context, value domain.EngineeringEntity) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT INTO engineering_entities(
		workspace_id,id,entity_type,name,status,owner_ref
	) VALUES(?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		entity_type=excluded.entity_type,
		name=excluded.name,
		status=excluded.status,
		owner_ref=excluded.owner_ref`,
		value.WorkspaceID(), value.ID(), string(value.Type()), value.Name(), string(value.Status()), value.OwnerRef(),
	)
	if err != nil {
		return fmt.Errorf("put engineering entity: %w", err)
	}
	return nil
}

func (s *Store) GetEntity(ctx context.Context, workspaceID, id string) (domain.EngineeringEntity, error) {
	var entityType, name, status, ownerRef string
	if err := s.db.QueryRowContext(ctx, `SELECT entity_type,name,status,owner_ref
		FROM engineering_entities WHERE workspace_id=? AND id=?`, workspaceID, id,
	).Scan(&entityType, &name, &status, &ownerRef); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.EngineeringEntity{}, domain.ErrNotFound
		}
		return domain.EngineeringEntity{}, fmt.Errorf("get engineering entity: %w", err)
	}
	value, err := domain.NewEngineeringEntity(id, workspaceID, domain.EntityType(entityType), name, domain.EntityStatus(status), ownerRef)
	if err != nil {
		return domain.EngineeringEntity{}, fmt.Errorf("rehydrate engineering entity: %w", err)
	}
	return value, nil
}

func (s *Store) ListEntities(ctx context.Context, workspaceID string) ([]domain.EngineeringEntity, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,entity_type,name,status,owner_ref
		FROM engineering_entities WHERE workspace_id=? ORDER BY id`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list engineering entities: %w", err)
	}
	defer rows.Close()
	var values []domain.EngineeringEntity
	for rows.Next() {
		var id, entityType, name, status, ownerRef string
		if err := rows.Scan(&id, &entityType, &name, &status, &ownerRef); err != nil {
			return nil, fmt.Errorf("scan engineering entity: %w", err)
		}
		value, err := domain.NewEngineeringEntity(id, workspaceID, domain.EntityType(entityType), name, domain.EntityStatus(status), ownerRef)
		if err != nil {
			return nil, fmt.Errorf("rehydrate engineering entity: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering entities: %w", err)
	}
	return values, nil
}

func (s *Store) PutSourceBinding(ctx context.Context, value domain.SourceBinding) error {
	provenance := value.Provenance()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT INTO engineering_source_bindings(
		workspace_id,id,entity_id,source_type,source_locator,source_revision,observed_at,authority
	) VALUES(?,?,?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		entity_id=excluded.entity_id,
		source_type=excluded.source_type,
		source_locator=excluded.source_locator,
		source_revision=excluded.source_revision,
		observed_at=excluded.observed_at,
		authority=excluded.authority`,
		value.WorkspaceID(), value.ID(), value.EntityID(), provenance.SourceType(), provenance.Locator(), provenance.Revision(),
		formatTime(provenance.ObservedAt()), string(value.Authority()),
	)
	if err != nil {
		return fmt.Errorf("put engineering source binding: %w", err)
	}
	return nil
}

func (s *Store) GetSourceBinding(ctx context.Context, workspaceID, id string) (domain.SourceBinding, error) {
	row := s.db.QueryRowContext(ctx, `SELECT entity_id,source_type,source_locator,source_revision,observed_at,authority
		FROM engineering_source_bindings WHERE workspace_id=? AND id=?`, workspaceID, id)
	value, err := scanSourceBinding(row, workspaceID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.SourceBinding{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.SourceBinding{}, fmt.Errorf("get engineering source binding: %w", err)
	}
	return value, nil
}

func (s *Store) ListSourceBindings(ctx context.Context, workspaceID, entityID string) ([]domain.SourceBinding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,entity_id,source_type,source_locator,source_revision,observed_at,authority
		FROM engineering_source_bindings WHERE workspace_id=? AND entity_id=? ORDER BY id`, workspaceID, entityID)
	if err != nil {
		return nil, fmt.Errorf("list engineering source bindings: %w", err)
	}
	defer rows.Close()
	var values []domain.SourceBinding
	for rows.Next() {
		var id string
		var entity, sourceType, locator, revision, observedAt, authority string
		if err := rows.Scan(&id, &entity, &sourceType, &locator, &revision, &observedAt, &authority); err != nil {
			return nil, fmt.Errorf("scan engineering source binding: %w", err)
		}
		value, err := buildSourceBinding(workspaceID, id, entity, sourceType, locator, revision, observedAt, authority)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering source bindings: %w", err)
	}
	return values, nil
}

func (s *Store) PutThreadEdge(ctx context.Context, value domain.ThreadEdge) error {
	from, to, provenance := value.From(), value.To(), value.Provenance()
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	_, err := s.db.ExecContext(ctx, `INSERT INTO engineering_thread_edges(
		workspace_id,id,from_kind,from_id,relation_type,to_kind,to_id,authority,
		source_type,source_locator,source_revision,observed_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		from_kind=excluded.from_kind,
		from_id=excluded.from_id,
		relation_type=excluded.relation_type,
		to_kind=excluded.to_kind,
		to_id=excluded.to_id,
		authority=excluded.authority,
		source_type=excluded.source_type,
		source_locator=excluded.source_locator,
		source_revision=excluded.source_revision,
		observed_at=excluded.observed_at`,
		value.WorkspaceID(), value.ID(), string(from.Kind()), from.ID(), string(value.Relation()), string(to.Kind()), to.ID(),
		string(value.Authority()), provenance.SourceType(), provenance.Locator(), provenance.Revision(), formatTime(provenance.ObservedAt()),
	)
	if err != nil {
		return fmt.Errorf("put engineering thread edge: %w", err)
	}
	return nil
}

func (s *Store) GetThreadEdge(ctx context.Context, workspaceID, id string) (domain.ThreadEdge, error) {
	row := s.db.QueryRowContext(ctx, `SELECT from_kind,from_id,relation_type,to_kind,to_id,authority,
		source_type,source_locator,source_revision,observed_at
		FROM engineering_thread_edges WHERE workspace_id=? AND id=?`, workspaceID, id)
	value, err := scanThreadEdge(row, workspaceID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.ThreadEdge{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("get engineering thread edge: %w", err)
	}
	return value, nil
}

func (s *Store) ListThreadEdges(ctx context.Context, workspaceID string, node domain.NodeRef) ([]domain.ThreadEdge, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,from_kind,from_id,relation_type,to_kind,to_id,authority,
		source_type,source_locator,source_revision,observed_at
		FROM engineering_thread_edges
		WHERE workspace_id=? AND ((from_kind=? AND from_id=?) OR (to_kind=? AND to_id=?))
		ORDER BY id`, workspaceID, string(node.Kind()), node.ID(), string(node.Kind()), node.ID())
	if err != nil {
		return nil, fmt.Errorf("list engineering thread edges: %w", err)
	}
	defer rows.Close()
	var values []domain.ThreadEdge
	for rows.Next() {
		var id string
		var fromKind, fromID, relation, toKind, toID, authority string
		var sourceType, locator, revision, observedAt string
		if err := rows.Scan(&id, &fromKind, &fromID, &relation, &toKind, &toID, &authority, &sourceType, &locator, &revision, &observedAt); err != nil {
			return nil, fmt.Errorf("scan engineering thread edge: %w", err)
		}
		value, err := buildThreadEdge(workspaceID, id, fromKind, fromID, relation, toKind, toID, authority, sourceType, locator, revision, observedAt)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate engineering thread edges: %w", err)
	}
	return values, nil
}

type scanner interface {
	Scan(...any) error
}

func scanSourceBinding(row scanner, workspaceID, id string) (domain.SourceBinding, error) {
	var entityID, sourceType, locator, revision, observedAt, authority string
	if err := row.Scan(&entityID, &sourceType, &locator, &revision, &observedAt, &authority); err != nil {
		return domain.SourceBinding{}, err
	}
	return buildSourceBinding(workspaceID, id, entityID, sourceType, locator, revision, observedAt, authority)
}

func buildSourceBinding(workspaceID, id, entityID, sourceType, locator, revision, observedAt, authority string) (domain.SourceBinding, error) {
	observed, err := parseTime(observedAt)
	if err != nil {
		return domain.SourceBinding{}, fmt.Errorf("parse source binding observed_at: %w", err)
	}
	provenance, err := domain.NewProvenance(sourceType, locator, revision, observed)
	if err != nil {
		return domain.SourceBinding{}, fmt.Errorf("rehydrate source binding provenance: %w", err)
	}
	value, err := domain.NewSourceBinding(id, workspaceID, entityID, provenance, domain.Authority(authority))
	if err != nil {
		return domain.SourceBinding{}, fmt.Errorf("rehydrate source binding: %w", err)
	}
	return value, nil
}

func scanThreadEdge(row scanner, workspaceID, id string) (domain.ThreadEdge, error) {
	var fromKind, fromID, relation, toKind, toID, authority string
	var sourceType, locator, revision, observedAt string
	if err := row.Scan(&fromKind, &fromID, &relation, &toKind, &toID, &authority, &sourceType, &locator, &revision, &observedAt); err != nil {
		return domain.ThreadEdge{}, err
	}
	return buildThreadEdge(workspaceID, id, fromKind, fromID, relation, toKind, toID, authority, sourceType, locator, revision, observedAt)
}

func buildThreadEdge(workspaceID, id, fromKind, fromID, relation, toKind, toID, authority, sourceType, locator, revision, observedAt string) (domain.ThreadEdge, error) {
	from, err := domain.NewNodeRef(domain.NodeKind(fromKind), fromID)
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("rehydrate thread edge source: %w", err)
	}
	to, err := domain.NewNodeRef(domain.NodeKind(toKind), toID)
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("rehydrate thread edge target: %w", err)
	}
	observed, err := parseTime(observedAt)
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("parse thread edge observed_at: %w", err)
	}
	provenance, err := domain.NewProvenance(sourceType, locator, revision, observed)
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("rehydrate thread edge provenance: %w", err)
	}
	value, err := domain.NewThreadEdge(id, workspaceID, from, domain.RelationType(relation), to, domain.Authority(authority), provenance)
	if err != nil {
		return domain.ThreadEdge{}, fmt.Errorf("rehydrate thread edge: %w", err)
	}
	return value, nil
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, value)
}
