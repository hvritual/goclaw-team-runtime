package domain

import "context"

// ThreadEdgeDeleteRepository is deliberately separate from the Phase-1 core
// Repository port so explicit unlink capability is additive and no cascade
// semantics are implied for existing repositories.
type ThreadEdgeDeleteRepository interface {
	DeleteThreadEdge(context.Context, string, string) error
}
