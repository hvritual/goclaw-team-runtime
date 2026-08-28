package sqlite

import (
	"context"
	"fmt"

	"github.com/hvritual/workspace/internal/modules/engineering/internal/domain"
)

func (s *Store) DeleteThreadEdge(ctx context.Context, workspaceID, id string) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	result, err := s.db.ExecContext(ctx, `DELETE FROM engineering_thread_edges WHERE workspace_id=? AND id=?`, workspaceID, id)
	if err != nil {
		return fmt.Errorf("delete engineering thread edge: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count deleted engineering thread edges: %w", err)
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
