package workspace

import (
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
)

type GovernanceService = application.GovernanceService
type GovernanceRequest = application.GovernanceRequest
type OutboxDraft = application.OutboxDraft
type PreparedGovernanceMutation = application.PreparedGovernanceMutation

type SQLiteGovernance = persistence.GovernanceRepository
type SQLiteGovernanceOption = persistence.GovernanceRepositoryOption
type SQLiteGovernancePhase = persistence.GovernancePhase
type SQLiteDomainMutation = persistence.DomainMutation

func NewGovernanceService() GovernanceService {
	return application.NewGovernanceService()
}

func NewSQLiteGovernance(config SqlitePersistenceConfig, options ...SQLiteGovernanceOption) (*SQLiteGovernance, error) {
	return persistence.NewGovernanceRepository(config, options...)
}
