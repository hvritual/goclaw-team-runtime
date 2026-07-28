package gateway

import (
	"strings"

	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/teamcontrol"
)

type projectBroadcastScope struct {
	ProjectID string
	TopicID   string
}

// resolveProjectBroadcastScope returns the authoritative project boundary for
// a WebSocket notification. Gateway messages carry the boundary in their
// server-generated chat route; metadata may repeat it, but may never redirect
// an event to a different project or topic. Other channels have no scoped
// route, so both metadata fields are required.
func resolveProjectBroadcastScope(
	channel, chatID string,
	metadata interface{},
) (projectBroadcastScope, bool) {
	metadataProject := metadataProjectID(metadata)
	metadataTopic := metadataTopicID(metadata)

	if channel == "gateway" {
		scope, ok := parseGatewayProjectRoute(chatID)
		if !ok {
			return projectBroadcastScope{}, false
		}
		if metadataProject != "" && metadataProject != scope.ProjectID {
			return projectBroadcastScope{}, false
		}
		if metadataTopic != "" && metadataTopic != scope.TopicID {
			return projectBroadcastScope{}, false
		}
		return scope, true
	}

	if metadataProject == "" || metadataTopic == "" {
		return projectBroadcastScope{}, false
	}
	if _, normalizedTopic, err := session.ProjectConversationKey(
		metadataProject,
		metadataTopic,
	); err != nil || normalizedTopic != metadataTopic {
		return projectBroadcastScope{}, false
	}
	return projectBroadcastScope{
		ProjectID: metadataProject,
		TopicID:   metadataTopic,
	}, true
}

func parseGatewayProjectRoute(chatID string) (projectBroadcastScope, bool) {
	const prefix = "project:"
	const separator = ":topic:"

	chatID = strings.TrimSpace(chatID)
	if !strings.HasPrefix(chatID, prefix) {
		return projectBroadcastScope{}, false
	}
	remainder := strings.TrimPrefix(chatID, prefix)
	index := strings.Index(remainder, separator)
	if index <= 0 {
		return projectBroadcastScope{}, false
	}
	projectID := strings.TrimSpace(remainder[:index])
	topicID := strings.TrimSpace(remainder[index+len(separator):])
	if projectID == "" || topicID == "" {
		return projectBroadcastScope{}, false
	}
	if _, normalizedTopic, err := session.ProjectConversationKey(
		projectID,
		topicID,
	); err != nil || normalizedTopic != topicID {
		return projectBroadcastScope{}, false
	}
	return projectBroadcastScope{
		ProjectID: projectID,
		TopicID:   topicID,
	}, true
}

// projectBroadcastAllowed is the shared server-side security boundary used by
// both outbound messages and streaming chat events. In Team mode the legacy
// session match is never sufficient: delivery requires a complete validated
// scope, an authenticated principal, and current project read authorization.
func projectBroadcastAllowed(
	service *teamcontrol.Service,
	conn *Connection,
	legacySessionMatch bool,
	scope projectBroadcastScope,
	scopeOK bool,
) bool {
	if service == nil {
		return legacySessionMatch
	}
	if !scopeOK || conn == nil || strings.TrimSpace(conn.PrincipalID) == "" {
		return false
	}
	if strings.TrimSpace(scope.ProjectID) == "" ||
		strings.TrimSpace(scope.TopicID) == "" {
		return false
	}
	return service.Authorize(
		conn.PrincipalID,
		scope.ProjectID,
		teamcontrol.ActionProjectRead,
	) == nil
}
