// dddgen:service-implementation MemberService; method bodies are user-owned.
package application

import (
	"context"

	"github.com/hvritual/workspace/internal/modules/auth/contract"
)

type MemberService struct{}

func NewMemberService() *MemberService { return &MemberService{} }

func (s *MemberService) ListMembers(ctx context.Context, request contract.ListMembersRequest) (contract.ListMembersResponse, error) {
	return contract.ListMembersResponse{}, contract.ErrMemberNotImplemented
}

func (s *MemberService) UpdateMemberRole(ctx context.Context, request contract.UpdateMemberRoleRequest) (contract.UpdateMemberRoleResponse, error) {
	return contract.UpdateMemberRoleResponse{}, contract.ErrMemberNotImplemented
}
func (s *MemberService) ProvisionWorkspaceOwner(ctx context.Context, request contract.ProvisionWorkspaceOwnerRequest) (contract.ProvisionWorkspaceOwnerResponse, error) {
	return contract.ProvisionWorkspaceOwnerResponse{}, contract.ErrMemberNotImplemented
}
