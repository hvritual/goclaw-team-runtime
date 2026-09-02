package sqlite

import "github.com/hvritual/workspace/internal/modules/engineering/internal/domain"

var _ domain.ExecutionEvidenceRepository = (*Store)(nil)
