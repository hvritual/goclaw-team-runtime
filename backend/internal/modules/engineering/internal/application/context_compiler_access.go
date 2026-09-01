package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/engineering/contract"
)

// AuthorizedContextCompiler binds the deterministic compiler to the existing
// Engineering workspace authorization boundary. Compilation persists an
// immutable ContextPack, so it requires owner/admin write authority.
type AuthorizedContextCompiler struct {
	authorizer *Service
	compiler   contract.ContextCompiler
}

func NewAuthorizedContextCompiler(authorizer *Service, compiler contract.ContextCompiler) (*AuthorizedContextCompiler, error) {
	if authorizer == nil || compiler == nil {
		return nil, contract.ErrUnavailable
	}
	return &AuthorizedContextCompiler{authorizer: authorizer, compiler: compiler}, nil
}

func (s *AuthorizedContextCompiler) CompileContext(ctx context.Context, actor contract.Actor, request contract.CompileContextRequest) (contract.CompileContextResult, error) {
	if err := s.authorizer.authorize(ctx, actor, request.WorkspaceID, true); err != nil {
		return contract.CompileContextResult{}, err
	}
	return s.compiler.Compile(ctx, request)
}

var _ contract.AuthorizedContextCompiler = (*AuthorizedContextCompiler)(nil)
