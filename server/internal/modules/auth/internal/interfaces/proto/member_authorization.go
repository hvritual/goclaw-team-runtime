package proto

import (
	"context"

	authv1 "github.com/multica-ai/multica/server/gen/go/auth/v1"
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

func (s *MemberServer) AuthorizeCreateInvitation(ctx context.Context, request *authv1.CreateInvitationRequest) error {
	var input contract.Member_CreateInvitationRequest
	if err := decodeMemberProto(request, &input); err != nil {
		return err
	}
	authorizer, ok := s.service.(contract.InvitationCreationAuthorizer)
	if !ok {
		return memberTransportError(contract.ErrMemberNotImplemented)
	}
	if err := authorizer.AuthorizeCreateInvitation(ctx, input); err != nil {
		return memberTransportError(err)
	}
	return nil
}
