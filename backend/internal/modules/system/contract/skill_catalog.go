package contract

import (
	"context"
	"errors"
	"net/http"
	"time"
)

var (
	ErrInvalidSkill        = errors.New("invalid skill")
	ErrSkillAccessDenied   = errors.New("skill access denied")
	ErrSkillAlreadyExists  = errors.New("skill already exists")
	ErrSkillNotFound       = errors.New("skill not found")
	ErrSkillTransition     = errors.New("invalid skill transition")
	ErrSkillImportConflict = errors.New("Skill import conflict")
)

const (
	PermissionSkillRead    = "workspace.skill.read_published"
	PermissionSkillCreate  = "workspace.skill.create"
	PermissionSkillImport  = "workspace.skill.import"
	PermissionSkillVersion = "workspace.skill.version"
	PermissionSkillArchive = "workspace.skill.archive"
)

type SkillRevisionConflict struct{ CurrentRevision int64 }

func (e SkillRevisionConflict) Error() string { return "skill revision conflict" }

type SkillIdentity struct {
	WorkspaceID string
	ActorType   string
	ActorID     string
}

type SkillIdentityResolver func(*http.Request) (SkillIdentity, error)
type SkillMutationAuthorizer func(*http.Request) error
type SkillAccessAuthorizer func(context.Context, SkillIdentity, string) error
type SkillCreateExecutor interface {
	Execute(context.Context, string, ...any) error
	ExecuteResult(context.Context, string, ...any) (int64, error)
}

type SkillCreateBinding func(context.Context, SkillCreateExecutor) error
type SkillVisibilityPreflight func(context.Context, SkillIdentity, string, string) error
type SkillVisibilityBinder func(context.Context, SkillCreateExecutor, SkillIdentity, string, string) error
type SkillVisibilityResolver func(context.Context, string, string) (SkillVisibilityReference, error)
type SkillVisibilityLister func(context.Context, string) ([]SkillVisibilityReference, error)

type SkillVisibilityReference struct {
	WorkspaceID string
	SkillID     string
	VersionID   string
	Enabled     bool
	AgentIDs    []string
}

type SkillCatalogEntry struct {
	ID          string         `json:"id"`
	WorkspaceID string         `json:"workspace_id"`
	VersionID   string         `json:"version_id"`
	Version     string         `json:"version"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Config      map[string]any `json:"config"`
	Status      string         `json:"status"`
	Revision    int64          `json:"revision"`
	CreatedBy   string         `json:"created_by"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
	Archived    bool           `json:"archived"`
}

type SkillProvenance struct {
	OriginWorkspaceID string `json:"origin_workspace_id"`
	CreatedBy         string `json:"created_by"`
	CreatedAt         string `json:"created_at"`
}

type SkillAuditEntry struct {
	ID          string `json:"id"`
	VersionID   string `json:"version_id"`
	WorkspaceID string `json:"workspace_id"`
	ActorType   string `json:"actor_type"`
	ActorID     string `json:"actor_id"`
	Action      string `json:"action"`
	CreatedAt   string `json:"created_at"`
}

type SkillHistory struct {
	SkillID    string            `json:"skill_id"`
	Provenance SkillProvenance   `json:"provenance"`
	Audit      []SkillAuditEntry `json:"audit"`
}

type CreateSkillCatalogRequest struct {
	WorkspaceID string
	ActorType   string
	ActorID     string
	Name        string
	Description string
	Config      map[string]any
}

type SkillCatalogRepository interface {
	Create(context.Context, CreateSkillCatalogRequest, string, string, time.Time, SkillCreateBinding) (SkillCatalogEntry, error)
	CreateVersion(context.Context, SkillIdentity, string, UpdateSkillCatalogRequest, string, time.Time) (SkillCatalogEntry, error)
	TransitionVersion(context.Context, SkillIdentity, string, string, string, int64, time.Time) (SkillCatalogEntry, error)
	Archive(context.Context, SkillIdentity, string, int64, time.Time) error
	Restore(context.Context, SkillIdentity, string, int64, time.Time) (SkillCatalogEntry, error)
	Get(context.Context, SkillIdentity, string, string, bool) (SkillCatalogEntry, error)
	List(context.Context, SkillIdentity, bool) ([]SkillCatalogEntry, error)
	GetReferenced(context.Context, string, string) (SkillCatalogEntry, error)
	History(context.Context, SkillIdentity, string) (SkillHistory, error)
}

type UpdateSkillCatalogRequest struct {
	Name             *string
	Description      *string
	Config           map[string]any
	ConfigPresent    bool
	ExpectedRevision int64
	RequireManifest  bool
}

type SkillCatalogService interface {
	Create(context.Context, SkillIdentity, CreateSkillCatalogRequest) (SkillCatalogEntry, error)
	CreateVersion(context.Context, SkillIdentity, string, UpdateSkillCatalogRequest) (SkillCatalogEntry, error)
	TransitionVersion(context.Context, SkillIdentity, string, string, string, int64) (SkillCatalogEntry, error)
	Archive(context.Context, SkillIdentity, string, int64) error
	Restore(context.Context, SkillIdentity, string, int64) (SkillCatalogEntry, error)
	Get(context.Context, SkillIdentity, string, string) (SkillCatalogEntry, error)
	List(context.Context, SkillIdentity) ([]SkillCatalogEntry, error)
	History(context.Context, SkillIdentity, string) (SkillHistory, error)
}
