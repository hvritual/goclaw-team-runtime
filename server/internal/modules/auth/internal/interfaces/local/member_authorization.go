package local

import (
	"context"

	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
)

func (c *MemberClient) AuthorizeCreateInvitation(
	ctx context.Context,
	request contract.Member_CreateInvitationRequest,
) error {
	authorizer, ok := c.service.(contract.InvitationCreationAuthorizer)
	if !ok {
		return contract.ErrMemberNotImplemented
	}
	return authorizer.AuthorizeCreateInvitation(ctx, request)
}

var _ contract.InvitationCreationAuthorizer = (*MemberClient)(nil)
