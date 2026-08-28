package sqlite

import (
	"context"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) ApplySourceProjection(ctx context.Context, projection domain.SourceProjection) error {
	entity := projection.Entity()
	binding := projection.Binding()
	provenance := binding.Provenance()

	s.writeMu.Lock()
	defer s.writeMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin engineering source projection: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_entities(
		workspace_id,id,entity_type,name,status,owner_ref
	) VALUES(?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		entity_type=excluded.entity_type,
		name=excluded.name,
		status=excluded.status,
		owner_ref=excluded.owner_ref`,
		entity.WorkspaceID(), entity.ID(), string(entity.Type()), entity.Name(), string(entity.Status()), entity.OwnerRef(),
	); err != nil {
		return fmt.Errorf("project engineering entity: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_source_bindings(
		workspace_id,id,entity_id,source_type,source_locator,source_revision,observed_at,authority
	) VALUES(?,?,?,?,?,?,?,?)
	ON CONFLICT(workspace_id,id) DO UPDATE SET
		entity_id=excluded.entity_id,
		source_type=excluded.source_type,
		source_locator=excluded.source_locator,
		source_revision=excluded.source_revision,
		observed_at=excluded.observed_at,
		authority=excluded.authority`,
		binding.WorkspaceID(), binding.ID(), binding.EntityID(), provenance.SourceType(), provenance.Locator(), provenance.Revision(),
		formatTime(provenance.ObservedAt()), string(binding.Authority()),
	); err != nil {
		return fmt.Errorf("project engineering source binding: %w", err)
	}

	for _, id := range projection.DeleteEdgeIDs() {
		if _, err := tx.ExecContext(ctx, `DELETE FROM engineering_thread_edges WHERE workspace_id=? AND id=?`, entity.WorkspaceID(), id); err != nil {
			return fmt.Errorf("delete stale engineering source edge %q: %w", id, err)
		}
	}
	for _, edge := range projection.UpsertEdges() {
		from, to, edgeProvenance := edge.From(), edge.To(), edge.Provenance()
		if _, err := tx.ExecContext(ctx, `INSERT INTO engineering_thread_edges(
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
			edge.WorkspaceID(), edge.ID(), string(from.Kind()), from.ID(), string(edge.Relation()), string(to.Kind()), to.ID(),
			string(edge.Authority()), edgeProvenance.SourceType(), edgeProvenance.Locator(), edgeProvenance.Revision(), formatTime(edgeProvenance.ObservedAt()),
		); err != nil {
			return fmt.Errorf("project engineering source edge %q: %w", edge.ID(), err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit engineering source projection: %w", err)
	}
	return nil
}
