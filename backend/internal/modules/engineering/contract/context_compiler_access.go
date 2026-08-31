package contract

import "context"

// AuthorizedContextCompiler exposes ContextPack compilation through the same
// workspace actor boundary as other Engineering application operations.
type AuthorizedContextCompiler interface {
	CompileContext(context.Context, Actor, CompileContextRequest) (CompileContextResult, error)
}
