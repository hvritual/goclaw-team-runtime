package gateway

import (
	"errors"
	"fmt"
	"strings"

	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/teamcontrol"
)

const teamSessionPrefix = "user:"

var errUnauthenticatedPrincipal = errors.New("authenticated team principal is required")

func teamSessionID(userID string) string {
	return teamSessionPrefix + strings.TrimSpace(userID)
}

func (h *Handler) principalID(sessionID string) (string, error) {
	if h.teamSvc == nil {
		return sessionID, nil
	}
	userID := strings.TrimSpace(strings.TrimPrefix(sessionID, teamSessionPrefix))
	if userID == "" || userID == strings.TrimSpace(sessionID) {
		return "", errUnauthenticatedPrincipal
	}
	user, err := h.teamSvc.GetUser(userID)
	if err != nil {
		return "", fmt.Errorf("resolve authenticated user: %w", err)
	}
	if user.Status != teamcontrol.UserActive {
		return "", fmt.Errorf("%w: user is not active", errUnauthenticatedPrincipal)
	}
	return user.ID, nil
}

// actorForSession deliberately ignores request-supplied actor identities in
// team mode. Legacy single-user installations retain their existing behavior.
func (h *Handler) actorForSession(sessionID, requested string) (string, error) {
	if h.teamSvc != nil {
		return h.principalID(sessionID)
	}
	if actor := strings.TrimSpace(requested); actor != "" {
		return actor, nil
	}
	return sessionID, nil
}

func (h *Handler) authorizeProject(
	sessionID, projectID string,
	action teamcontrol.Action,
) (string, error) {
	userID, err := h.principalID(sessionID)
	if err != nil {
		return "", err
	}
	if h.teamSvc == nil {
		return userID, nil
	}
	if strings.TrimSpace(projectID) == "" {
		return "", errors.New("project_id is required")
	}
	if err := h.teamSvc.Authorize(userID, projectID, action); err != nil {
		return "", err
	}
	return userID, nil
}

func (h *Handler) authorizeChat(sessionID string, params map[string]interface{}) error {
	if h.teamSvc == nil {
		return nil
	}
	_, err := h.authorizeProject(
		sessionID,
		stringParam(params["project_id"]),
		teamcontrol.ActionProjectRead,
	)
	return err
}

func (h *Handler) chatIdentity(
	sessionID string,
	params map[string]interface{},
) (senderID, chatID string, metadata map[string]interface{}, err error) {
	metadata = projectMetadata(params)
	if h.teamSvc == nil {
		return sessionID, sessionID, metadata, nil
	}
	projectID := stringParam(params["project_id"])
	userID, err := h.authorizeProject(
		sessionID,
		projectID,
		teamcontrol.ActionProjectRead,
	)
	if err != nil {
		return "", "", nil, err
	}
	topicID := strings.TrimSpace(stringParam(params["topic_id"]))
	_, topicID, err = session.ProjectConversationKey(projectID, topicID)
	if err != nil {
		return "", "", nil, err
	}
	metadata["project_id"] = projectID
	metadata["topic_id"] = topicID
	metadata["principal_id"] = userID
	return userID,
		fmt.Sprintf("project:%s:topic:%s", projectID, topicID),
		metadata,
		nil
}
