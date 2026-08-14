package http

import (
	"errors"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"github.com/hvritual/workspace/internal/modules/auth/contract"
)

type MemberListHandler struct {
	service  contract.MemberService
	identity func(*http.Request) (string, error)
}

func NewMemberListHandler(service contract.MemberService, identity func(*http.Request) (string, error)) *MemberListHandler {
	return &MemberListHandler{service: service, identity: identity}
}

func (h *MemberListHandler) Register(server *kratoshttp.Server) {
	server.Route("/").GET("/api/workspaces/{workspace_id}/members", h.list)
}

func (h *MemberListHandler) list(ctx kratoshttp.Context) error {
	userID, err := h.identity(ctx.Request())
	if err != nil {
		return writeError(ctx, http.StatusUnauthorized, "user not authenticated")
	}
	result, err := h.service.ListMembers(contract.WithMemberActor(ctx.Request().Context(), userID), contract.ListMembersRequest{WorkspaceId: ctx.Vars().Get("workspace_id")})
	if errors.Is(err, contract.ErrWorkspaceMembershipHidden) {
		return writeError(ctx, http.StatusNotFound, "workspace not found")
	}
	if err != nil {
		return writeError(ctx, http.StatusInternalServerError, "failed to list members")
	}
	members := make([]memberResponse, len(result.Members))
	for index, value := range result.Members {
		members[index] = memberResponse{
			ID: value.Id, WorkspaceID: value.WorkspaceId, UserID: value.UserId,
			Role: value.Role, CreatedAt: value.CreatedAt, Name: value.Name,
			Email: value.Email, AvatarURL: value.AvatarUrl,
		}
	}
	return ctx.JSON(http.StatusOK, members)
}

type memberResponse struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	UserID      string  `json:"user_id"`
	Role        string  `json:"role"`
	CreatedAt   string  `json:"created_at"`
	Name        string  `json:"name"`
	Email       string  `json:"email"`
	AvatarURL   *string `json:"avatar_url"`
}
