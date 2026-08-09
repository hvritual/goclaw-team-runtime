package workspace

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hvritual/workspace/internal/modules/workspace/contract"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/application"
	persistence "github.com/hvritual/workspace/internal/modules/workspace/internal/infrastructure/sqlite"
	"github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/local"
	protoadapter "github.com/hvritual/workspace/internal/modules/workspace/internal/interfaces/proto"
)

// NewWithSqliteWorkspaceChain explicitly enables the migrated
// Todo -> Issue -> Knowledge -> Requirement -> Setting chain in addition to
// Project and Relationship. The default module remains generated stubs.
func NewWithSqliteWorkspaceChain(config SqlitePersistenceConfig, dependencies WorkspaceServiceDependencies) (*Module, error) {
	if dependencies.Assets == nil {
		return nil, errors.New("workspace asset reader is required")
	}
	if dependencies.Skills == nil {
		return nil, errors.New("Skill reference reader is required")
	}
	module, err := NewWithSqliteWorkspaceServices(config, dependencies)
	if err != nil {
		return nil, err
	}

	projects, err := persistence.NewProjectRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Project SQLite persistence: %w", err)
	}
	issues, err := persistence.NewIssueRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Issue SQLite persistence: %w", err)
	}
	todos, err := persistence.NewTodoRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Todo SQLite persistence: %w", err)
	}
	knowledge, err := persistence.NewKnowledgeRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Knowledge SQLite persistence: %w", err)
	}
	requirements, err := persistence.NewRequirementRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Requirement SQLite persistence: %w", err)
	}
	settings, err := persistence.NewSettingRepository(config)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Setting SQLite persistence: %w", err)
	}

	newID := func(generator func(context.Context) (string, error)) application.ProjectIDGenerator {
		if generator == nil {
			return func(context.Context) (string, error) { return uuid.NewString(), nil }
		}
		return application.ProjectIDGenerator(generator)
	}
	now := application.Clock(dependencies.Now)
	if now == nil {
		now = time.Now
	}

	todoService, err := application.NewTodoUseCase(todos, projects, issues, dependencies.Authorizer, dependencies.Actors, newID(dependencies.NewTodoID), now)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Todo application: %w", err)
	}
	issueService, err := application.NewIssueUseCase(
		issues, projects, dependencies.Authorizer, dependencies.Actors, dependencies.Assets,
		newID(dependencies.NewIssueID), now,
	)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Issue application: %w", err)
	}
	knowledgeService, err := application.NewKnowledgeUseCase(knowledge, dependencies.Authorizer, dependencies.Assets, newID(dependencies.NewKnowledgeID), now)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Knowledge application: %w", err)
	}
	requirementService, err := application.NewRequirementUseCase(requirements, projects, issues, dependencies.Authorizer, newID(dependencies.NewRequirementID), newID(dependencies.NewRequirementVersionID), now)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Requirement application: %w", err)
	}
	settingService, err := application.NewSettingUseCase(settings, dependencies.Authorizer, dependencies.Actors, dependencies.Skills, now)
	if err != nil {
		return nil, fmt.Errorf("configure Workspace Setting application: %w", err)
	}

	replacements := map[string]bool{
		"todo": false, "issue": false, "knowledge": false, "requirement": false, "setting": false,
	}
	for index, extension := range module.extensions {
		switch extension.(type) {
		case *TodoExtension:
			module.extensions[index] = newTodoExtension(todoService)
			replacements["todo"] = true
		case *IssueExtension:
			module.extensions[index] = newIssueExtension(issueService)
			replacements["issue"] = true
		case *KnowledgeExtension:
			module.extensions[index] = newKnowledgeExtension(knowledgeService)
			replacements["knowledge"] = true
		case *RequirementExtension:
			module.extensions[index] = newRequirementExtension(requirementService)
			replacements["requirement"] = true
		case *SettingExtension:
			module.extensions[index] = newSettingExtension(settingService)
			replacements["setting"] = true
		}
	}
	for name, replaced := range replacements {
		if !replaced {
			return nil, fmt.Errorf("Workspace generated extension missing: %s", name)
		}
	}
	return module, nil
}

func newTodoExtension(service contract.TodoService) *TodoExtension {
	client := local.NewTodo(service)
	return &TodoExtension{local: client, server: protoadapter.NewTodoServer(client)}
}

func newIssueExtension(service contract.IssueService) *IssueExtension {
	client := local.NewIssue(service)
	return &IssueExtension{local: client, server: protoadapter.NewIssueServer(client)}
}

func newKnowledgeExtension(service contract.KnowledgeService) *KnowledgeExtension {
	client := local.NewKnowledge(service)
	return &KnowledgeExtension{local: client, server: protoadapter.NewKnowledgeServer(client)}
}

func newRequirementExtension(service contract.RequirementService) *RequirementExtension {
	client := local.NewRequirement(service)
	return &RequirementExtension{local: client, server: protoadapter.NewRequirementServer(client)}
}

func newSettingExtension(service contract.SettingService) *SettingExtension {
	client := local.NewSetting(service)
	return &SettingExtension{local: client, server: protoadapter.NewSettingServer(client)}
}
