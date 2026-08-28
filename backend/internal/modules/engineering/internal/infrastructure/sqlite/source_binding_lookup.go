package sqlite

import (
	"context"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) FindSourceBindingBySource(ctx context.Context, workspaceID, sourceType, locator string) (domain.SourceBinding, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,entity_id,source_type,source_locator,source_revision,observed_at,authority
		FROM engineering_source_bindings
		WHERE workspace_id=? AND source_type=? AND source_locator=?
		ORDER BY id LIMIT 2`, workspaceID, sourceType, locator)
	if err != nil {
		return domain.SourceBinding{}, fmt.Errorf("find engineering source binding: %w", err)
	}
	defer rows.Close()
	var values []domain.SourceBinding
	for rows.Next() {
		var id, entityID, storedSourceType, storedLocator, revision, observedAt, authority string
		if err := rows.Scan(&id, &entityID, &storedSourceType, &storedLocator, &revision, &observedAt, &authority); err != nil {
			return domain.SourceBinding{}, fmt.Errorf("scan engineering source binding lookup: %w", err)
		}
		value, err := buildSourceBinding(workspaceID, id, entityID, storedSourceType, storedLocator, revision, observedAt, authority)
		if err != nil {
			return domain.SourceBinding{}, err
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return domain.SourceBinding{}, fmt.Errorf("iterate engineering source binding lookup: %w", err)
	}
	switch len(values) {
	case 0:
		return domain.SourceBinding{}, domain.ErrNotFound
	case 1:
		return values[0], nil
	default:
		return domain.SourceBinding{}, domain.ErrConflict
	}
}
