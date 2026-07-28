package gateway

import (
	"crypto/sha256"
	"fmt"

	"github.com/smallnest/goclaw/session"
	"github.com/smallnest/goclaw/teamcontrol"
)

const (
	defaultChatHistoryLimit = 100
	maxChatHistoryLimit     = 200
)

type chatHistoryMessage struct {
	ID        string `json:"id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

func (h *Handler) rpcChatHistory(
	sessionID string,
	params map[string]interface{},
) (interface{}, error) {
	if h.teamSvc == nil {
		return nil, fmt.Errorf(
			"%w: chat.history requires team mode",
			teamcontrol.ErrForbidden,
		)
	}
	if h.sessionMgr == nil {
		return nil, fmt.Errorf("chat history service is not enabled")
	}
	projectID := stringParam(params["project_id"])
	if _, err := h.authorizeProject(
		sessionID,
		projectID,
		teamcontrol.ActionProjectRead,
	); err != nil {
		return nil, err
	}
	conversation, topicID, found, err := h.sessionMgr.GetExistingProjectConversation(
		projectID,
		stringParam(params["topic_id"]),
	)
	if err != nil {
		return nil, fmt.Errorf("load project conversation: %w", err)
	}

	var history []session.Message
	if found {
		history = conversation.GetHistory(0)
	}
	visible := make([]chatHistoryMessage, 0, len(history))
	for index, message := range history {
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		digest := sha256.Sum256([]byte(fmt.Sprintf(
			"%d\x00%s\x00%s\x00%s",
			index,
			message.Role,
			message.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
			message.Content,
		)))
		visible = append(visible, chatHistoryMessage{
			ID:        fmt.Sprintf("history-%x", digest[:12]),
			Role:      message.Role,
			Content:   message.Content,
			Timestamp: message.Timestamp.UTC().Format("2006-01-02T15:04:05.999999999Z07:00"),
		})
	}

	limit := intParam(params["limit"], defaultChatHistoryLimit)
	if limit <= 0 {
		limit = defaultChatHistoryLimit
	}
	if limit > maxChatHistoryLimit {
		limit = maxChatHistoryLimit
	}
	total := len(visible)
	if len(visible) > limit {
		visible = visible[len(visible)-limit:]
	}
	return map[string]interface{}{
		"project_id": projectID,
		"topic_id":   topicID,
		"messages":   visible,
		"has_more":   total > len(visible),
	}, nil
}
