package contract

import (
	"context"
	"strings"
)

type workspaceActorContextKey struct{}

// WorkspaceActor identifies the authenticated caller without exposing Auth
// transport or persistence types to Workspace application code.
type WorkspaceActor struct {
	Type string
	ID   string
}

// WithWorkspaceActor attaches a validated caller projection at a transport or
// composition boundary.
func WithWorkspaceActor(ctx context.Context, actorType, actorID string) context.Context {
	actor := WorkspaceActor{Type: strings.TrimSpace(actorType), ID: strings.TrimSpace(actorID)}
	return context.WithValue(ctx, workspaceActorContextKey{}, actor)
}

// WorkspaceActorFromContext returns the caller projection when both parts are
// present. Membership is still verified through WorkspaceActorReader.
func WorkspaceActorFromContext(ctx context.Context) (WorkspaceActor, bool) {
	actor, ok := ctx.Value(workspaceActorContextKey{}).(WorkspaceActor)
	return actor, ok && actor.Type != "" && actor.ID != ""
}

// WorkspaceAccessAuthorizer is the consumer-owned authorization seam for
// Workspace use cases. Implementations must resolve the caller from context.
type WorkspaceAccessAuthorizer interface {
	AuthorizeWorkspace(context.Context, string, string) error
}

// WorkspaceActorReader is the consumer-owned seam for Auth-owned actor
// identity. Workspace never reads Auth persistence directly.
type WorkspaceActorReader interface {
	ActorBelongsToWorkspace(context.Context, string, string, string) (bool, error)
}

// WorkspaceAssetReader validates Space-owned Asset references without exposing
// Space storage to Workspace.
type WorkspaceAssetReader interface {
	AssetBelongsToWorkspace(context.Context, string, string) (bool, error)
}

// SkillReferenceReader validates System-owned Skill and version references.
type SkillReferenceReader interface {
	SkillReferenceExists(context.Context, string, *string) (bool, error)
}
