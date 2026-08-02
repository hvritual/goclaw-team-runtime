package auth

import (
	"github.com/multica-ai/multica/server/internal/modules/auth/contract"
	"github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/local"
	protoadapter "github.com/multica-ai/multica/server/internal/modules/auth/internal/interfaces/proto"
)

// NewMemberExtensionWithService assembles the generated local and transport
// adapters around an explicit application contract.
func NewMemberExtensionWithService(service contract.MemberService) *MemberExtension {
	client := local.NewMember(service)
	return &MemberExtension{local: client, server: protoadapter.NewMemberServer(client)}
}
